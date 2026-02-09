package commands

import (
	"fmt"
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
			if err := cmd.Root().GenBashCompletion(os.Stdout); err != nil {
				return fmt.Errorf("generating bash completion: %w", err)
			}
		case "zsh":
			if err := cmd.Root().GenZshCompletion(os.Stdout); err != nil {
				return fmt.Errorf("generating zsh completion: %w", err)
			}
		case "fish":
			if err := cmd.Root().GenFishCompletion(os.Stdout, true); err != nil {
				return fmt.Errorf("generating fish completion: %w", err)
			}
		case "powershell":
			if err := cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout); err != nil {
				return fmt.Errorf("generating powershell completion: %w", err)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
