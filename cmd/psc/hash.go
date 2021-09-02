package main

import (
	"github.com/anishathalye/periscope/internal/periscope"

	"github.com/spf13/cobra"
)

var hashCmd = &cobra.Command{
	Use:                   "hash path ...",
	Short:                 "Hash a file",
	DisableFlagsInUseLine: true,
	Args:                  cobra.MinimumNArgs(1),
	ValidArgsFunction:     hashValidArgs,
	RunE:                  hashRun,
}

func init() {
	rootCmd.AddCommand(hashCmd)
}

func hashValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveDefault
}

func hashRun(cmd *cobra.Command, paths []string) error {
	ps, err := periscope.New(&periscope.Options{
		Debug: rootFlags.debug,
	})
	if err != nil {
		return err
	}
	options := &periscope.HashOptions{}
	return ps.Hash(paths, options)
}
