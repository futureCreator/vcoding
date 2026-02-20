package source

import (
	"context"
)

// PromptSource uses a user-provided message as pipeline input.
type PromptSource struct {
	Prompt string
}

func (s *PromptSource) Fetch(ctx context.Context) (*Input, error) {
	title := s.Prompt
	if len(title) > 50 {
		title = title[:50] + "..."
	}

	return &Input{
		Title: title,
		Body:  s.Prompt,
		Slug:  slugFromTitle(s.Prompt),
		Mode:  "ask",
		Ref:   "user-prompt",
	}, nil
}
