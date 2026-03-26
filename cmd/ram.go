package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var ramCmd = &cobra.Command{
	Use:   "ram <file> <server>",
	Short: "Show RAM cost of a script",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if currentConn == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "not connected to Bitburner")
			return
		}
		gb, err := currentConn.CalculateRam(context.Background(), args[1], args[0])
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "error: %v\n", err)
			return
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s on %s: %.2f GB\n", args[0], args[1], gb)
	},
}

func init() {
	rootCmd.AddCommand(ramCmd)
}
