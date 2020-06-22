package main

import (
	"github.com/anishathalye/periscope"

	"github.com/spf13/cobra"
)

var refreshCmd = &cobra.Command{
	Use:                   "refresh",
	Short:                 "Remove deleted files from database",
	DisableFlagsInUseLine: true,
	Args:                  cobra.NoArgs,
	RunE:                  refreshRun,
}

func init() {
	rootCmd.AddCommand(refreshCmd)
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
