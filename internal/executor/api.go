package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/futureCreator/vcoding/internal/config"
	"github.com/futureCreator/vcoding/internal/cost"
	vlog "github.com/futureCreator/vcoding/internal/log"
)

// APIExecutor calls the OpenRouter API (OpenAI-compatible).
type APIExecutor struct {
	Config     *config.Config
	Prompts    map[string]string // template name → content
	HTTPClient *http.Client
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (e *APIExecutor) Execute(ctx context.Context, req *Request) (*Result, error) {
	start := time.Now()

	systemPrompt, err := e.resolvePrompt(req.Step.PromptTemplate)
	if err != nil {
		return nil, err
	}

	userContent := buildUserContent(req)

	model := req.Step.Model
	if model == "" {
		model = e.Config.Roles.Planner
	}

	payload := chatRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	endpoint := e.Config.Provider.Endpoint + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+e.Config.APIKey())

	client := e.HTTPClient
	if client == nil {
		timeout := 300 * time.Second
		if e.Config.Provider.APITimeout != "" {
			if d, err := time.ParseDuration(e.Config.Provider.APITimeout); err == nil {
				timeout = d
			}
		}
		client = &http.Client{Timeout: timeout}
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("empty choices in API response")
	}

	output := chatResp.Choices[0].Message.Content

	// Cost extraction: header > usage > 0+warn
	var apiCost float64
	if c, ok := cost.FromHeader(resp.Header.Get("x-openrouter-cost")); ok {
		apiCost = c
	} else if chatResp.Usage.PromptTokens > 0 {
		apiCost = cost.FromUsage(model, cost.Usage{
			PromptTokens:     chatResp.Usage.PromptTokens,
			CompletionTokens: chatResp.Usage.CompletionTokens,
		})
	} else {
		vlog.Warn("could not determine cost for step", "model", model)
	}

	return &Result{
		Output:    output,
		Cost:      apiCost,
		Duration:  time.Since(start),
		TokensIn:  chatResp.Usage.PromptTokens,
		TokensOut: chatResp.Usage.CompletionTokens,
	}, nil
}

func (e *APIExecutor) resolvePrompt(template string) (string, error) {
	if template == "" {
		return "", nil
	}
	if content, ok := e.Prompts[template]; ok {
		return content, nil
	}
	return "", fmt.Errorf("prompt template %q not found", template)
}

// ResolvePrompt exposes prompt lookup for external callers (e.g. token budget accounting).
func (e *APIExecutor) ResolvePrompt(name string) (string, bool) {
	content, ok := e.Prompts[name]
	return content, ok
}

// diffKeys are virtual input keys that should be rendered as diff code blocks.
var diffKeys = map[string]string{
	"git:diff": "git diff",
}

func buildUserContent(req *Request) string {
	var sb strings.Builder
	// Sort keys for deterministic output
	keys := make([]string, 0, len(req.InputFiles))
	for k := range req.InputFiles {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, name := range keys {
		content := req.InputFiles[name]
		if label, isDiff := diffKeys[name]; isDiff {
			if content != "" {
				sb.WriteString(fmt.Sprintf("## %s\n\n```diff\n%s\n```\n\n", label, content))
			}
		} else {
			sb.WriteString(fmt.Sprintf("## %s\n\n%s\n\n", name, content))
		}
	}
	return sb.String()
}
