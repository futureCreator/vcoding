package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := defaults()
	if cfg.DefaultPipeline != "default" {
		t.Errorf("expected default pipeline 'default', got %q", cfg.DefaultPipeline)
	}
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

	cfg.DefaultPipeline = ""
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty DefaultPipeline")
	}
}

func TestMergeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte("default_pipeline: custom\nlog_level: debug\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := defaults()
	if err := mergeFile(cfg, path); err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultPipeline != "custom" {
		t.Errorf("expected 'custom', got %q", cfg.DefaultPipeline)
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
