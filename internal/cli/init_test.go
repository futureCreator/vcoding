package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/epmk/vcoding/internal/assets"
	"github.com/epmk/vcoding/internal/config"
	"gopkg.in/yaml.v3"
)

// chdirTemp changes the working directory to a new temp dir for the duration of
// the test, restoring the original directory when the test completes.
func chdirTemp(t *testing.T) string {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	return tmp
}

func TestDefaultTemplateContainsComments(t *testing.T) {
	content, err := assets.LoadTemplate("config.yaml")
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}
	for _, want := range []string{
		"# OpenRouter-compatible",
		"# Models for each pipeline role",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("default template missing comment %q", want)
		}
	}
}

func TestMinimalTemplateHasNoInlineComments(t *testing.T) {
	content, err := assets.LoadTemplate("config.minimal.yaml")
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Only the first line (header) is allowed to be a comment.
		if i == 0 {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			t.Errorf("minimal template has unexpected comment at line %d: %q", i+1, line)
		}
	}
}

func TestTemplatesAreValidYAML(t *testing.T) {
	for _, name := range []string{"config.yaml", "config.minimal.yaml"} {
		content, err := assets.LoadTemplate(name)
		if err != nil {
			t.Fatalf("LoadTemplate(%q): %v", name, err)
		}
		var cfg config.Config
		if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
			t.Errorf("template %q is not valid YAML: %v", name, err)
		}
	}
}

func TestTemplatesDeeplyEqualDefaults(t *testing.T) {
	defaults := config.Defaults()
	for _, name := range []string{"config.yaml", "config.minimal.yaml"} {
		content, err := assets.LoadTemplate(name)
		if err != nil {
			t.Fatalf("LoadTemplate(%q): %v", name, err)
		}
		var cfg config.Config
		if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
			t.Fatalf("unmarshal %q: %v", name, err)
		}
		if !reflect.DeepEqual(cfg, *defaults) {
			t.Errorf("template %q does not match config.Defaults():\ngot:  %+v\nwant: %+v", name, cfg, *defaults)
		}
	}
}

func TestInitCreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	chdirTemp(t)
	initMinimal = false

	if err := runInit(initCmd, nil); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	configPath := filepath.Join(tmpDir, ".vcoding", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}
	if len(data) == 0 {
		t.Error("config file is empty")
	}
}

func TestInitSkipsWhenFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	chdirTemp(t)

	configDir := filepath.Join(tmpDir, ".vcoding")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	original := []byte("original content")
	if err := os.WriteFile(configPath, original, 0644); err != nil {
		t.Fatal(err)
	}

	if err := runInit(initCmd, nil); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(original) {
		t.Errorf("runInit overwrote existing config: got %q, want %q", data, original)
	}
}

func TestInitMinimalFlag(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	chdirTemp(t)
	initMinimal = true
	defer func() { initMinimal = false }()

	if err := runInit(initCmd, nil); err != nil {
		t.Fatalf("runInit --minimal: %v", err)
	}

	configPath := filepath.Join(tmpDir, ".vcoding", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if i == 0 {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			t.Errorf("minimal config has unexpected comment at line %d: %q", i+1, line)
		}
	}
}

func TestInitCreatesConventionFiles(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	chdirTemp(t)

	if err := runInit(initCmd, nil); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	for _, name := range []string{"CLAUDE.md", ".cursorrules", "AGENTS.md"} {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Errorf("convention file %q not created: %v", name, err)
			continue
		}
		if !strings.Contains(string(data), ".vcoding/PLAN.md") {
			t.Errorf("convention file %q does not mention .vcoding/PLAN.md", name)
		}
	}
}

func TestInitUpdatesExistingConventionFiles(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	chdirTemp(t)

	// Pre-create convention files with stale content.
	if err := os.WriteFile("CLAUDE.md", []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := runInit(initCmd, nil); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	data, err := os.ReadFile("CLAUDE.md")
	if err != nil {
		t.Fatalf("CLAUDE.md not found after init: %v", err)
	}
	if string(data) == "old content" {
		t.Error("runInit did not update existing CLAUDE.md")
	}
}
