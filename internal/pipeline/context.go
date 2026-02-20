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

// FilterProjectContextByPlanFiles filters the project context to only include
// files specified in the plan's "Files to Change" section.
// If planContent is empty or no files are found, returns the original context.
func FilterProjectContextByPlanFiles(planContent, projectCtx string) string {
	if projectCtx == "" || planContent == "" {
		return projectCtx
	}

	targetFiles, _ := ExtractFilesFromPlan(planContent)
	if len(targetFiles) == 0 {
		return projectCtx
	}

	// Build a set of target files for quick lookup
	targetSet := make(map[string]bool)
	for _, f := range targetFiles {
		targetSet[f] = true
		// Also add variants without leading ./
		targetSet[strings.TrimPrefix(f, "./")] = true
		// And with leading ./
		targetSet["./"+f] = true
	}

	// Parse the project context markdown and filter files
	var result strings.Builder
	lines := strings.Split(projectCtx, "\n")
	inTargetFile := false

	for _, line := range lines {
		// Check if this is a file header (### filename)
		if strings.HasPrefix(line, "### ") {
			// Extract filename from header
			fileName := strings.TrimPrefix(line, "### ")
			fileName = strings.TrimSpace(fileName)

			// Check if this file should be included
			if targetSet[fileName] {
				inTargetFile = true
				result.WriteString(line + "\n")
			} else {
				inTargetFile = false
			}
			continue
		}

		// If we're in a target file, include all lines until next file
		if inTargetFile {
			result.WriteString(line + "\n")
		}
	}

	filtered := result.String()
	if filtered == "" {
		// If filtering removed everything, return original to be safe
		return projectCtx
	}

	return "## Project Context (Filtered to Files in Plan)\n\n" + filtered
}

// EstimateTokens estimates the number of tokens in a string.
// Uses a rough heuristic of ~4 characters per token (approximation for GPT models).
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	// Rough estimate: 1 token ≈ 4 characters for English text
	return len(text) / 4
}
