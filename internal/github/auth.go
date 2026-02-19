package github

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ghRun executes a gh subcommand and returns combined stdout output.
// It is a package-level variable so tests can replace it with a mock.
var ghRun = func(args ...string) ([]byte, error) {
	return exec.Command("gh", args...).Output()
}

// CheckGHVersion verifies that the gh CLI is installed and at version >= 2.0.0.
func CheckGHVersion() error {
	out, err := ghRun("--version")
	if err != nil {
		if isGhNotFound(err) {
			return fmt.Errorf("'gh' CLI is not installed. Install it from https://cli.github.com/ and run 'gh auth login'")
		}
		return fmt.Errorf("checking gh version: %w", err)
	}

	// Output format: "gh version 2.x.y (...)\n..."
	line := strings.SplitN(string(out), "\n", 2)[0]
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return fmt.Errorf("unexpected 'gh --version' output: %q", line)
	}

	ver := strings.TrimPrefix(parts[2], "v")
	var major int
	if _, err := fmt.Sscanf(ver, "%d.", &major); err != nil {
		return fmt.Errorf("could not parse gh version from %q", ver)
	}
	if major < 2 {
		return fmt.Errorf("'gh' CLI version %s is below the required minimum 2.0.0. Upgrade from https://cli.github.com/", ver)
	}
	return nil
}

// CheckGHAuth verifies that the gh CLI is authenticated for the given host.
// Pass an empty string or "github.com" for the default host.
func CheckGHAuth(host string) error {
	args := []string{"auth", "status"}
	if host != "" && host != "github.com" {
		args = append(args, "--hostname", host)
	}

	_, err := ghRun(args...)
	if err != nil {
		if isGhNotFound(err) {
			return fmt.Errorf("'gh' CLI is not installed. Install it from https://cli.github.com/ and run 'gh auth login'")
		}
		return fmt.Errorf("'gh' CLI is not authenticated. Run 'gh auth login' to authenticate (or set GH_TOKEN in CI)")
	}
	return nil
}

// RunPreflight runs all gh CLI prerequisite checks in order:
// version check → auth check.
// Pass an empty string or "github.com" for the default host.
func RunPreflight(host string) error {
	if err := CheckGHVersion(); err != nil {
		return err
	}
	return CheckGHAuth(host)
}

// isGhNotFound returns true when err indicates that the gh binary was not found.
func isGhNotFound(err error) bool {
	return errors.Is(err, exec.ErrNotFound)
}
