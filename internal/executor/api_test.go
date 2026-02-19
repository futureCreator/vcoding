package executor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/epmk/vcoding/internal/config"
	"github.com/epmk/vcoding/internal/types"
)

func TestAPIExecutorMock(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("x-openrouter-cost", "0.05")
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "# PLAN\n\nStep 1: do something"}},
			},
			"usage": map[string]int{"prompt_tokens": 100, "completion_tokens": 50},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	cfg := config.Config{
		Provider: config.ProviderConfig{
			Endpoint:  ts.URL,
			APIKeyEnv: "",
		},
	}

	exec := &APIExecutor{
		Config:     &cfg,
		Prompts:    map[string]string{"plan": "You are a planner."},
		HTTPClient: ts.Client(),
	}

	req := &Request{
		Step: types.Step{
			Name:           "Plan",
			Executor:       "api",
			Model:          "test/model",
			PromptTemplate: "plan",
			Input:          []string{"TICKET.md"},
		},
		InputFiles: map[string]string{"TICKET.md": "Fix the auth bug"},
	}

	result, err := exec.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if result.Cost != 0.05 {
		t.Errorf("expected cost 0.05 from header, got %f", result.Cost)
	}
	if result.TokensIn != 100 {
		t.Errorf("expected 100 prompt tokens, got %d", result.TokensIn)
	}
}

