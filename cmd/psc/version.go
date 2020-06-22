package main

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:                   "version",
	Short:                 "Show version number",
	DisableFlagsInUseLine: true,
	Args:                  cobra.NoArgs,
	RunE:                  versionRun,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

var (
	version string
	commit  string
	date    string
)

func versionRun(cmd *cobra.Command, _ []string) error {
	if version != "" && commit != "" {
		// release built with GoReleaser
		fmt.Printf("Periscope v%s (git sha1 %s)\n", version, commit[:10])
		return nil
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		fmt.Printf("Periscope %s\n", bi.Main.Version)
		return nil
	}
	fmt.Println("Periscope v? (version information unavailable)")
	return nil
}
