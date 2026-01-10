package commands

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/port402/x402-cli/internal/output"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display version, build information, and runtime details.`,
	Run:   runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) {
	if GetJSONOutput() {
		output.PrintJSON(map[string]string{
			"version":   Version,
			"commit":    Commit,
			"buildDate": BuildDate,
			"go":        runtime.Version(),
			"os":        runtime.GOOS,
			"arch":      runtime.GOARCH,
		})
		return
	}

	fmt.Printf("x402 version %s\n", Version)
	if Commit != "none" {
		fmt.Printf("  Commit:     %s\n", Commit)
	}
	if BuildDate != "unknown" {
		fmt.Printf("  Built:      %s\n", BuildDate)
	}
	fmt.Printf("  Go version: %s\n", runtime.Version())
	fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
