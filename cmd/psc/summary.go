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
	ValidArgsFunction:     summaryValidArgs,
	RunE:                  summaryRun,
}

func init() {
	rootCmd.AddCommand(summaryCmd)
}

func summaryValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
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
