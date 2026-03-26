package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var filesCmd = &cobra.Command{
	Use:   "files <server>",
	Short: "List files on a server",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if currentConn == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "not connected to Bitburner")
			return
		}
		names, err := currentConn.GetFileNames(context.Background(), args[0])
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "error: %v\n", err)
			return
		}
		for _, name := range names {
			fmt.Fprintln(cmd.OutOrStdout(), name)
		}
	},
}

func init() {
	rootCmd.AddCommand(filesCmd)
}
