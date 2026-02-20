package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := defaults()
	if cfg.MaxContextTokens != 80000 {
		t.Errorf("expected MaxContextTokens 80000, got %d", cfg.MaxContextTokens)
	}
	if cfg.GitHub.BaseBranch != "main" {
		t.Errorf("expected base branch 'main', got %q", cfg.GitHub.BaseBranch)
	}
}

func TestValidate(t *testing.T) {
	cfg := defaults()
	if err := cfg.Validate(); err != nil {
		t.Errorf("defaults should be valid: %v", err)
	}

	cfg.Provider.Endpoint = ""
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty provider.endpoint")
	}
}

func TestMergeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte("log_level: debug\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := defaults()
	if err := mergeFile(cfg, path); err != nil {
		t.Fatal(err)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected 'debug', got %q", cfg.LogLevel)
	}
}

func TestMergeFileNotExist(t *testing.T) {
	cfg := defaults()
	err := mergeFile(cfg, "/nonexistent/path/config.yaml")
	if err == nil || !os.IsNotExist(err) {
		t.Errorf("expected os.IsNotExist error, got %v", err)
	}
}

func TestMergeFileRejectsGitHubToken(t *testing.T) {
	dir := t.TempDir()

	cases := []struct {
		name    string
		content string
	}{
		{
			name:    "top-level github_token",
			content: "github_token: ghp_abc123\n",
		},
		{
			name:    "github.token nested field",
			content: "github:\n  token: ghp_abc123\n  base_branch: main\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name+".yaml")
			if err := os.WriteFile(path, []byte(tc.content), 0644); err != nil {
				t.Fatal(err)
			}
			cfg := defaults()
			err := mergeFile(cfg, path)
			if err == nil {
				t.Error("expected error for deprecated token field, got nil")
			}
		})
	}
}
