package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/epmk/vcoding/internal/config"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"50KB", 50 * 1024},
		{"1MB", 1024 * 1024},
		{"100KB", 100 * 1024},
	}
	for _, tt := range tests {
		got, err := parseSize(tt.input)
		if err != nil {
			t.Errorf("parseSize(%q) error: %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("parseSize(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestScan(t *testing.T) {
	dir := t.TempDir()
	// Create test files
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# README"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "binary.bin"), []byte("binary"), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to temp dir
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	cfg := &config.ProjectCtxConfig{
		MaxFiles:        10,
		MaxFileSize:     "50KB",
		IncludePatterns: []string{"*.go", "*.md"},
		ExcludePatterns: []string{"vendor/"},
	}

	entries, err := Scan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}
