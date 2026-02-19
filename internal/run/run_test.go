package run

import (
	"os"
	"strings"
	"testing"
)

func TestSanitizeSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Fix Auth Bug", "fix-auth-bug"},
		{"Add User's Profile (v2)", "add-user-s-profile-v2"},
		{"  spaces  ", "spaces"},
		{"", "run"},
		{"123-abc", "123-abc"},
		{strings.Repeat("a", 50), strings.Repeat("a", 40)},
	}
	for _, tt := range tests {
		got := sanitizeSlug(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeSlug(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNew(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	r, err := New("do", "SPEC.md", "add-logging", "main", "abc1234")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if r.Meta.Status != "running" {
		t.Errorf("expected status 'running', got %q", r.Meta.Status)
	}
	if r.Meta.GitBranch != "main" {
		t.Errorf("expected branch 'main', got %q", r.Meta.GitBranch)
	}

	// Verify meta.json was written
	if _, err := os.Stat(r.FilePath("meta.json")); err != nil {
		t.Errorf("meta.json not created: %v", err)
	}

	// Verify latest symlink
	latestTarget, err := os.Readlink(dir + "/.vcoding/runs/latest")
	if err != nil {
		t.Errorf("latest symlink not created: %v", err)
	}
	if latestTarget != r.ID {
		t.Errorf("latest symlink points to %q, want %q", latestTarget, r.ID)
	}
}
