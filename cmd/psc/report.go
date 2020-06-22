package main

import (
	"github.com/anishathalye/periscope"

	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:                   "report",
	Short:                 "Report scan results",
	DisableFlagsInUseLine: true,
	Args:                  cobra.NoArgs,
	RunE:                  reportRun,
}

func init() {
	rootCmd.AddCommand(reportCmd)
}

func reportRun(cmd *cobra.Command, _ []string) error {
	ps, err := periscope.New(&periscope.Options{
		Debug: rootFlags.debug,
	})
	if err != nil {
		return err
	}
	return ps.Report(&periscope.ReportOptions{})
}
