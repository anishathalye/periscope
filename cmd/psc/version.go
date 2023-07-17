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
	ValidArgsFunction:     versionValidArgs,
	RunE:                  versionRun,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func versionValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}

var (
	version string
	commit  string
)

func versionRun(cmd *cobra.Command, _ []string) error {
	if version != "" && commit != "" {
		// release built with GoReleaser
		fmt.Printf("Periscope v%s (git %s)\n", version, commit[:10])
		return nil
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		var version string
		if bi.Main.Version != "(devel)" {
			version = bi.Main.Version + " "
		}
		var vcs, revision, dirty string
		for _, kv := range bi.Settings {
			switch kv.Key {
			case "vcs":
				vcs = kv.Value
			case "vcs.revision":
				revision = kv.Value
			case "vcs.modified":
				if kv.Value == "true" {
					dirty = "-dirty"
				}
			}
		}
		if vcs != "" && revision != "" {
			fmt.Printf("Periscope %s(%s %s%s)\n", version, vcs, revision[:10], dirty)
		} else {
			fmt.Printf("Periscope %s\n", bi.Main.Version)
		}
		return nil
	}
	fmt.Println("Periscope v? (version information unavailable)")
	return nil
}
