package cmd

import (
	"fmt"

	"github.com/KieranGliver/bitburner-larry/internal/communication"
	"github.com/spf13/cobra"
)

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Show connection status",
	Run: func(cmd *cobra.Command, args []string) {
		if currentConn == nil || currentConn.Status == communication.Disconnected {
			fmt.Fprintln(cmd.OutOrStdout(), "disconnected")
			return
		}
		fmt.Fprintln(cmd.OutOrStdout(), "connected")
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)
}
