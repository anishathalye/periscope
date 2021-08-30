package main

import (
	"github.com/anishathalye/periscope/internal/periscope"

	"github.com/spf13/cobra"
)

var refreshCmd = &cobra.Command{
	Use:                   "refresh",
	Short:                 "Remove deleted files from database",
	DisableFlagsInUseLine: true,
	Args:                  cobra.NoArgs,
	ValidArgsFunction:     refreshValidArgs,
	RunE:                  refreshRun,
}

func init() {
	rootCmd.AddCommand(refreshCmd)
}

func refreshValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func refreshRun(cmd *cobra.Command, paths []string) error {
	ps, err := periscope.New(&periscope.Options{
		Debug: rootFlags.debug,
	})
	if err != nil {
		return err
	}
	return ps.Refresh(&periscope.RefreshOptions{})
}
