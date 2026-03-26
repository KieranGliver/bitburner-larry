/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// getAllServersCmd represents the getAllServers command
var getAllServersCmd = &cobra.Command{
	Use:   "getAllServers",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
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
			fmt.Fprintln(cmd.OutOrStdout(), s.Hostname)
		}
	},
}

func init() {
	rootCmd.AddCommand(getAllServersCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getAllServersCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getAllServersCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
