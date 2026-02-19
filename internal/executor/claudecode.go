package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

	prompt := buildUserContent(req)

	entry := e.Config.Executors.ClaudeCode
	cmdName := entry.Command
	if cmdName == "" {
		cmdName = "claude"
	}
	args := entry.Args
	if len(args) == 0 {
		args = []string{"-p", "--output-format", "json"}
	}

	timeout, err := parseTimeout(entry.Timeout)
	if err != nil {
		timeout = 300 * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, cmdName, args...)
	cmd.Stdin = bytes.NewBufferString(prompt)
	cmd.Dir = req.RunDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("claude-code executor: %w\nstderr: %s", err, stderr.String())
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
		return 300 * time.Second, nil
	}
	return time.ParseDuration(s)
}
