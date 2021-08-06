package cli

import (
	"os"

	"github.com/spf13/cobra"
)

func CompletionCmd() *cobra.Command {

	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion script",
		Long: `To load completions:

Bash:

  $ source <(kots completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ kots completion bash > /etc/bash_completion.d/kots
  # macOS:
  $ kots completion bash > /usr/local/etc/bash_completion.d/kots

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ kots completion zsh > "${fpath[1]}/_kots"

  # You will need to start a new shell for this setup to take effect.

fish:

  $ kots completion fish | source

  # To load completions for each session, execute once:
  $ kots completion fish > ~/.config/fish/completions/kots.fish

PowerShell:

  PS> kots completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> kots completion powershell > kots.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
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
		},
	}
}
