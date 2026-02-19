package cli

import (
	"github.com/epmk/vcoding/internal/source"
	"github.com/spf13/cobra"
)

var doPipeline string
var doForce bool

var doCmd = &cobra.Command{
	Use:          "do <spec-file>",
	Short:        "Run pipeline on a spec file",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		src := &source.SpecSource{Path: args[0]}
		return runPipeline(cmd.Context(), src, doPipeline, doForce)
	},
}

func init() {
	doCmd.Flags().StringVarP(&doPipeline, "pipeline", "p", "default", "Pipeline to use")
	doCmd.Flags().BoolVar(&doForce, "force", false, "Skip dirty working tree check")
}
