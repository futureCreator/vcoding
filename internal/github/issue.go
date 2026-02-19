package github

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// Issue holds GitHub issue data.
type Issue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

// FetchIssue retrieves a GitHub issue via the gh CLI.
func FetchIssue(ctx context.Context, number string) (*Issue, error) {
	cmd := exec.CommandContext(ctx, "gh", "issue", "view", number,
		"--json", "number,title,body,labels")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh issue view %s: %w", number, err)
	}

	var issue Issue
	if err := json.Unmarshal(out, &issue); err != nil {
		return nil, fmt.Errorf("parsing issue JSON: %w", err)
	}
	return &issue, nil
}
