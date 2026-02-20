// Package assets provides embedded prompt templates and default pipeline definitions.
// NOTE: Template files in templates/ must stay in sync with config.defaults() in internal/config/config.go.
package assets

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed prompts/*.md
var promptsFS embed.FS

//go:embed pipelines/*.yaml
var pipelinesFS embed.FS

//go:embed all:templates
var templatesFS embed.FS

// LoadPrompt returns the content of a prompt template by name.
// Override lookup order: project .vcoding/prompts/ > user ~/.vcoding/prompts/ > embedded.
func LoadPrompt(name string) (string, error) {
	return loadWithOverride("prompts", name+".md", promptsFS)
}

// LoadPipeline returns the content of a pipeline YAML by name.
// Override lookup order: project .vcoding/pipelines/ > user ~/.vcoding/pipelines/ > embedded.
func LoadPipeline(name string) ([]byte, error) {
	content, err := loadWithOverride("pipelines", name+".yaml", pipelinesFS)
	if err != nil {
		return nil, err
	}
	return []byte(content), nil
}

// AllPrompts returns all embedded prompt templates as a map (name → content).
func AllPrompts() (map[string]string, error) {
	return readAll(promptsFS, "prompts", ".md")
}

func loadWithOverride(dir, filename string, embedded embed.FS) (string, error) {
	// 1. project-level override
	projectPath := filepath.Join(".vcoding", dir, filename)
	if data, err := os.ReadFile(projectPath); err == nil {
		return string(data), nil
	}

	// 2. user-level override
	if home, err := os.UserHomeDir(); err == nil {
		userPath := filepath.Join(home, ".vcoding", dir, filename)
		if data, err := os.ReadFile(userPath); err == nil {
			return string(data), nil
		}
	}

	// 3. embedded default
	embeddedPath := filepath.Join(dir, filename)
	data, err := embedded.ReadFile(embeddedPath)
	if err != nil {
		return "", fmt.Errorf("%s %q not found", dir, filename)
	}
	return string(data), nil
}

// LoadTemplate returns the content of a config template by explicit filename.
// Valid filenames: "config.yaml" (with comments) or "config.minimal.yaml" (without comments).
func LoadTemplate(filename string) (string, error) {
	data, err := templatesFS.ReadFile(filepath.Join("templates", filename))
	if err != nil {
		return "", fmt.Errorf("template %q not found", filename)
	}
	return string(data), nil
}

// conventionFilenames are the project-level AI convention files created by vcoding init.
var conventionFilenames = []string{"CLAUDE.md", ".cursorrules", "AGENTS.md"}

// ConventionFiles returns the content of each project-level convention file template.
// The returned map key is the target filename (e.g. "CLAUDE.md", ".cursorrules").
func ConventionFiles() (map[string]string, error) {
	result := make(map[string]string, len(conventionFilenames))
	for _, name := range conventionFilenames {
		data, err := templatesFS.ReadFile(filepath.Join("templates", name))
		if err != nil {
			return nil, fmt.Errorf("convention template %q not found: %w", name, err)
		}
		result[name] = string(data)
	}
	return result, nil
}

func readAll(fsys embed.FS, dir, ext string) (map[string]string, error) {
	result := map[string]string{}
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) != ext {
			continue
		}
		data, err := fsys.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		key := name[:len(name)-len(ext)]
		result[key] = string(data)
	}
	return result, nil
}
