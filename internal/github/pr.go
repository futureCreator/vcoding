package github

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/epmk/vcoding/internal/config"
)

// PROptions configures how a PR is created.
type PROptions struct {
	Title      string
	Body       string
	BaseBranch string
	Branch     string
	IssueRef   string // e.g. "42" — becomes "Closes #42"
}

// CreatePR creates a GitHub PR via the gh CLI.
func CreatePR(ctx context.Context, opts PROptions) (string, error) {
	body := opts.Body
	if opts.IssueRef != "" {
		body += fmt.Sprintf("\n\nCloses #%s", opts.IssueRef)
	}

	args := []string{"pr", "create",
		"--title", opts.Title,
		"--body", body,
		"--base", opts.BaseBranch,
		"--head", opts.Branch,
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gh pr create: %w\nstderr: %s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

// CreateBranch creates a new git branch and pushes it.
func CreateBranch(ctx context.Context, branch, base string) error {
	// Check if branch already exists
	checkCmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", branch)
	if err := checkCmd.Run(); err == nil {
		return fmt.Errorf("branch %q already exists", branch)
	}

	// Create and switch to new branch
	cmd := exec.CommandContext(ctx, "git", "checkout", "-b", branch, base)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout -b %s: %w\n%s", branch, err, string(out))
	}

	// Push branch to remote
	pushCmd := exec.CommandContext(ctx, "git", "push", "-u", "origin", branch)
	if out, err := pushCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push: %w\n%s", err, string(out))
	}
	return nil
}

// ExtractTitleFromTicket extracts the first heading from TICKET.md content.
func ExtractTitleFromTicket(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			return strings.TrimSpace(strings.TrimLeft(line, "#"))
		}
		if line != "" {
			return line
		}
	}
	return "chore: automated changes"
}

// BranchName returns the vcoding branch name for a slug.
func BranchName(slug string) string {
	return "vcoding/" + slug
}

// PRExecutor implements executor.Executor for the github-pr step.
type PRExecutor struct {
	Config  *config.Config
	Slug    string
	IssueRef string
}
