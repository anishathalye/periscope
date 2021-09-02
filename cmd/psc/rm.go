package main

import (
	"github.com/anishathalye/periscope/internal/herror"
	"github.com/anishathalye/periscope/internal/periscope"

	"github.com/spf13/cobra"
)

var rmFlags struct {
	recursive bool
	verbose   bool
	dryRun    bool
	contained optionPath
	arbitrary bool
}

var rmCmd = &cobra.Command{
	Use:                   "rm [flags] path ...",
	Short:                 "Remove duplicates",
	DisableFlagsInUseLine: true,
	Args:                  cobra.MinimumNArgs(1),
	ValidArgsFunction:     rmValidArgs,
	PreRunE:               rmPreRun,
	RunE:                  rmRun,
}

func init() {
	rmCmd.Flags().BoolVarP(&rmFlags.recursive, "recursive", "r", false, "recursively delete duplicates")
	rmCmd.Flags().BoolVarP(&rmFlags.verbose, "verbose", "v", false, "list files being deleted")
	rmCmd.Flags().BoolVarP(&rmFlags.dryRun, "dry-run", "n", false, "do not delete files, but show files eligible for deletion")
	rmCmd.Flags().VarP(&rmFlags.contained, "contained", "c", "delete only files that have a duplicate here")
	rmCmd.Flags().BoolVarP(&rmFlags.arbitrary, "arbitrary", "a", false, "arbitrarily choose a file to leave out when deleting a set with no other duplicates")
	rootCmd.AddCommand(rmCmd)
}

func rmValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveDefault
}

func rmPreRun(cmd *cobra.Command, paths []string) error {
	if rmFlags.arbitrary && rmFlags.contained.valid {
		return herror.User(nil, "-a/--arbitrary and -c/--contained can't be used together")
	}
	return nil
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
		Arbitrary:    rmFlags.arbitrary,
	}
	return ps.Rm(paths, options)
}
