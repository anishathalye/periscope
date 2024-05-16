package main

import (
	"io"
	"log"

	"github.com/spf13/cobra"
)

var rootFlags struct {
	debug bool
}

var rootCmd = &cobra.Command{
	Use:              "psc",
	PersistentPreRun: preRoot,
	SilenceUsage:     true,
	SilenceErrors:    true,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&rootFlags.debug, "debug", false, "enable debug mode")
	rootCmd.PersistentFlags().MarkHidden("debug")
}

func preRoot(cmd *cobra.Command, args []string) {
	if !rootFlags.debug {
		log.SetFlags(0)
		log.SetOutput(io.Discard)
	} else {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}
}
