package main

import (
	"github.com/anishathalye/periscope/internal/herror"

	"fmt"
	"os"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		if herr, ok := err.(herror.Interface); ok {
			fmt.Fprint(os.Stderr, herr.Herror(rootFlags.debug))
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
