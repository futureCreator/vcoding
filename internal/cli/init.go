package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/epmk/vcoding/internal/assets"
	"github.com/spf13/cobra"
)

var initMinimal bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize vcoding configuration and project convention files",
	Long: `Initialize vcoding by creating ~/.vcoding/config.yaml and project-level
convention files (CLAUDE.md, .cursorrules, AGENTS.md) in the current directory.

By default the generated config includes inline comments explaining each field.
Use --minimal to generate a comment-free config version.`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVar(&initMinimal, "minimal", false, "Generate config without inline comments")
}

func runInit(cmd *cobra.Command, args []string) error {
	if err := initGlobalConfig(); err != nil {
		return err
	}
	return initConventionFiles()
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

// initConventionFiles writes project-level AI convention files to the current directory.
// Existing files are overwritten with the latest template content.
func initConventionFiles() error {
	files, err := assets.ConventionFiles()
	if err != nil {
		return fmt.Errorf("loading convention templates: %w", err)
	}

	for _, name := range []string{"CLAUDE.md", ".cursorrules", "AGENTS.md"} {
		content := files[name]
		_, statErr := os.Stat(name)
		exists := statErr == nil

		if exists {
			f, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("opening %s: %w", name, err)
			}
			_, writeErr := fmt.Fprintf(f, "\n%s", content)
			closeErr := f.Close()
			if writeErr != nil {
				return fmt.Errorf("appending to %s: %w", name, writeErr)
			}
			if closeErr != nil {
				return fmt.Errorf("closing %s: %w", name, closeErr)
			}
			fmt.Printf("Appended to %s\n", name)
		} else {
			if err := os.WriteFile(name, []byte(content), 0644); err != nil {
				return fmt.Errorf("writing %s: %w", name, err)
			}
			fmt.Printf("Created %s\n", name)
		}
	}
	return nil
}
