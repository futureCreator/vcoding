package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/futureCreator/vcoding/internal/config"
)

// FileEntry represents a collected project file.
type FileEntry struct {
	Path    string
	Content string
}

// Scan collects project files based on config patterns.
func Scan(cfg *config.ProjectCtxConfig) ([]FileEntry, error) {
	maxSize, err := parseSize(cfg.MaxFileSize)
	if err != nil {
		return nil, fmt.Errorf("parsing max_file_size: %w", err)
	}

	var entries []FileEntry
	seen := map[string]bool{}

	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable paths
		}
		if info.IsDir() {
			// Check if this directory matches an exclude pattern
			dirName := path + "/"
			for _, pat := range cfg.ExcludePatterns {
				if strings.HasPrefix(dirName, pat) || strings.Contains(dirName, "/"+pat) {
					return filepath.SkipDir
				}
			}
			// Skip hidden directories except root "."
			if path != "." && strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if seen[path] {
			return nil
		}

		// Check exclude patterns
		for _, pat := range cfg.ExcludePatterns {
			if strings.Contains(path, pat) {
				return nil
			}
		}

		// Check include patterns
		matched := false
		for _, pat := range cfg.IncludePatterns {
			ok, _ := filepath.Match(pat, filepath.Base(path))
			if ok {
				matched = true
				break
			}
		}
		if !matched {
			return nil
		}

		if info.Size() > maxSize {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		seen[path] = true
		entries = append(entries, FileEntry{Path: path, Content: string(content)})

		if len(entries) >= cfg.MaxFiles {
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// FormatContext formats project files into a markdown string for LLM context.
func FormatContext(entries []FileEntry) string {
	if len(entries) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("## Project Context\n\n")
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("### %s\n\n```\n%s\n```\n\n", e.Path, e.Content))
	}
	return sb.String()
}

func parseSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 50 * 1024, nil
	}
	var multiplier int64 = 1
	if strings.HasSuffix(s, "KB") {
		multiplier = 1024
		s = s[:len(s)-2]
	} else if strings.HasSuffix(s, "MB") {
		multiplier = 1024 * 1024
		s = s[:len(s)-2]
	}
	var n int64
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	return n * multiplier, nil
}
