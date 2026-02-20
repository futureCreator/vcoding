package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/futureCreator/vcoding/internal/config"
	"github.com/futureCreator/vcoding/internal/executor"
	vlog "github.com/futureCreator/vcoding/internal/log"
	"github.com/futureCreator/vcoding/internal/run"
	"github.com/futureCreator/vcoding/internal/types"
)

// Engine orchestrates pipeline step execution.
type Engine struct {
	Config    *config.Config
	Pipeline  *Pipeline
	Executors map[string]executor.Executor
	Run       *run.Run
	Display   *Display
	Verbose   bool
}

// stepDisplayModel returns a human-readable label for the step's executor/model,
// suitable for the terminal output model column.
func (e *Engine) stepDisplayModel(step types.Step) string {
	switch {
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
		var artifactContent string

		if step.Executor == "" {
			stepErr = fmt.Errorf("step %q has no executor", step.Name)
		} else {
			detail, artifactContent, cost, stepErr = e.runExecutorStep(ctx, step, pipelineCtx)
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

		e.Display.StepDone(step.Name, displayModel, detail, cost, duration, artifactContent)
	}

	if err := e.Run.Complete(); err != nil {
		vlog.Warn("failed to mark run complete", "err", err)
	}

	e.Display.Summary(e.Run.Meta.TotalCost, time.Since(startTime))
	return nil
}

// resolveModel replaces role placeholders ($planner, $reviewer, $editor)
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
	}
	return model
}

func (e *Engine) runExecutorStep(ctx context.Context, step types.Step, pipelineCtx *Context) (detail, artifactContent string, cost float64, err error) {
	step.Model = e.resolveModel(step.Model)

	exec, ok := e.Executors[step.Executor]
	if !ok {
		return "", "", 0, fmt.Errorf("unknown executor %q", step.Executor)
	}

	inputFiles, err := pipelineCtx.ResolveInput(step.Input)
	if err != nil {
		return "", "", 0, err
	}

	// For Revise step, filter project context to only include files from PLAN.md
	if strings.EqualFold(step.Name, "Revise") {
		if planContent, ok := inputFiles["PLAN.md"]; ok {
			if projectCtx, ok := inputFiles["project:context"]; ok && projectCtx != "" {
				filteredCtx := FilterProjectContextByPlanFiles(planContent, projectCtx)

				// Debug: Log token counts and save filtered context
				originalTokens := EstimateTokens(projectCtx)
				filteredTokens := EstimateTokens(filteredCtx)
				targetFiles := ExtractFilesFromPlan(planContent)

				vlog.Info("Revise context filtering",
					"files_in_plan", len(targetFiles),
					"original_tokens", originalTokens,
					"filtered_tokens", filteredTokens,
					"reduction_percent", fmt.Sprintf("%.1f%%", float64(originalTokens-filteredTokens)/float64(originalTokens)*100))

				// Save filtered context for debugging
				if writeErr := e.Run.WriteFile("Revise-context-filtered.md", filteredCtx); writeErr != nil {
					vlog.Warn("failed to save filtered context", "err", writeErr)
				}
				// Also save original for comparison
				if writeErr := e.Run.WriteFile("Revise-context-original.md", projectCtx); writeErr != nil {
					vlog.Warn("failed to save original context", "err", writeErr)
				}

				inputFiles["project:context"] = filteredCtx
			}
		}
	}

	// Apply token budget truncation for API steps.
	if step.Executor == "api" && e.Config.MaxContextTokens > 0 {
		sp, _ := resolvePromptForBudget(e, step)
		inputFiles = TruncateToTokenBudget(inputFiles, sp, e.Config.MaxContextTokens)
	}

	req := &executor.Request{
		Step:       step,
		RunDir:     e.Run.Dir,
		InputFiles: inputFiles,
	}

	result, err := exec.Execute(ctx, req)
	if err != nil {
		return "", "", 0, err
	}

	// Save output file to run directory
	if step.Output != "" && result.Output != "" {
		if writeErr := e.Run.WriteFile(step.Output, result.Output); writeErr != nil {
			vlog.Warn("failed to write output file", "file", step.Output, "err", writeErr)
		}
		detail = step.Output
		artifactContent = result.Output
	} else if step.Output == "" {
		detail = fmt.Sprintf("%.0fs", result.Duration.Seconds())
	}

	return detail, artifactContent, result.Cost, nil
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
