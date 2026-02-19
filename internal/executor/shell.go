package executor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// ShellExecutor runs a shell command and captures its output.
type ShellExecutor struct {
	// ProjectDir is the working directory for shell commands.
	// If empty, the current process working directory is used.
	ProjectDir string
}

func (e *ShellExecutor) Execute(ctx context.Context, req *Request) (*Result, error) {
	start := time.Now()

	command := req.Step.Command
	if command == "" {
		return nil, fmt.Errorf("shell executor: no command specified for step %q", req.Step.Name)
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	// Determine working directory: step override > executor default > cwd
	switch {
	case req.Step.WorkDir != "":
		cmd.Dir = req.Step.WorkDir
	case e.ProjectDir != "":
		cmd.Dir = e.ProjectDir
	default:
		if cwd, err := os.Getwd(); err == nil {
			cmd.Dir = cwd
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n--- stderr ---\n" + stderr.String()
	}

	if err != nil {
		return nil, fmt.Errorf("shell command %q failed: %w\noutput: %s", command, err, output)
	}

	return &Result{
		Output:   output,
		Duration: time.Since(start),
	}, nil
}
