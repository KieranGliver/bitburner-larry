package cmd

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/KieranGliver/bitburner-larry/internal/brain"
	"github.com/KieranGliver/bitburner-larry/internal/communication"
	"github.com/spf13/cobra"
)

var currentConn *communication.BitburnerConn

// CurrentWorld holds the most recent world state from "col scan".
var CurrentWorld *brain.World

// ExecuteCommand runs a cobra command from the TUI terminal, captures its output,
// and returns it as a string. conn is stored for subcommands to access via currentConn.
func ExecuteCommand(input string, conn *communication.BitburnerConn) string {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs(strings.Fields(input))
	currentConn = conn
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
