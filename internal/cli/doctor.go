package cli

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/epmk/vcoding/internal/config"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check vcoding prerequisites and configuration",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	allOK := runChecks(true, true)
	fmt.Println()
	if allOK {
		fmt.Println("All checks passed. vcoding is ready.")
	} else {
		fmt.Println("Some checks failed. Fix the issues above before running vcoding.")
	}
	return nil
}

// checkGitAndGH runs only git and gh checks (no config).
// Used by init, which creates the config and doesn't require it upfront.
func checkGitAndGH() error {
	if !runChecks(false, false) {
		return fmt.Errorf("prerequisite checks failed; run `vcoding doctor` for details")
	}
	return nil
}

// checkPrerequisites runs all checks including config and API key.
// Used by pick/do before starting a pipeline run.
func checkPrerequisites() error {
	if !runChecks(false, true) {
		return fmt.Errorf("prerequisite checks failed; run `vcoding doctor` for details")
	}
	return nil
}

// runChecks performs prerequisite checks and prints results.
// If verbose is true, prints both passed (✅) and failed (❌) checks.
// If verbose is false, prints only failed checks.
// If includeConfig is true, also checks config and API key.
// Returns true if all checks passed.
func runChecks(verbose, includeConfig bool) bool {
	allOK := true

	check := func(label string, ok bool, hint string) {
		if ok {
			if verbose {
				fmt.Printf("✅ %s\n", label)
			}
		} else {
			fmt.Printf("❌ %s — %s\n", label, hint)
			allOK = false
		}
	}

	// 1. git repo
	_, err := exec.LookPath("git")
	check("git installed", err == nil, "install git")
	gitErr := exec.Command("git", "rev-parse", "--is-inside-work-tree").Run()
	check("inside git repository", gitErr == nil, "run `git init` or cd to a git repo")

	// 2. gh CLI
	_, err = exec.LookPath("gh")
	check("gh CLI installed", err == nil, "install gh: https://cli.github.com")
	if err == nil {
		ghVersionOK, ghVersionHint := checkGHVersion()
		check("gh CLI version >= 2.0.0", ghVersionOK, ghVersionHint)
		ghAuthErr := exec.Command("gh", "auth", "status").Run()
		check("gh CLI authenticated", ghAuthErr == nil, "run `gh auth login` (or set GH_TOKEN in CI)")
	}

	// 3. config (optional)
	if includeConfig {
		cfg, cfgErr := config.Load()
		check("config loadable", cfgErr == nil, fmt.Sprintf("fix config: %v", cfgErr))
		if cfgErr == nil {
			validateErr := cfg.Validate()
			check("config valid", validateErr == nil, fmt.Sprintf("%v", validateErr))

			apiKey := cfg.APIKey()
			check("OPENROUTER_API_KEY set", apiKey != "", "set environment variable OPENROUTER_API_KEY")
		}
	}

	return allOK
}

// checkGHVersion returns true if gh version >= 2.0.0.
func checkGHVersion() (bool, string) {
	out, err := exec.Command("gh", "--version").Output()
	if err != nil {
		return false, "could not determine gh version"
	}
	// Output format: "gh version 2.x.y (...)..."
	line := strings.SplitN(string(out), "\n", 2)[0]
	parts := strings.Fields(line)
	// parts: ["gh", "version", "2.x.y", ...]
	if len(parts) < 3 {
		return false, fmt.Sprintf("unexpected gh --version output: %q", line)
	}
	ver := strings.TrimPrefix(parts[2], "v")
	var major int
	if _, err := fmt.Sscanf(ver, "%d.", &major); err != nil {
		return false, fmt.Sprintf("could not parse gh version %q", ver)
	}
	if major < 2 {
		return false, fmt.Sprintf("gh version %s is too old; upgrade from https://cli.github.com/", ver)
	}
	return true, ""
}
