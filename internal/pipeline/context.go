package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Context manages file-based context between pipeline steps.
type Context struct {
	RunDir     string
	ProjectCtx string // pre-built project context markdown
	GitDiff    string
}

// ResolveInput loads the content of each input spec.
// Supports special inputs: "git:diff", "project:context"
// and regular filenames resolved from RunDir.
func (c *Context) ResolveInput(inputs []string) (map[string]string, error) {
	files := make(map[string]string)

	for _, inp := range inputs {
		switch inp {
		case "git:diff":
			files["git:diff"] = c.GitDiff
		case "project:context":
			files["project:context"] = c.ProjectCtx
		default:
			content, err := c.readFile(inp)
			if err != nil {
				return nil, fmt.Errorf("reading input %q: %w", inp, err)
			}
			files[inp] = content
		}
	}

	return files, nil
}

func (c *Context) readFile(name string) (string, error) {
	// Try run directory first
	runPath := filepath.Join(c.RunDir, name)
	if data, err := os.ReadFile(runPath); err == nil {
		return string(data), nil
	}
	// Try current working directory
	if data, err := os.ReadFile(name); err == nil {
		return string(data), nil
	}
	return "", fmt.Errorf("file %q not found in run dir or working dir", name)
}

// TruncateToTokenBudget truncates the combined content of files to stay within
// maxTokens (rough estimate: 4 chars ≈ 1 token).
// Keys are processed in sorted order for deterministic results.
func TruncateToTokenBudget(files map[string]string, systemPrompt string, maxTokens int) map[string]string {
	if maxTokens <= 0 {
		return files
	}
	// Reserve budget for system prompt
	used := len(systemPrompt) / 4
	budget := maxTokens - used
	if budget <= 0 {
		return files
	}

	// Sort keys for deterministic truncation order
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make(map[string]string)
	for _, name := range keys {
		content := files[name]
		contentTokens := len(content) / 4
		if used+contentTokens <= budget {
			result[name] = content
			used += contentTokens
		} else {
			// Truncate this file to fit remaining budget
			remaining := (budget - used) * 4
			if remaining > 0 {
				if remaining < len(content) {
					content = content[:remaining]
					content += "\n\n[... truncated due to token limit ...]"
				}
				result[name] = content
			}
			break
		}
	}
	return result
}

// BuildTicketContent creates TICKET.md content from an input title + body.
func BuildTicketContent(title, body string) string {
	var sb strings.Builder
	sb.WriteString("# ")
	sb.WriteString(title)
	sb.WriteString("\n\n")
	sb.WriteString(body)
	sb.WriteString("\n")
	return sb.String()
}
