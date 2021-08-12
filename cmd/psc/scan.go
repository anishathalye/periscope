package main

import (
	"github.com/anishathalye/periscope"

	"github.com/spf13/cobra"
)

var scanFlags struct {
	minimum size
	maximum size
}

var scanCmd = &cobra.Command{
	Use:               "scan [path ...]",
	Short:             "Scan paths for duplicates",
	Args:              cobra.ArbitraryArgs,
	ValidArgsFunction: scanValidArgs,
	RunE:              scanRun,
}

func init() {
	scanCmd.Flags().VarP(&scanFlags.minimum, "minimum", "m", "minimum file size to scan")
	scanCmd.Flags().VarP(&scanFlags.maximum, "maximum", "M", "maximum file size to scan")
	rootCmd.AddCommand(scanCmd)
}

func scanValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveFilterDirs
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
	options := &periscope.ScanOptions{
		Minimum: scanFlags.minimum.value,
		Maximum: scanFlags.maximum.value,
	}
	return ps.Scan(paths, options)
}
