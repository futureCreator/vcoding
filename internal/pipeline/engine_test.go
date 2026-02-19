package pipeline

import (
	"testing"

	"github.com/epmk/vcoding/internal/config"
	"github.com/epmk/vcoding/internal/types"
)

func newTestEngine(roles config.RolesConfig) *Engine {
	return &Engine{
		Config: &config.Config{
			Roles: roles,
		},
	}
}

func TestStepDisplayModel(t *testing.T) {
	roles := config.RolesConfig{
		Planner:  "anthropic/claude-opus-4-6",
		Reviewer: "openai/gpt-4o",
		Editor:   "anthropic/claude-sonnet-4-6",
	}
	e := newTestEngine(roles)

	tests := []struct {
		name     string
		step     types.Step
		expected string
	}{
		{
			name:     "api executor with literal model",
			step:     types.Step{Executor: "api", Model: "openai/gpt-4o"},
			expected: "openai/gpt-4o",
		},
		{
			name:     "api executor with planner placeholder",
			step:     types.Step{Executor: "api", Model: "$planner"},
			expected: "anthropic/claude-opus-4-6",
		},
		{
			name:     "api executor with reviewer placeholder",
			step:     types.Step{Executor: "api", Model: "$reviewer"},
			expected: "openai/gpt-4o",
		},
		{
			name:     "api executor with empty model falls back to api",
			step:     types.Step{Executor: "api", Model: ""},
			expected: "api",
		},
		{
			name:     "unknown executor returns executor name",
			step:     types.Step{Executor: "custom-exec"},
			expected: "custom-exec",
		},
		{
			name:     "no executor returns dash",
			step:     types.Step{},
			expected: "—",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.stepDisplayModel(tt.step)
			if got != tt.expected {
				t.Errorf("stepDisplayModel() = %q, want %q", got, tt.expected)
			}
		})
	}
}
