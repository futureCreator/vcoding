package github

import (
	"context"
	"fmt"
	"time"

	"github.com/epmk/vcoding/internal/executor"
)

// Execute implements executor.Executor for the github-pr step.
func (e *PRExecutor) Execute(ctx context.Context, req *executor.Request) (*executor.Result, error) {
	start := time.Now()

	// Extract title from TICKET.md (or title_from file)
	title := ""
	titleFile := req.Step.TitleFrom
	if titleFile == "" {
		titleFile = "TICKET.md"
	}
	if content, ok := req.InputFiles[titleFile]; ok {
		title = ExtractTitleFromTicket(content)
	}
	if title == "" {
		title = "chore: automated changes"
	}

	// Build PR body: prefer PR.md (generated from body_template), fall back to PLAN.md
	body := ""
	if prContent, ok := req.InputFiles["PR.md"]; ok {
		body = prContent
	} else if planContent, ok := req.InputFiles["PLAN.md"]; ok {
		body = planContent
	}

	branch := BranchName(e.Slug)
	baseBranch := e.Config.GitHub.BaseBranch
	if baseBranch == "" {
		baseBranch = "main"
	}

	prURL, err := CreatePR(ctx, PROptions{
		Title:      title,
		Body:       body,
		BaseBranch: baseBranch,
		Branch:     branch,
		IssueRef:   e.IssueRef,
	})
	if err != nil {
		return nil, fmt.Errorf("creating PR: %w", err)
	}

	return &executor.Result{
		Output:   prURL,
		Duration: time.Since(start),
	}, nil
}
