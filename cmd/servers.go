package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var serversCmd = &cobra.Command{
	Use:   "servers",
	Short: "List all servers",
	Run: func(cmd *cobra.Command, args []string) {
		if currentConn == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "not connected to Bitburner")
			return
		}
		servers, err := currentConn.GetAllServers(context.Background())
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "error: %v\n", err)
			return
		}
		for _, s := range servers {
			line := s.Hostname
			if s.HasAdminRights {
				line += " (admin)"
			}
			if s.PurchasedByPlayer {
				line += " (purchased)"
			}
			fmt.Fprintln(cmd.OutOrStdout(), line)
		}
	},
}

func init() {
	rootCmd.AddCommand(serversCmd)
}
