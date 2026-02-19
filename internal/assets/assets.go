// Package assets provides embedded prompt templates and default pipeline definitions.
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
