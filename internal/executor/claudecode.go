package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/epmk/vcoding/internal/config"
)

// ClaudeCodeExecutor delegates implementation to the claude CLI.
type ClaudeCodeExecutor struct {
	Config  *config.Config
	Prompts map[string]string // template name → content (same as APIExecutor)
}

type claudeCodeOutput struct {
	Result string `json:"result"`
}

func (e *ClaudeCodeExecutor) Execute(ctx context.Context, req *Request) (*Result, error) {
	start := time.Now()

	prompt := buildUserContent(req)

	entry := e.Config.Executors.ClaudeCode
	cmdName := entry.Command
	if cmdName == "" {
		cmdName = "claude"
	}

	// Always build args programmatically so model and system-prompt are never missing.
	args := []string{"-p", "--output-format", "json", "--dangerously-skip-permissions"}

	if req.Step.Model != "" {
		args = append(args, "--model", req.Step.Model)
	}

	if req.Step.PromptTemplate != "" && e.Prompts != nil {
		if content, ok := e.Prompts[req.Step.PromptTemplate]; ok && content != "" {
			args = append(args, "--system-prompt", content)
		}
	}

	timeout, err := parseTimeout(entry.Timeout)
	if err != nil {
		timeout = 1800 * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, cmdName, args...)
	cmd.Stdin = bytes.NewBufferString(prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	if req.Verbose {
		// Stream stderr to the terminal so the user can see claude's progress.
		// Stdout is still captured for JSON result parsing.
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stderr = &stderr
	}

	if err := cmd.Run(); err != nil {
		errDetail := stderr.String()
		if req.Verbose {
			errDetail = "(see stderr above)"
		}
		return nil, fmt.Errorf("claude-code executor: %w\nstderr: %s", err, errDetail)
	}

	output := stdout.String()

	// Try to parse JSON output from claude -p --output-format json
	var parsed claudeCodeOutput
	if err := json.Unmarshal([]byte(output), &parsed); err == nil && parsed.Result != "" {
		output = parsed.Result
	}

	return &Result{
		Output:   strings.TrimSpace(output),
		Duration: time.Since(start),
	}, nil
}

func parseTimeout(s string) (time.Duration, error) {
	if s == "" {
		return 1800 * time.Second, nil
	}
	return time.ParseDuration(s)
}
