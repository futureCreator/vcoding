package cli

import (
	"strings"

	"github.com/futureCreator/vcoding/internal/source"
	"github.com/spf13/cobra"
)

var askPipeline string
var askVerbose bool

var askCmd = &cobra.Command{
	Use:          "ask <message>",
	Short:        "Run pipeline from a direct message/prompt",
	Long:         "Run the planning pipeline using a direct user message as input.\nThe message will be treated as a task description and processed through the same workflow as issues or spec files.",
	Example:      "vcoding ask \"Implement user authentication with JWT tokens\"",
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		prompt := strings.Join(args, " ")
		src := &source.PromptSource{Prompt: prompt}
		return runPipeline(cmd.Context(), src, askPipeline, askVerbose)
	},
}

func init() {
	rootCmd.AddCommand(askCmd)
	askCmd.Flags().StringVarP(&askPipeline, "pipeline", "p", "default", "Pipeline to use")
	askCmd.Flags().BoolVarP(&askVerbose, "verbose", "v", false, "Stream executor output to terminal")
}
