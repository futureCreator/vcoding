package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration structure.
type Config struct {
	DefaultPipeline  string           `yaml:"default_pipeline"`
	Provider         ProviderConfig   `yaml:"provider"`
	Roles            RolesConfig      `yaml:"roles"`
	GitHub           GitHubConfig     `yaml:"github"`
	Executors        ExecutorsConfig  `yaml:"executors"`
	Language         LanguageConfig   `yaml:"language"`
	ProjectContext   ProjectCtxConfig `yaml:"project_context"`
	MaxContextTokens int              `yaml:"max_context_tokens"`
	LogLevel         string           `yaml:"log_level"`
}

type ProviderConfig struct {
	Endpoint   string `yaml:"endpoint"`
	APIKeyEnv  string `yaml:"api_key_env"`
	APITimeout string `yaml:"api_timeout"`
}

type RolesConfig struct {
	Planner  string `yaml:"planner"`
	Reviewer string `yaml:"reviewer"`
	Editor   string `yaml:"editor"`
	Auditor  string `yaml:"auditor"`
}

type GitHubConfig struct {
	DefaultRepo string `yaml:"default_repo"`
	BaseBranch  string `yaml:"base_branch"`
}

type ExecutorEntry struct {
	Command string `yaml:"command"`
	Timeout string `yaml:"timeout"`
}

type ExecutorsConfig struct {
	ClaudeCode ExecutorEntry `yaml:"claude-code"`
}

type LanguageConfig struct {
	Artifacts       string `yaml:"artifacts"`
	NormalizeTicket bool   `yaml:"normalize_ticket"`
}

type ProjectCtxConfig struct {
	MaxFiles        int      `yaml:"max_files"`
	MaxFileSize     string   `yaml:"max_file_size"`
	IncludePatterns []string `yaml:"include_patterns"`
	ExcludePatterns []string `yaml:"exclude_patterns"`
}

// Validate checks that required fields are present.
func (c *Config) Validate() error {
	if c.DefaultPipeline == "" {
		return fmt.Errorf("default_pipeline is required")
	}
	if c.Provider.Endpoint == "" {
		return fmt.Errorf("provider.endpoint is required")
	}
	return nil
}

// APIKey returns the resolved OpenRouter API key.
func (c *Config) APIKey() string {
	if c.Provider.APIKeyEnv == "" {
		return os.Getenv("OPENROUTER_API_KEY")
	}
	return os.Getenv(c.Provider.APIKeyEnv)
}

// Load resolves config from project → user → defaults.
func Load() (*Config, error) {
	cfg := defaults()

	// user-level config
	home, err := os.UserHomeDir()
	if err == nil {
		userPath := filepath.Join(home, ".vcoding", "config.yaml")
		if err := mergeFile(cfg, userPath); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("loading user config: %w", err)
		}
	}

	// project-level config (highest priority)
	projectPath := filepath.Join(".vcoding", "config.yaml")
	if err := mergeFile(cfg, projectPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading project config: %w", err)
	}

	return cfg, nil
}

func mergeFile(dst *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	// Check for deprecated github_token field before merging.
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err == nil {
		if gh, ok := raw["github"].(map[string]interface{}); ok {
			if _, hasToken := gh["token"]; hasToken {
				return fmt.Errorf("configuration field 'github.token' is no longer supported. " +
					"Please remove it from %s and authenticate via `gh auth login` (or set GH_TOKEN in CI). " +
					"See README for migration details.", path)
			}
		}
		if _, hasToken := raw["github_token"]; hasToken {
			return fmt.Errorf("configuration field 'github_token' is no longer supported. " +
				"Please remove it from %s and authenticate via `gh auth login` (or set GH_TOKEN in CI). " +
				"See README for migration details.", path)
		}
	}
	return yaml.Unmarshal(data, dst)
}

func defaults() *Config {
	return &Config{
		DefaultPipeline: "default",
		Provider: ProviderConfig{
			Endpoint:   "https://openrouter.ai/api/v1",
			APIKeyEnv:  "OPENROUTER_API_KEY",
			APITimeout: "300s",
		},
		Roles: RolesConfig{
			Planner:  "anthropic/claude-opus-4-6",
			Reviewer: "deepseek/deepseek-r1",
			Editor:   "z-ai/glm-5",
			Auditor:  "openai/gpt-5.2-codex",
		},
		GitHub: GitHubConfig{
			BaseBranch: "main",
		},
		Executors: ExecutorsConfig{
			// Args here are appended AFTER the required flags (-p, --output-format json,
			// --dangerously-skip-permissions, --model, --system-prompt) that the executor
			// always builds programmatically. Use this only for extra flags (e.g. --verbose).
			ClaudeCode: ExecutorEntry{
				Command: "claude",
				Timeout: "300s",
			},
		},
		Language: LanguageConfig{
			Artifacts:       "en",
			NormalizeTicket: true,
		},
		ProjectContext: ProjectCtxConfig{
			MaxFiles:        20,
			MaxFileSize:     "50KB",
			IncludePatterns: []string{"*.go", "*.rs", "*.ts", "*.py", "*.md"},
			ExcludePatterns: []string{"vendor/", "node_modules/", ".git/", ".vcoding/"},
		},
		MaxContextTokens: 80000,
		LogLevel:         "info",
	}
}
