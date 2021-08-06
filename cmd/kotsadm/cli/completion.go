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

  $ source <(kotsadm completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ kotsadm completion bash > /etc/bash_completion.d/kotsadm
  # macOS:
  $ kotsadm completion bash > /usr/local/etc/bash_completion.d/kotsadm

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ kotsadm completion zsh > "${fpath[1]}/_kotsadm"

  # You will need to start a new shell for this setup to take effect.

fish:

  $ kotsadm completion fish | source

  # To load completions for each session, execute once:
  $ kotsadm completion fish > ~/.config/fish/completions/kotsadm.fish

PowerShell:

  PS> kotsadm completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> kotsadm completion powershell > kotsadm.ps1
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
