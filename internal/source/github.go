package source

import (
	"context"
	"fmt"

	"github.com/epmk/vcoding/internal/github"
)

// GitHubSource fetches a GitHub issue via the gh CLI.
type GitHubSource struct {
	IssueNumber string
}

func (s *GitHubSource) Fetch(ctx context.Context) (*Input, error) {
	issue, err := github.FetchIssue(ctx, s.IssueNumber)
	if err != nil {
		return nil, fmt.Errorf("fetching issue %s: %w", s.IssueNumber, err)
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
	var sb []byte
	for i := 0; i < len(title); i++ {
		c := title[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			sb = append(sb, c)
		} else if c >= 'A' && c <= 'Z' {
			sb = append(sb, c+32) // to lower
		} else if len(sb) > 0 && sb[len(sb)-1] != '-' {
			sb = append(sb, '-')
		}
	}
	// trim trailing dash
	for len(sb) > 0 && sb[len(sb)-1] == '-' {
		sb = sb[:len(sb)-1]
	}
	s := string(sb)
	if len(s) > 40 {
		s = s[:40]
		for len(s) > 0 && s[len(s)-1] == '-' {
			s = s[:len(s)-1]
		}
	}
	if s == "" {
		return "issue"
	}
	return s
}
