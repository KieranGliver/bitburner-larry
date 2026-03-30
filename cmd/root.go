package cmd

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/KieranGliver/bitburner-larry/internal/app"
	"github.com/spf13/cobra"
)

var currentState *app.AppState

// ExecuteCommand runs a cobra command from the TUI terminal, captures its output,
// and returns it as a string. s is stored for subcommands to access via appState.
func ExecuteCommand(input string, s *app.AppState) string {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs(strings.Fields(input))
	currentState = s
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(&buf, "error: %v", err)
	}
	return strings.TrimSpace(buf.String())
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:           "larry",
	Short:         "Bitburner sync terminal",
	SilenceErrors: true,
	SilenceUsage:  true,
}
