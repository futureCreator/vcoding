package github

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

// ghRun executes a gh subcommand and returns combined stdout output.
// It is a package-level variable so tests can replace it with a mock.
var ghRun = func(args ...string) ([]byte, error) {
	return exec.Command("gh", args...).Output()
}

var (
	tokenCache   string
	tokenCacheMu sync.Mutex
)

// tokenPattern matches well-known GitHub token prefixes followed by at least 36 alphanumeric chars.
var tokenPattern = regexp.MustCompile(`^(ghp_|ghs_|gho_|ghu_)[a-zA-Z0-9]{36,}$`)

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

// GetGHToken retrieves a GitHub token via gh auth token and caches it for the
// process lifetime. Subsequent calls return the cached token without spawning a
// new subprocess. Thread-safe.
func GetGHToken(host string) (string, error) {
	tokenCacheMu.Lock()
	defer tokenCacheMu.Unlock()

	if tokenCache != "" {
		return tokenCache, nil
	}

	args := []string{"auth", "token"}
	if host != "" && host != "github.com" {
		args = append(args, "--hostname", host)
	}

	out, err := ghRun(args...)
	if err != nil {
		if isGhNotFound(err) {
			return "", fmt.Errorf("'gh' CLI is not installed. Install it from https://cli.github.com/ and run 'gh auth login'")
		}
		return "", fmt.Errorf("'gh' CLI is not authenticated. Run 'gh auth login' to authenticate (or set GH_TOKEN in CI)")
	}

	token := strings.TrimSpace(string(out))
	if !tokenPattern.MatchString(token) {
		return "", fmt.Errorf("token returned by 'gh auth token' has an unexpected format. Ensure 'gh' is up to date")
	}

	tokenCache = token
	return token, nil
}

// RunPreflight runs all gh CLI prerequisite checks in order:
// version check → auth check → token retrieval (warms the cache).
// Pass an empty string or "github.com" for the default host.
func RunPreflight(host string) error {
	if err := CheckGHVersion(); err != nil {
		return err
	}
	if err := CheckGHAuth(host); err != nil {
		return err
	}
	if _, err := GetGHToken(host); err != nil {
		return err
	}
	return nil
}

// ResetTokenCache clears the cached token. Used in tests.
func ResetTokenCache() {
	tokenCacheMu.Lock()
	defer tokenCacheMu.Unlock()
	tokenCache = ""
}

// isGhNotFound returns true when err indicates that the gh binary was not found.
func isGhNotFound(err error) bool {
	return errors.Is(err, exec.ErrNotFound)
}
