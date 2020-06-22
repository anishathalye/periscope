package main

import (
	"github.com/anishathalye/periscope"

	"github.com/spf13/cobra"
)

var summaryCmd = &cobra.Command{
	Use:                   "summary",
	Short:                 "Report scan result summary",
	DisableFlagsInUseLine: true,
	Args:                  cobra.NoArgs,
	RunE:                  summaryRun,
}

func init() {
	rootCmd.AddCommand(summaryCmd)
}

func summaryRun(cmd *cobra.Command, _ []string) error {
	ps, err := periscope.New(&periscope.Options{
		Debug: rootFlags.debug,
	})
	if err != nil {
		return err
	}
	return ps.Summary(&periscope.SummaryOptions{})
}
