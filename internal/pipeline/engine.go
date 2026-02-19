package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/epmk/vcoding/internal/config"
	"github.com/epmk/vcoding/internal/executor"
	vlog "github.com/epmk/vcoding/internal/log"
	"github.com/epmk/vcoding/internal/run"
	"github.com/epmk/vcoding/internal/types"
)

// Engine orchestrates pipeline step execution.
type Engine struct {
	Config    *config.Config
	Pipeline  *Pipeline
	Executors map[string]executor.Executor
	Run       *run.Run
	Display   *Display
}

// stepDisplayModel returns a human-readable label for the step's executor/model,
// suitable for the terminal output model column.
// Security: shell steps always return "shell" to avoid exposing raw commands.
func (e *Engine) stepDisplayModel(step types.Step) string {
	switch {
	case step.Type == "github-pr":
		return "github-pr"
	case step.Executor == "claude-code":
		return "claude-code"
	case step.Executor == "shell":
		// Never expose raw shell commands; they may contain secrets or sensitive paths.
		return "shell"
	case step.Executor == "api":
		model := e.resolveModel(step.Model)
		if model == "" {
			return "api"
		}
		return model
	case step.Executor != "":
		return step.Executor
	default:
		return "—"
	}
}

// Execute runs all steps in sequence.
func (e *Engine) Execute(ctx context.Context, pipelineCtx *Context) error {
	startTime := time.Now()
	var prURL string

	for _, step := range e.Pipeline.Steps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		displayModel := e.stepDisplayModel(step)
		e.Display.StepStart(step.Name, displayModel)
		stepStart := time.Now()

		var stepErr error
		var detail string
		var cost float64

		switch {
		case step.Type == "github-pr":
			detail, stepErr = e.runPRStep(ctx, step)
			if stepErr == nil {
				prURL = detail
			}

		case step.Executor == "":
			stepErr = fmt.Errorf("step %q has no executor", step.Name)

		default:
			detail, cost, stepErr = e.runExecutorStep(ctx, step, pipelineCtx)
		}

		duration := time.Since(stepStart)

		if stepErr != nil {
			e.Display.StepFailed(step.Name, displayModel, stepErr)
			if err := e.Run.Fail(stepErr.Error()); err != nil {
				vlog.Error("failed to update run meta", "err", err)
			}
			e.Display.Failed(stepErr)
			return fmt.Errorf("step %q failed: %w", step.Name, stepErr)
		}

		sr := run.StepResult{
			Name:       step.Name,
			Status:     "completed",
			Cost:       cost,
			DurationMS: duration.Milliseconds(),
		}
		if err := e.Run.AddStepResult(sr); err != nil {
			vlog.Warn("failed to save step result", "step", step.Name, "err", err)
		}

		e.Display.StepDone(step.Name, displayModel, detail, cost, duration)
	}

	if err := e.Run.Complete(); err != nil {
		vlog.Warn("failed to mark run complete", "err", err)
	}

	e.Display.Summary(e.Run.Meta.TotalCost, time.Since(startTime), prURL)
	return nil
}

// resolveModel replaces role placeholders ($planner, $reviewer, $editor, $auditor)
// with the corresponding model ID from config. If the model string is not a
// placeholder, it is returned unchanged.
func (e *Engine) resolveModel(model string) string {
	switch strings.ToLower(model) {
	case "$planner":
		return e.Config.Roles.Planner
	case "$reviewer":
		return e.Config.Roles.Reviewer
	case "$editor":
		return e.Config.Roles.Editor
	case "$auditor":
		return e.Config.Roles.Auditor
	}
	return model
}

func (e *Engine) runExecutorStep(ctx context.Context, step types.Step, pipelineCtx *Context) (detail string, cost float64, err error) {
	step.Model = e.resolveModel(step.Model)

	exec, ok := e.Executors[step.Executor]
	if !ok {
		return "", 0, fmt.Errorf("unknown executor %q", step.Executor)
	}

	// Resolve prompt for API steps
	var systemPrompt string
	if step.PromptTemplate != "" {
		// Prompt is pre-loaded into APIExecutor; no need to re-resolve here
	}
	_ = systemPrompt

	inputFiles, err := pipelineCtx.ResolveInput(step.Input)
	if err != nil {
		return "", 0, err
	}

	// Apply token budget truncation for API steps
	if step.Executor == "api" && e.Config.MaxContextTokens > 0 {
		systemPrompt, _ := resolvePromptForBudget(e, step)
		inputFiles = TruncateToTokenBudget(inputFiles, systemPrompt, e.Config.MaxContextTokens)
	}

	req := &executor.Request{
		Step:       step,
		RunDir:     e.Run.Dir,
		InputFiles: inputFiles,
	}

	result, err := exec.Execute(ctx, req)
	if err != nil {
		return "", 0, err
	}

	// Save output file to run directory
	if step.Output != "" && result.Output != "" {
		if writeErr := e.Run.WriteFile(step.Output, result.Output); writeErr != nil {
			vlog.Warn("failed to write output file", "file", step.Output, "err", writeErr)
		}
		detail = step.Output
	} else if step.Output == "" {
		detail = fmt.Sprintf("%.0fs", result.Duration.Seconds())
	}

	return detail, result.Cost, nil
}

