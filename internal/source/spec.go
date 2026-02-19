package source

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

// SpecSource reads a local spec file as pipeline input.
type SpecSource struct {
	Path string
}

func (s *SpecSource) Fetch(ctx context.Context) (*Input, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		return nil, fmt.Errorf("reading spec file %s: %w", s.Path, err)
	}
	content := string(data)

	title := extractTitle(content)
	slug := slugFromTitle(title)
	if slug == "" || slug == "issue" {
		// Fall back to file name
		base := s.Path
		if idx := strings.LastIndex(base, "/"); idx >= 0 {
			base = base[idx+1:]
		}
		base = strings.TrimSuffix(base, ".md")
		slug = slugFromTitle(base)
	}

	return &Input{
		Title: title,
		Body:  content,
		Slug:  slug,
		Mode:  "do",
		Ref:   s.Path,
	}, nil
}

// extractTitle returns the first non-empty line, stripping leading "#".
func extractTitle(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		line = strings.TrimLeft(line, "#")
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return "spec"
}
