package project

import (
	"fmt"
	"os/exec"
	"strings"
)

// GitInfo holds current git state.
type GitInfo struct {
	Branch string
	Commit string
	Diff   string
	IsDirty bool
}

// CollectGitInfo gathers branch, commit, and diff information.
func CollectGitInfo() (*GitInfo, error) {
	branch, err := gitOutput("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("getting git branch: %w", err)
	}

	commit, err := gitOutput("rev-parse", "--short", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("getting git commit: %w", err)
	}

	dirty, err := isDirty()
	if err != nil {
		return nil, err
	}

	return &GitInfo{
		Branch:  branch,
		Commit:  commit,
		IsDirty: dirty,
	}, nil
}

// Diff returns the current staged+unstaged diff.
func Diff() (string, error) {
	staged, err := gitOutput("diff", "--cached")
	if err != nil {
		return "", fmt.Errorf("getting staged diff: %w", err)
	}
	unstaged, err := gitOutput("diff")
	if err != nil {
		return "", fmt.Errorf("getting unstaged diff: %w", err)
	}
	combined := strings.TrimSpace(staged + "\n" + unstaged)
	return combined, nil
}

func isDirty() (bool, error) {
	out, err := gitOutput("status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("checking git status: %w", err)
	}
	return strings.TrimSpace(out) != "", nil
}

func gitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
