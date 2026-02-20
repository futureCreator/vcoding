package cli

import (
	"github.com/futureCreator/vcoding/internal/source"
	"github.com/spf13/cobra"
)

var doPipeline string
var doVerbose bool

var doCmd = &cobra.Command{
	Use:          "do <spec-file>",
	Short:        "Run pipeline on a spec file",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		src := &source.SpecSource{Path: args[0]}
		return runPipeline(cmd.Context(), src, doPipeline, doVerbose)
	},
}

func init() {
	doCmd.Flags().StringVarP(&doPipeline, "pipeline", "p", "default", "Pipeline to use")
	doCmd.Flags().BoolVarP(&doVerbose, "verbose", "v", false, "Stream executor output to terminal")
}
