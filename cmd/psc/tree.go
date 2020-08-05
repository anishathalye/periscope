package main

import (
	"github.com/anishathalye/periscope"

	"github.com/spf13/cobra"
)

var treeFlags struct {
	all bool
}

var treeCmd = &cobra.Command{
	Use:               "tree [path]",
	Short:             "List all duplicates recursively",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: treeValidArgs,
	RunE:              treeRun,
}

func init() {
	treeCmd.Flags().BoolVarP(&treeFlags.all, "all", "a", false, "show hidden files/directories")
	rootCmd.AddCommand(treeCmd)
}

func treeValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveFilterDirs
}

func treeRun(cmd *cobra.Command, paths []string) error {
	ps, err := periscope.New(&periscope.Options{
		Debug: rootFlags.debug,
	})
	if err != nil {
		return err
	}
	var path string
	if len(paths) == 1 {
		path = paths[0]
	} else {
		path = "."
	}
	return ps.Tree(path, &periscope.TreeOptions{
		All: treeFlags.all,
	})
}
