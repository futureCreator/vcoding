package cli

import (
	"fmt"

	"github.com/futureCreator/vcoding/pkg/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vcoding",
	Short: "Multi-model issue-to-PR pipeline CLI",
	Long:  `vCoding orchestrates multiple AI models to take an issue or spec from input to PR automatically.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(pickCmd)
	rootCmd.AddCommand(doCmd)
	rootCmd.AddCommand(statsCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("vcoding %s\n", version.Version)
	},
}
