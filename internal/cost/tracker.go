package cost

import "fmt"

// Usage holds token counts from an API response.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
}

// ModelPricing holds per-token pricing for a model (in USD per token).
type ModelPricing struct {
	InputPerToken  float64
	OutputPerToken float64
}

// defaultPricing provides fallback pricing for common models.
var defaultPricing = map[string]ModelPricing{
	"anthropic/claude-opus-4-6":   {InputPerToken: 15.0 / 1_000_000, OutputPerToken: 75.0 / 1_000_000},
	"anthropic/claude-sonnet-4-6": {InputPerToken: 3.0 / 1_000_000, OutputPerToken: 15.0 / 1_000_000},
	"moonshotai/kimi-k2.5":        {InputPerToken: 0.14 / 1_000_000, OutputPerToken: 0.14 / 1_000_000},
	"openai/codex-5.3":            {InputPerToken: 3.0 / 1_000_000, OutputPerToken: 15.0 / 1_000_000},
}

// FromHeader extracts cost from the x-openrouter-cost header value.
// Returns 0, false if the header is absent or unparseable.
func FromHeader(headerValue string) (float64, bool) {
	if headerValue == "" {
		return 0, false
	}
	var v float64
	if _, err := parseFloat(headerValue, &v); err != nil {
		return 0, false
	}
	return v, true
}

// FromUsage calculates cost from token usage and model pricing.
func FromUsage(model string, usage Usage) float64 {
	pricing, ok := defaultPricing[model]
	if !ok {
		return 0
	}
	return float64(usage.PromptTokens)*pricing.InputPerToken +
		float64(usage.CompletionTokens)*pricing.OutputPerToken
}

func parseFloat(s string, v *float64) (int, error) {
	_, err := fmt.Sscanf(s, "%f", v)
	return 1, err
}
