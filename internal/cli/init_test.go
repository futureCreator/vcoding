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

func TestDefaultTemplateContainsComments(t *testing.T) {
	content, err := assets.LoadTemplate("config.yaml")
	if err != nil {
		t.Fatalf("LoadTemplate: %v", err)
	}
	for _, want := range []string{
		"# Pipeline to use",
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
