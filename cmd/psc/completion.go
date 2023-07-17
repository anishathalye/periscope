package main

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:

# The completion script requires the bash-completion package. Once you have it
# installed and configured, you can use completions:
$ source <(psc completion bash)

# To load completions for each session, execute once:
Linux:
  $ psc completion bash > /etc/bash_completion.d/psc
MacOS:
  $ psc completion bash > /usr/local/etc/bash_completion.d/psc

Zsh:

# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:
$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ psc completion zsh > "${fpath[1]}/_psc"

# You will need to start a new shell for this setup to take effect.

Fish:

$ psc completion fish | source

# To load completions for each session, execute once:
$ psc completion fish > ~/.config/fish/completions/psc.fish
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE:                  completionRun,
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

func completionRun(cmd *cobra.Command, args []string) error {
	switch args[0] {
	case "bash":
		cmd.Root().GenBashCompletion(os.Stdout)
	case "zsh":
		cmd.Root().GenZshCompletion(os.Stdout)
	case "fish":
		cmd.Root().GenFishCompletion(os.Stdout, true)
	case "powershell":
		cmd.Root().GenPowerShellCompletion(os.Stdout)
	}
	return nil
}
