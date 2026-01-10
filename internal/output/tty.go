// Package output handles terminal detection and output formatting.
package output

import (
	"os"

	"golang.org/x/term"
)

// IsTTY returns true if stdout is connected to a terminal.
// When false, output is being piped or redirected.
func IsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// IsStdinTTY returns true if stdin is connected to a terminal.
// When false, input is being piped.
func IsStdinTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// IsStderrTTY returns true if stderr is connected to a terminal.
func IsStderrTTY() bool {
	return term.IsTerminal(int(os.Stderr.Fd()))
}
