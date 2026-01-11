package commands

import (
	"fmt"
	"runtime"
	"strings"

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

	// Compact format: x402 0.1.0 (e0b2c4f)
	commitShort := truncate(Commit, 7)
	if commitShort != "none" {
		fmt.Printf("x402 %s (%s)\n", Version, commitShort)
	} else {
		fmt.Printf("x402 %s\n", Version)
	}

	if BuildDate != "unknown" {
		fmt.Printf("  Built:    %s\n", truncate(BuildDate, 10))
	}

	goVersion := strings.TrimPrefix(runtime.Version(), "go")
	fmt.Printf("  Go:       %s\n", goVersion)
	fmt.Printf("  Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

// truncate returns at most maxLen characters from s.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
