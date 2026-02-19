package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/epmk/vcoding/internal/assets"
	"github.com/epmk/vcoding/internal/config"
	"github.com/epmk/vcoding/internal/executor"
	"github.com/epmk/vcoding/internal/github"
	vlog "github.com/epmk/vcoding/internal/log"
	"github.com/epmk/vcoding/internal/pipeline"
	"github.com/epmk/vcoding/internal/project"
	"github.com/epmk/vcoding/internal/run"
	"github.com/epmk/vcoding/internal/source"
)

// runPipeline is the shared entry point for pick and do commands.
func runPipeline(ctx context.Context, src source.Source, pipelineName string, force bool) error {
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

	// Check dirty working tree
	if !force {
		dirty, err := project.IsDirtyWorkingTree()
		if err != nil {
			vlog.Warn("could not check git status", "err", err)
		} else if dirty {
			return fmt.Errorf("working tree has uncommitted changes; commit or stash them first (or use --force)")
		}
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

	// Write TICKET.md or SPEC.md to run directory
	ticketContent := pipeline.BuildTicketContent(input.Title, input.Body)
	ticketFile := "TICKET.md"
	if input.Mode == "do" {
		ticketFile = "TICKET.md" // normalize spec to ticket too
	}
	if err := r.WriteFile(ticketFile, ticketContent); err != nil {
		return fmt.Errorf("writing ticket: %w", err)
	}

	// Load all prompt templates
	prompts, err := assets.AllPrompts()
	if err != nil {
		return fmt.Errorf("loading prompts: %w", err)
	}

	// Build executors
	executors := buildExecutors(cfg, prompts, input.Slug, input.Ref)

	// Collect project context
	projectCtxStr := ""
	if files, err := project.Scan(&cfg.ProjectContext); err == nil {
		projectCtxStr = project.FormatContext(files)
	} else {
		vlog.Warn("could not scan project files", "err", err)
	}

	// Collect git diff
	gitDiff, _ := project.Diff()
	gitDiffBase, _ := project.DiffFromBase(cfg.GitHub.BaseBranch)

	// Build pipeline context
	pipelineCtx := &pipeline.Context{
		RunDir:      r.Dir,
		ProjectCtx:  projectCtxStr,
		GitDiff:     gitDiff,
		GitDiffBase: gitDiffBase,
	}

	// Create branch for this run
	branchName := github.BranchName(input.Slug)
	if err := github.CreateBranch(ctx, branchName, cfg.GitHub.BaseBranch); err != nil {
		vlog.Warn("could not create git branch", "branch", branchName, "err", err)
	}

	// Run pipeline
	disp := pipeline.NewDisplay(input.Title)
	disp.Header()

	engine := &pipeline.Engine{
		Config:    cfg,
		Pipeline:  ppl,
		Executors: executors,
		Run:       r,
		Display:   disp,
	}

	return engine.Execute(ctx, pipelineCtx)
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

func buildExecutors(cfg *config.Config, prompts map[string]string, slug, issueRef string) map[string]executor.Executor {
	execs := map[string]executor.Executor{
		"api": &executor.APIExecutor{
			Config:  cfg,
			Prompts: prompts,
		},
		"claude-code": &executor.ClaudeCodeExecutor{
			Config: cfg,
		},
		"shell": &executor.ShellExecutor{},
		"github-pr": &github.PRExecutor{
			Config:   cfg,
			Slug:     slug,
			IssueRef: issueRef,
		},
	}
	return execs
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
