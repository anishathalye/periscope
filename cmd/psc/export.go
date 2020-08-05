package main

import (
	"github.com/anishathalye/periscope"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:                   "export",
	Short:                 "Export scan results",
	DisableFlagsInUseLine: true,
	Args:                  cobra.NoArgs,
	ValidArgsFunction:     exportValidArgs,
	RunE:                  exportRun,
}

func init() {
	rootCmd.AddCommand(exportCmd)
}

func exportValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func exportRun(cmd *cobra.Command, _ []string) error {
	ps, err := periscope.New(&periscope.Options{
		Debug: rootFlags.debug,
	})
	if err != nil {
		return err
	}
	return ps.Export(&periscope.ExportOptions{Format: periscope.JsonFormat})
}
