package main

import (
	"github.com/anishathalye/periscope"

	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:                   "scan [path ...]",
	Short:                 "Scan paths for duplicates",
	DisableFlagsInUseLine: true,
	Args:                  cobra.ArbitraryArgs,
	RunE:                  scanRun,
}

func init() {
	rootCmd.AddCommand(scanCmd)
}

func scanRun(cmd *cobra.Command, paths []string) error {
	ps, err := periscope.New(&periscope.Options{
		Debug: rootFlags.debug,
	})
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}
	return ps.Scan(paths, &periscope.ScanOptions{})
}
