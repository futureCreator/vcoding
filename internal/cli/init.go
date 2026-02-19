package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize vcoding configuration",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home dir: %w", err)
	}

	configDir := filepath.Join(home, ".vcoding")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config already exists: %s\n", configPath)
		return nil
	}

	defaultConfig := `# vcoding configuration
default_pipeline: default

provider:
  endpoint: https://openrouter.ai/api/v1
  api_key_env: OPENROUTER_API_KEY

roles:
  planner: anthropic/claude-opus-4-6
  reviewer: moonshotai/kimi-k2.5
  editor: anthropic/claude-sonnet-4-6
  auditor: openai/codex-5.3

github:
  base_branch: main

executors:
  claude-code:
    command: claude
    args: ["-p", "--output-format", "json"]
    timeout: 300s

language:
  artifacts: en
  normalize_ticket: true

project_context:
  max_files: 20
  max_file_size: 50KB
  include_patterns: ["*.go", "*.rs", "*.ts", "*.py", "*.md"]
  exclude_patterns: ["vendor/", "node_modules/", ".git/"]

max_context_tokens: 80000
log_level: info
`

	if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("Created %s\n", configPath)
	fmt.Println("Edit the file to set your API keys and preferences.")
	fmt.Println("Set OPENROUTER_API_KEY environment variable for API access.")
	return nil
}
