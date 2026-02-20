package cli

import (
	"github.com/futureCreator/vcoding/internal/source"
	"github.com/spf13/cobra"
)

var pickPipeline string
var pickVerbose bool

var pickCmd = &cobra.Command{
	Use:          "pick <issue-number>",
	Short:        "Run pipeline on a GitHub issue",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		src := &source.GitHubSource{IssueNumber: args[0]}
		return runPipeline(cmd.Context(), src, pickPipeline, pickVerbose)
	},
}

func init() {
	pickCmd.Flags().StringVarP(&pickPipeline, "pipeline", "p", "default", "Pipeline to use")
	pickCmd.Flags().BoolVarP(&pickVerbose, "verbose", "v", false, "Stream executor output to terminal")
}
