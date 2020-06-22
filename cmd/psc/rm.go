package main

import (
	"github.com/anishathalye/periscope"

	"github.com/spf13/cobra"
)

var rmFlags struct {
	recursive bool
	verbose   bool
	dryRun    bool
	contained optionPath
}

var rmCmd = &cobra.Command{
	Use:   "rm path ...",
	Short: "Remove duplicates",
	Args:  cobra.MinimumNArgs(1),
	RunE:  rmRun,
}

func init() {
	rmCmd.Flags().BoolVarP(&rmFlags.recursive, "recursive", "r", false, "recursively delete duplicates")
	rmCmd.Flags().BoolVarP(&rmFlags.verbose, "verbose", "v", false, "list files being deleted")
	rmCmd.Flags().BoolVarP(&rmFlags.dryRun, "dry-run", "n", false, "do not delete files, but show files eligible for deletion")
	rmCmd.Flags().VarP(&rmFlags.contained, "contained", "c", "delete only files that have a duplicate here")
	rootCmd.AddCommand(rmCmd)
}

func rmRun(cmd *cobra.Command, paths []string) error {
	ps, err := periscope.New(&periscope.Options{
		Debug: rootFlags.debug,
	})
	if err != nil {
		return err
	}
	options := &periscope.RmOptions{
		Recursive:    rmFlags.recursive,
		Verbose:      rmFlags.verbose || rmFlags.dryRun,
		DryRun:       rmFlags.dryRun,
		HasContained: rmFlags.contained.valid,
		Contained:    rmFlags.contained.value,
	}
	return ps.Rm(paths, options)
}
