package main

import (
	"github.com/anishathalye/periscope"

	"github.com/spf13/cobra"
)

var finishCmd = &cobra.Command{
	Use:                   "finish",
	Short:                 "Delete duplicate database",
	DisableFlagsInUseLine: true,
	Args:                  cobra.NoArgs,
	RunE:                  finishRun,
}

func init() {
	rootCmd.AddCommand(finishCmd)
}

func finishRun(cmd *cobra.Command, _ []string) error {
	return periscope.Finish(&periscope.FinishOptions{})
}
