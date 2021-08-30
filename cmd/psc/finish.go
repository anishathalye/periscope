package main

import (
	"github.com/anishathalye/periscope/internal/periscope"

	"github.com/spf13/cobra"
)

var finishCmd = &cobra.Command{
	Use:                   "finish",
	Short:                 "Delete duplicate database",
	DisableFlagsInUseLine: true,
	Args:                  cobra.NoArgs,
	ValidArgsFunction:     finishValidArgs,
	RunE:                  finishRun,
}

func init() {
	rootCmd.AddCommand(finishCmd)
}

func finishValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func finishRun(cmd *cobra.Command, _ []string) error {
	return periscope.Finish(&periscope.FinishOptions{})
}
