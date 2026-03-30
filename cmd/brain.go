package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var brainCmd = &cobra.Command{
	Use:   "brain",
	Short: "Control the hacking brain loop",
}

var brainStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the brain loop (ticks every 10s)",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if currentState.BrainRunning() {
			fmt.Fprintln(cmd.OutOrStdout(), "brain already running")
			return
		}
		currentState.BrainStart()
		fmt.Fprintln(cmd.OutOrStdout(), "brain started")
	},
}

var brainTickCmd = &cobra.Command{
	Use:   "tick",
	Short: "Run a single brain tick immediately",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		currentState.BrainTick()
		fmt.Fprintln(cmd.OutOrStdout(), "tick done")
	},
}

var brainEndCmd = &cobra.Command{
	Use:   "end",
	Short: "Stop the brain loop (brain state is preserved)",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if !currentState.BrainRunning() {
			fmt.Fprintln(cmd.OutOrStdout(), "brain not running")
			return
		}
		currentState.BrainStop()
		fmt.Fprintln(cmd.OutOrStdout(), "brain stopped")
	},
}

func init() {
	brainCmd.AddCommand(brainStartCmd, brainTickCmd, brainEndCmd)
	rootCmd.AddCommand(brainCmd)
}
