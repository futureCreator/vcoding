package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/futureCreator/vcoding/internal/assets"
	"github.com/futureCreator/vcoding/pkg/version"
	"github.com/spf13/cobra"
)

var initMinimal bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize vcoding configuration",
	Long: `Initialize vcoding by creating ~/.vcoding/config.yaml and generating project
agent instruction files (CLAUDE.md, AGENTS.md, SKILL.md) in the current directory.

By default the generated config includes inline comments explaining each field.
Use --minimal to generate a comment-free config version.`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVar(&initMinimal, "minimal", false, "Generate config without inline comments")
}

func runInit(cmd *cobra.Command, args []string) error {
	if err := checkGitAndGH(); err != nil {
		return err
	}
	if err := initGlobalConfig(); err != nil {
		return err
	}
	return initProjectFiles()
}

// initGlobalConfig creates ~/.vcoding/config.yaml if it does not exist.
func initGlobalConfig() error {
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

	templateFilename := "config.yaml"
	if initMinimal {
		templateFilename = "config.minimal.yaml"
	}

	content, err := assets.LoadTemplate(templateFilename)
	if err != nil {
		return fmt.Errorf("failed to load template %q: %w (this may indicate a corrupted installation)", templateFilename, err)
	}

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("Created %s\n", configPath)
	fmt.Println("Edit the file to set your API keys and preferences.")
	fmt.Println("Set OPENROUTER_API_KEY environment variable for API access.")
	return nil
}

// initProjectFiles generates CLAUDE.md, AGENTS.md, and SKILL.md in the project root.
func initProjectFiles() error {
	data := struct{ Version string }{Version: version.Version}

	for _, filename := range []string{"CLAUDE.md", "AGENTS.md", "SKILL.md"} {
		if _, err := os.Stat(filename); err == nil {
			fmt.Printf("%s already exists, skipping\n", filename)
			continue
		}

		content, err := assets.RenderTemplate(filename, data)
		if err != nil {
			return fmt.Errorf("failed to render template %q: %w (this may indicate a corrupted installation)", filename, err)
		}

		// Write with 0644 permissions (world-readable, no execute bit)
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", filename, err)
		}

		fmt.Printf("Created %s\n", filename)
	}

	return nil
}
