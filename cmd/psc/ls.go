package main

import (
	"github.com/anishathalye/periscope/internal/herror"
	"github.com/anishathalye/periscope/internal/periscope"

	"github.com/spf13/cobra"
)

var lsFlags struct {
	all       bool
	verbose   bool
	duplicate bool
	unique    bool
	relative  bool
	recursive bool
	files     bool
}

var lsCmd = &cobra.Command{
	Use:                   "ls [flags] [path ...]",
	Short:                 "List a directory",
	DisableFlagsInUseLine: true,
	Args:                  cobra.ArbitraryArgs,
	ValidArgsFunction:     lsValidArgs,
	PreRunE:               lsPreRun,
	RunE:                  lsRun,
}

func init() {
	lsCmd.Flags().BoolVarP(&lsFlags.all, "all", "a", false, "show hidden files")
	lsCmd.Flags().BoolVarP(&lsFlags.verbose, "verbose", "v", false, "list duplicates")
	lsCmd.Flags().BoolVarP(&lsFlags.duplicate, "duplicate", "d", false, "show only duplicates")
	lsCmd.Flags().BoolVarP(&lsFlags.unique, "unique", "u", false, "show only unique files")
	lsCmd.Flags().BoolVarP(&lsFlags.relative, "relative", "r", false, "show duplicates using relative paths")
	lsCmd.Flags().BoolVarP(&lsFlags.recursive, "recursive", "R", false, "list subdirectories recursively")
	lsCmd.Flags().BoolVarP(&lsFlags.files, "files", "f", false, "show only files")
	rootCmd.AddCommand(lsCmd)
}

func lsValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveFilterDirs
}

func lsPreRun(cmd *cobra.Command, paths []string) error {
	if lsFlags.duplicate && lsFlags.unique {
		return herror.User(nil, "-d/--duplicate and -u/--unique can't be used together")
	}
	return nil
}

func lsRun(cmd *cobra.Command, paths []string) error {
	ps, err := periscope.New(&periscope.Options{
		Debug: rootFlags.debug,
	})
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}
	options := &periscope.LsOptions{
		All:       lsFlags.all,
		Verbose:   lsFlags.verbose,
		Duplicate: lsFlags.duplicate,
		Unique:    lsFlags.unique,
		Relative:  lsFlags.relative,
		Recursive: lsFlags.recursive,
		Files:     lsFlags.files,
	}
	return ps.Ls(paths, options)
}
