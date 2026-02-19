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
	Config *config.Config
}

type claudeCodeOutput struct {
	Result string `json:"result"`
}

func (e *ClaudeCodeExecutor) Execute(ctx context.Context, req *Request) (*Result, error) {
	start := time.Now()

	// If a system prompt is provided, prepend it to the user content.
	userContent := buildUserContent(req)
	var prompt string
	if req.SystemPrompt != "" {
		prompt = req.SystemPrompt + "\n\n---\n\n" + userContent
	} else {
		prompt = userContent
	}

	entry := e.Config.Executors.ClaudeCode
	cmdName := entry.Command
	if cmdName == "" {
		cmdName = "claude"
	}
	args := entry.Args
	if len(args) == 0 {
		args = []string{"-p", "--output-format", "json", "--dangerously-skip-permissions"}
	}

	// Inject --model flag if a model is specified and not already in args.
	if req.Step.Model != "" && !containsArg(args, "--model") {
		args = append(args, "--model", req.Step.Model)
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

// containsArg reports whether args contains the given flag string.
func containsArg(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}
