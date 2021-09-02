package main

import (
	"github.com/anishathalye/periscope/internal/periscope"

	"github.com/spf13/cobra"
)

var forgetCmd = &cobra.Command{
	Use:                   "forget path ...",
	Short:                 "Forget duplicates in a directory",
	DisableFlagsInUseLine: true,
	Args:                  cobra.MinimumNArgs(1),
	ValidArgsFunction:     forgetValidArgs,
	RunE:                  forgetRun,
}

func init() {
	rootCmd.AddCommand(forgetCmd)
}

func forgetValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveDefault
}

func forgetRun(cmd *cobra.Command, paths []string) error {
	ps, err := periscope.New(&periscope.Options{
		Debug: rootFlags.debug,
	})
	if err != nil {
		return err
	}
	options := &periscope.ForgetOptions{}
	return ps.Forget(paths, options)
}
