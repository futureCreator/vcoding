package pipeline

import (
	"testing"
)

func TestParse(t *testing.T) {
	yml := `
name: test
steps:
  - name: Plan
    executor: api
    model: anthropic/claude-opus-4-6
    prompt_template: plan
    input: [TICKET.md]
    output: PLAN.md
  - name: Test
    executor: shell
    command: "go test ./..."
    output: TEST.md
`
	p, err := Parse([]byte(yml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "test" {
		t.Errorf("expected name 'test', got %q", p.Name)
	}
	if len(p.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(p.Steps))
	}
	if p.Steps[0].Executor != "api" {
		t.Errorf("expected executor 'api', got %q", p.Steps[0].Executor)
	}
	if p.Steps[1].Command != "go test ./..." {
		t.Errorf("expected command 'go test ./...', got %q", p.Steps[1].Command)
	}
}

func TestParseNoName(t *testing.T) {
	yml := `steps: []`
	_, err := Parse([]byte(yml))
	if err == nil {
		t.Error("expected error for pipeline without name")
	}
}
