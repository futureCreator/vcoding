package source

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// GitHubSource fetches a GitHub issue via the gh CLI.
type GitHubSource struct {
	IssueNumber string
}

type ghIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

func (s *GitHubSource) Fetch(ctx context.Context) (*Input, error) {
	cmd := exec.CommandContext(ctx, "gh", "issue", "view", s.IssueNumber, "--json", "number,title,body")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh issue view %s: %w", s.IssueNumber, err)
	}

	var issue ghIssue
	if err := json.Unmarshal(out, &issue); err != nil {
		return nil, fmt.Errorf("parsing issue JSON: %w", err)
	}

	slug := slugFromTitle(issue.Title)

	return &Input{
		Title: issue.Title,
		Body:  issue.Body,
		Slug:  fmt.Sprintf("%s-%s", s.IssueNumber, slug),
		Mode:  "pick",
		Ref:   s.IssueNumber,
	}, nil
}

func slugFromTitle(title string) string {
	title = strings.ToLower(title)
	var sb strings.Builder
	for _, r := range title {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		} else if sb.Len() > 0 {
			last := rune(sb.String()[sb.Len()-1])
			if last != '-' {
				sb.WriteRune('-')
			}
		}
	}
	s := strings.Trim(sb.String(), "-")
	if len(s) > 40 {
		s = s[:40]
		s = strings.TrimRight(s, "-")
	}
	if s == "" {
		return "issue"
	}
	return s
}
