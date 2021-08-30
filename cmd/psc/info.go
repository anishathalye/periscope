package main

import (
	"github.com/anishathalye/periscope/internal/periscope"

	"github.com/spf13/cobra"
)

var infoFlags struct {
	relative bool
}

var infoCmd = &cobra.Command{
	Use:               "info path ...",
	Short:             "Inspect a file",
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: infoValidArgs,
	RunE:              infoRun,
}

func init() {
	infoCmd.Flags().BoolVarP(&infoFlags.relative, "relative", "r", false, "show duplicates using relative paths")
	rootCmd.AddCommand(infoCmd)
}

func infoValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveDefault
}

func infoRun(cmd *cobra.Command, paths []string) error {
	ps, err := periscope.New(&periscope.Options{
		Debug: rootFlags.debug,
	})
	if err != nil {
		return err
	}
	options := &periscope.InfoOptions{
		Relative: infoFlags.relative,
	}
	return ps.Info(paths, options)
}
