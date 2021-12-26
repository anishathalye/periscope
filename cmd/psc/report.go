package main

import (
	"github.com/anishathalye/periscope/internal/periscope"

	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:                   "report [path]",
	Short:                 "Report scan results",
	DisableFlagsInUseLine: true,
	Args:                  cobra.MaximumNArgs(1),
	ValidArgsFunction:     reportValidArgs,
	RunE:                  reportRun,
}

func init() {
	rootCmd.AddCommand(reportCmd)
}

func reportValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveFilterDirs
}

func reportRun(cmd *cobra.Command, paths []string) error {
	ps, err := periscope.New(&periscope.Options{
		Debug: rootFlags.debug,
	})
	if err != nil {
		return err
	}
	var path string
	if len(paths) == 1 {
		path = paths[0]
	}
	return ps.Report(path, &periscope.ReportOptions{})
}
