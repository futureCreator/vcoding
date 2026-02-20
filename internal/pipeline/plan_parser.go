package pipeline

import (
	"regexp"
	"strings"
)

// ExtractFilesFromPlan parses PLAN.md content and extracts file paths from
// the "Files to Change" section.
func ExtractFilesFromPlan(planContent string) ([]string, []string) {
	// Find all ## or ### headers for debugging
	headerPattern := regexp.MustCompile(`(?m)^#{2,3}\s*(.+)$`)
	matches := headerPattern.FindAllStringSubmatch(planContent, -1)
	var allHeaders []string
	for _, m := range matches {
		if len(m) > 1 {
			allHeaders = append(allHeaders, strings.TrimSpace(m[1]))
		}
	}

	// Find the "Files to Change" section header (case-insensitive, flexible whitespace, ## or ###)
	sectionPattern := regexp.MustCompile(`(?im)^#{2,3}\s*Files\s+to\s+Change\s*$`)
	loc := sectionPattern.FindStringIndex(planContent)
	if loc == nil {
		return nil, allHeaders
	}

	// Get content after the header
	section := planContent[loc[1]:]

	// Find where the section ends (next ##, ###, or #### at start of line)
	for _, marker := range []string{"\n## ", "\n### ", "\n####"} {
		if nextSection := strings.Index(section, marker); nextSection > 0 {
			section = section[:nextSection]
			break
		}
	}

	var files []string

	// Split by lines and process each
	lines := strings.Split(section, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines
		if line == "" {
			continue
		}

		// Match bullet patterns: - item or * item
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
			// Remove bullet
			content := strings.TrimPrefix(line, "-")
			content = strings.TrimPrefix(content, "*")
			content = strings.TrimSpace(content)

			// Remove backticks if present
			content = strings.Trim(content, "`")

			// Extract file path (stop at description markers)
			file := content
			if idx := strings.Index(content, " - "); idx > 0 {
				file = strings.TrimSpace(content[:idx])
			}
			if idx := strings.Index(content, ":"); idx > 0 {
				// Check if it looks like a description after colon
				afterColon := strings.TrimSpace(content[idx+1:])
				if len(afterColon) > 5 && !strings.Contains(afterColon, "/") && !strings.Contains(afterColon, ".") {
					file = strings.TrimSpace(content[:idx])
				}
			}

			// Validate it's a file path, not a sentence
			if isValidFilePath(file) {
				files = append(files, file)
			}
		}
	}

	return files, allHeaders
}

// isValidFilePath checks if the string looks like a file path
func isValidFilePath(s string) bool {
	if s == "" {
		return false
	}

	// Should not contain spaces (file paths don't have spaces in the path itself)
	if strings.Contains(s, " ") {
		return false
	}

	// Should have at least one dot (extension) or slash (path separator)
	if !strings.Contains(s, ".") && !strings.Contains(s, "/") {
		return false
	}

	// Skip common sentence patterns that happen to have dots
	lower := strings.ToLower(s)
	sentencePatterns := []string{"need to", "should", "will", "can", "may", "might", "could", "would"}
	for _, pattern := range sentencePatterns {
		if strings.HasPrefix(lower, pattern) {
			return false
		}
	}

	return true
}
