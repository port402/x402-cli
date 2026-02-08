package commands

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for x402.

To load completions:

Bash:
  $ source <(x402 completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ x402 completion bash > /etc/bash_completion.d/x402
  # macOS:
  $ x402 completion bash > $(brew --prefix)/etc/bash_completion.d/x402

Zsh:
  $ source <(x402 completion zsh)
  # To load completions for each session, execute once:
  $ x402 completion zsh > "${fpath[1]}/_x402"

Fish:
  $ x402 completion fish | source
  # To load completions for each session, execute once:
  $ x402 completion fish > ~/.config/fish/completions/x402.fish

PowerShell:
  PS> x402 completion powershell | Out-String | Invoke-Expression`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
