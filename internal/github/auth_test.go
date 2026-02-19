package github

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// mockRunner replaces ghRun with a function that returns the given output/error.
func mockRunner(out []byte, err error) func(args ...string) ([]byte, error) {
	return func(args ...string) ([]byte, error) {
		return out, err
	}
}

// notFoundError simulates exec.ErrNotFound wrapped the same way exec.Command does.
func notFoundError() error {
	return &exec.Error{Name: "gh", Err: exec.ErrNotFound}
}

func TestCheckGHVersion(t *testing.T) {
	cases := []struct {
		name    string
		output  string
		runErr  error
		wantErr string
	}{
		{
			name:   "success v2",
			output: "gh version 2.45.0 (2024-01-01)\nhttps://github.com/cli/cli/releases/tag/v2.45.0\n",
		},
		{
			name:   "success v3",
			output: "gh version 3.0.0 (2025-01-01)\n",
		},
		{
			name:    "not installed",
			runErr:  notFoundError(),
			wantErr: "not installed",
		},
		{
			name:    "version too old",
			output:  "gh version 1.14.0 (2021-06-01)\n",
			wantErr: "below the required minimum",
		},
		{
			name:    "unexpected output format",
			output:  "unexpected\n",
			wantErr: "unexpected 'gh --version' output",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ghRun = mockRunner([]byte(tc.output), tc.runErr)
			err := CheckGHVersion()
			if tc.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tc.wantErr)
				} else if !containsStr(err.Error(), tc.wantErr) {
					t.Errorf("expected error containing %q, got: %v", tc.wantErr, err)
				}
			}
		})
	}
}

func TestCheckGHAuth(t *testing.T) {
	cases := []struct {
		name    string
		host    string
		runErr  error
		wantErr string
	}{
		{name: "authenticated default host"},
		{name: "authenticated github.com", host: "github.com"},
		{name: "authenticated enterprise", host: "ghe.example.com"},
		{name: "not installed", runErr: notFoundError(), wantErr: "not installed"},
		{name: "not authenticated", runErr: fmt.Errorf("exit status 1"), wantErr: "not authenticated"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ghRun = mockRunner(nil, tc.runErr)
			err := CheckGHAuth(tc.host)
			if tc.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tc.wantErr)
				} else if !containsStr(err.Error(), tc.wantErr) {
					t.Errorf("expected error containing %q, got: %v", tc.wantErr, err)
				}
			}
		})
	}
}

func TestGetGHToken(t *testing.T) {
	cases := []struct {
		name      string
		output    string
		runErr    error
		wantErr   string
		wantToken string
	}{
		{
			name:      "valid ghp_ token",
			output:    "ghp_abcdefghijklmnopqrstuvwxyz012345678901\n",
			wantToken: "ghp_abcdefghijklmnopqrstuvwxyz012345678901",
		},
		{
			name:      "valid ghs_ token",
			output:    "ghs_ABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890\n",
			wantToken: "ghs_ABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890",
		},
		{
			name:    "not installed",
			runErr:  notFoundError(),
			wantErr: "not installed",
		},
		{
			name:    "not authenticated",
			runErr:  fmt.Errorf("exit status 1"),
			wantErr: "not authenticated",
		},
		{
			name:    "malformed token",
			output:  "not-a-real-token\n",
			wantErr: "unexpected format",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ResetTokenCache()
			ghRun = mockRunner([]byte(tc.output), tc.runErr)
			token, err := GetGHToken("")
			if tc.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if token != tc.wantToken {
					t.Errorf("expected token %q, got %q", tc.wantToken, token)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tc.wantErr)
				} else if !containsStr(err.Error(), tc.wantErr) {
					t.Errorf("expected error containing %q, got: %v", tc.wantErr, err)
				}
			}
		})
	}
}

func TestGetGHTokenCaching(t *testing.T) {
	ResetTokenCache()
	callCount := 0
	ghRun = func(args ...string) ([]byte, error) {
		callCount++
		return []byte("ghp_abcdefghijklmnopqrstuvwxyz012345678901\n"), nil
	}

	// First call should invoke ghRun.
	tok1, err := GetGHToken("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second call should return cached value without invoking ghRun again.
	tok2, err := GetGHToken("")
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}

	if tok1 != tok2 {
		t.Errorf("tokens differ: %q vs %q", tok1, tok2)
	}
	if callCount != 1 {
		t.Errorf("expected ghRun to be called once, got %d calls", callCount)
	}
}

func TestCheckGHAuthPassesHostname(t *testing.T) {
	var capturedArgs []string
	ghRun = func(args ...string) ([]byte, error) {
		capturedArgs = args
		return nil, nil
	}

	if err := CheckGHAuth("ghe.example.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for i, a := range capturedArgs {
		if a == "--hostname" && i+1 < len(capturedArgs) && capturedArgs[i+1] == "ghe.example.com" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected --hostname ghe.example.com in args, got %v", capturedArgs)
	}
}

func TestIsGhNotFound(t *testing.T) {
	if !isGhNotFound(&exec.Error{Name: "gh", Err: exec.ErrNotFound}) {
		t.Error("expected true for exec.ErrNotFound")
	}
	if isGhNotFound(errors.New("some other error")) {
		t.Error("expected false for non-ErrNotFound error")
	}
}

func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}
