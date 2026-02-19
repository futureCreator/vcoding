package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"# My Title\n\nbody", "My Title"},
		{"## Second Level\nbody", "Second Level"},
		{"\n\n# After Blank\n", "After Blank"},
		{"No heading\nbody", "No heading"},
		{"", "spec"},
	}
	for _, tt := range tests {
		got := extractTitle(tt.input)
		if got != tt.want {
			t.Errorf("extractTitle(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSlugFromTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Add User Authentication", "add-user-authentication"},
		{"Fix Bug #42!", "fix-bug-42"},
		{"  leading spaces  ", "leading-spaces"},
	}
	for _, tt := range tests {
		got := slugFromTitle(tt.input)
		if got != tt.want {
			t.Errorf("slugFromTitle(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSpecSourceFetch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SPEC.md")
	content := "# Add Logging\n\nWe need structured logging.\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	src := &SpecSource{Path: path}
	inp, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error: %v", err)
	}
	if inp.Title != "Add Logging" {
		t.Errorf("expected title 'Add Logging', got %q", inp.Title)
	}
	if inp.Mode != "do" {
		t.Errorf("expected mode 'do', got %q", inp.Mode)
	}
	if inp.Slug != "add-logging" {
		t.Errorf("expected slug 'add-logging', got %q", inp.Slug)
	}
}