func (e *Engine) runPRStep(ctx context.Context, step types.Step) (prURL string, err error) {
	// Generate PR body via body_template if specified
	if step.BodyTemplate != "" {
		if genErr := e.generatePRBody(ctx, step); genErr != nil {
			vlog.Warn("failed to generate PR body from template", "template", step.BodyTemplate, "err", genErr)
		}
	}

	exec, ok := e.Executors["github-pr"]
	if !ok {
		return "", fmt.Errorf("github-pr executor not registered")
	}

	inputFiles := map[string]string{}
	if step.TitleFrom != "" {
		if content, readErr := e.Run.ReadFile(step.TitleFrom); readErr == nil {
			inputFiles[step.TitleFrom] = content
		}
	}
	// Prefer generated PR.md; fall back to PLAN.md
	if content, readErr := e.Run.ReadFile("PR.md"); readErr == nil {
		inputFiles["PR.md"] = content
	} else if content, readErr := e.Run.ReadFile("PLAN.md"); readErr == nil {
		inputFiles["PLAN.md"] = content
	}

	req := &executor.Request{
		Step:       step,
		RunDir:     e.Run.Dir,
		InputFiles: inputFiles,
	}

	result, err := exec.Execute(ctx, req)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Output), nil
}

// generatePRBody calls the API executor with the body_template prompt and PLAN.md
// to produce PR.md in the run directory.
func (e *Engine) generatePRBody(ctx context.Context, step types.Step) error {
	apiExec, ok := e.Executors["api"]
	if !ok {
		return fmt.Errorf("api executor not registered")
	}

	planContent, err := e.Run.ReadFile("PLAN.md")
	if err != nil {
		return fmt.Errorf("reading PLAN.md: %w", err)
	}

	summaryStep := types.Step{
		Name:           "pr-body-generation",
		Executor:       "api",
		PromptTemplate: step.BodyTemplate,
		Model:          step.Model,
	}
	req := &executor.Request{
		Step:   summaryStep,
		RunDir: e.Run.Dir,
		InputFiles: map[string]string{
			"PLAN.md": planContent,
		},
	}

	result, err := apiExec.Execute(ctx, req)
	if err != nil {
		return fmt.Errorf("generating PR body: %w", err)
	}

	if writeErr := e.Run.WriteFile("PR.md", result.Output); writeErr != nil {
		return fmt.Errorf("writing PR.md: %w", writeErr)
	}
	return nil
}

// resolvePromptForBudget returns the system prompt text for token budget accounting.
func resolvePromptForBudget(e *Engine, step types.Step) (string, bool) {
	if step.PromptTemplate == "" {
		return "", false
	}
	apiExec, ok := e.Executors["api"]
	if !ok {
		return "", false
	}
	type prompter interface {
		ResolvePrompt(name string) (string, bool)
	}
	if p, ok := apiExec.(prompter); ok {
		return p.ResolvePrompt(step.PromptTemplate)
	}
	return "", false
}

// LoadPipeline resolves a pipeline by name from user/project overrides or embedded defaults.
func LoadPipeline(name string) (*Pipeline, error) {
	// 1. project-level override
	projectPath := filepath.Join(".vcoding", "pipelines", name+".yaml")
	if _, err := os.Stat(projectPath); err == nil {
		return ParseFile(projectPath)
	}

	// 2. user-level override
	if home, err := os.UserHomeDir(); err == nil {
		userPath := filepath.Join(home, ".vcoding", "pipelines", name+".yaml")
		if _, err := os.Stat(userPath); err == nil {
			return ParseFile(userPath)
		}
	}

	// 3. embedded default (via assets package — imported at call site)
	return nil, fmt.Errorf("pipeline %q not found", name)
}
