package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/epmk/vcoding/internal/assets"
	"github.com/epmk/vcoding/internal/config"
	"github.com/epmk/vcoding/internal/executor"
	vlog "github.com/epmk/vcoding/internal/log"
	"github.com/epmk/vcoding/internal/pipeline"
	"github.com/epmk/vcoding/internal/project"
	"github.com/epmk/vcoding/internal/run"
	"github.com/epmk/vcoding/internal/source"
)

// runPipeline is the shared entry point for pick and do commands.
func runPipeline(ctx context.Context, src source.Source, pipelineName string, verbose bool) error {
	if err := checkPrerequisites(); err != nil {
		return err
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Init logging
	logFile := openLogFile()
	vlog.Init(cfg.LogLevel, logFile)
	if logFile != nil {
		defer logFile.Close()
	}

	// Collect git info
	gitInfo, err := project.CollectGitInfo()
	if err != nil {
		vlog.Warn("could not collect git info", "err", err)
		gitInfo = &project.GitInfo{}
	}

	// Fetch input
	input, err := src.Fetch(ctx)
	if err != nil {
		return fmt.Errorf("fetching input: %w", err)
	}

	// Load pipeline
	ppl, err := loadPipeline(cfg, pipelineName)
	if err != nil {
		return fmt.Errorf("loading pipeline %q: %w", pipelineName, err)
	}

	// Create run directory
	r, err := run.New(input.Mode, input.Ref, input.Slug, gitInfo.Branch, gitInfo.Commit)
	if err != nil {
		return fmt.Errorf("creating run: %w", err)
	}

	// Write TICKET.md to run directory
	ticketContent := pipeline.BuildTicketContent(input.Title, input.Body)
	if err := r.WriteFile("TICKET.md", ticketContent); err != nil {
		return fmt.Errorf("writing ticket: %w", err)
	}

	// Load all prompt templates
	prompts, err := assets.AllPrompts()
	if err != nil {
		return fmt.Errorf("loading prompts: %w", err)
	}

	// Build executors
	executors := buildExecutors(cfg, prompts)

	// Collect project context
	projectCtxStr := ""
	if files, err := project.Scan(&cfg.ProjectContext); err == nil {
		projectCtxStr = project.FormatContext(files)
	} else {
		vlog.Warn("could not scan project files", "err", err)
	}

	// Collect git diff
	gitDiff, _ := project.Diff()

	// Build pipeline context
	pipelineCtx := &pipeline.Context{
		RunDir:     r.Dir,
		ProjectCtx: projectCtxStr,
		GitDiff:    gitDiff,
	}

	// Run pipeline
	disp := pipeline.NewDisplay(input.Title, verbose)
	disp.Header()

	engine := &pipeline.Engine{
		Config:    cfg,
		Pipeline:  ppl,
		Executors: executors,
		Run:       r,
		Display:   disp,
		Verbose:   verbose,
	}

	if err := engine.Execute(ctx, pipelineCtx); err != nil {
		return err
	}

	// Copy the final PLAN.md to .vcoding/PLAN.md for easy access.
	if plan, err := r.ReadFile("PLAN.md"); err == nil {
		dest := filepath.Join(".vcoding", "PLAN.md")
		if writeErr := os.WriteFile(dest, []byte(plan), 0644); writeErr == nil {
			fmt.Printf("\nPlan saved to %s\n", dest)
		}
	}

	return nil
}

func loadPipeline(cfg *config.Config, name string) (*pipeline.Pipeline, error) {
	// Try filesystem first (project/user overrides)
	ppl, err := pipeline.LoadPipeline(name)
	if err == nil {
		return ppl, nil
	}
	// Fall back to embedded
	data, err := assets.LoadPipeline(name)
	if err != nil {
		return nil, fmt.Errorf("pipeline %q not found", name)
	}
	return pipeline.Parse(data)
}

func buildExecutors(cfg *config.Config, prompts map[string]string) map[string]executor.Executor {
	return map[string]executor.Executor{
		"api": &executor.APIExecutor{Config: cfg, Prompts: prompts},
	}
}

func openLogFile() *os.File {
	dir := ".vcoding"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil
	}
	f, err := os.OpenFile(dir+"/vcoding.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil
	}
	return f
}
