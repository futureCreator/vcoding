package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate-config",
	Short: "Remove deprecated GitHub token fields from config files",
	Long: `migrate-config removes deprecated GitHub token configuration fields
(github_token and github.token) from your vcoding config files.

A backup of each modified file is created with a .bak extension before
any changes are written.

Authentication is now handled via the gh CLI. Run 'gh auth login' to
authenticate, or set GH_TOKEN in CI environments.`,
	RunE: runMigrateConfig,
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}

func runMigrateConfig(cmd *cobra.Command, args []string) error {
	var paths []string

	home, err := os.UserHomeDir()
	if err == nil {
		paths = append(paths, filepath.Join(home, ".vcoding", "config.yaml"))
	}
	paths = append(paths, filepath.Join(".vcoding", "config.yaml"))

	anyMigrated := false
	for _, path := range paths {
		migrated, err := migrateConfigFile(path)
		if err != nil {
			return fmt.Errorf("migrating %s: %w", path, err)
		}
		if migrated {
			anyMigrated = true
		}
	}

	if !anyMigrated {
		fmt.Println("No deprecated GitHub token fields found. Nothing to migrate.")
	}
	return nil
}

// migrateConfigFile removes deprecated token fields from a single config file.
// Returns true if the file was modified, false if it was not found or had no
// deprecated fields. An error is returned only for unexpected failures.
func migrateConfigFile(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return false, fmt.Errorf("parsing YAML: %w", err)
	}

	removed := []string{}

	if _, ok := raw["github_token"]; ok {
		delete(raw, "github_token")
		removed = append(removed, "github_token")
	}

	if gh, ok := raw["github"].(map[string]interface{}); ok {
		if _, ok := gh["token"]; ok {
			delete(gh, "token")
			removed = append(removed, "github.token")
		}
	}

	if len(removed) == 0 {
		return false, nil
	}

	// Write backup before modifying.
	backupPath := path + ".bak"
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return false, fmt.Errorf("creating backup %s: %w", backupPath, err)
	}

	updated, err := yaml.Marshal(raw)
	if err != nil {
		return false, fmt.Errorf("serializing YAML: %w", err)
	}

	if err := os.WriteFile(path, updated, 0644); err != nil {
		return false, fmt.Errorf("writing %s: %w", path, err)
	}

	fmt.Printf("Migrated %s\n", path)
	for _, field := range removed {
		fmt.Printf("  Removed deprecated field: %s\n", field)
	}
	fmt.Printf("  Backup saved to: %s\n", backupPath)
	fmt.Println("  Run 'gh auth login' to authenticate (or set GH_TOKEN in CI).")

	return true, nil
}
