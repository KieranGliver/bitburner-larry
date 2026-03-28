package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/KieranGliver/bitburner-larry/internal/brain"
	col "github.com/KieranGliver/bitburner-larry/internal/col"
	"github.com/KieranGliver/bitburner-larry/internal/logger"
	"github.com/KieranGliver/bitburner-larry/internal/world"
	"github.com/spf13/cobra"
)

var (
	currentBrain *brain.Brain
	brainCancel  context.CancelFunc
)

// brainTargets defines the priority order for the brain to target servers.
// Servers earlier in the list are targeted first. Servers not in the list rank last.
var brainTargets = []string{
	"n00dles",
	"foodnstuff",
	"sigma-cosmetics",
}

func rankByTargetList(a, b world.BitServer) bool {
	idxA, idxB := len(brainTargets), len(brainTargets)
	for i, name := range brainTargets {
		if a.Hostname == name {
			idxA = i
		}
		if b.Hostname == name {
			idxB = i
		}
	}
	return idxA < idxB
}

func ensureBrain() {
	if currentBrain == nil {
		onLog := func(msg string, level logger.Level) {
			fmt.Fprintf(os.Stderr, "[brain] %s: %s\n", level, msg)
		}
		currentBrain = brain.New(rankByTargetList, onLog)
	}
}

var brainCmd = &cobra.Command{
	Use:   "brain",
	Short: "Control the hacking brain loop",
}

var brainStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the brain loop (ticks every 10s)",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if currentConn == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "not connected to Bitburner")
			return
		}
		if brainCancel != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "brain already running")
			return
		}
		ensureBrain()
		ctx, cancel := context.WithCancel(context.Background())
		brainCancel = cancel
		conn := currentConn
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					w := col.CurrentWorld
					if w == nil {
						fmt.Fprintln(os.Stderr, "[brain] WARN: no world state, skipping tick")
						continue
					}
					currentBrain.Tick(w, conn)
				}
			}
		}()
		fmt.Fprintln(cmd.OutOrStdout(), "brain started")
	},
}

var brainTickCmd = &cobra.Command{
	Use:   "tick",
	Short: "Run a single brain tick immediately",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if currentConn == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "not connected to Bitburner")
			return
		}
		ensureBrain()
		w := col.CurrentWorld
		if w == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "no world state yet")
			return
		}
		currentBrain.Tick(w, currentConn)
		fmt.Fprintln(cmd.OutOrStdout(), "tick done")
	},
}

var brainEndCmd = &cobra.Command{
	Use:   "end",
	Short: "Stop the brain loop (brain state is preserved)",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if brainCancel == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "brain not running")
			return
		}
		brainCancel()
		brainCancel = nil
		fmt.Fprintln(cmd.OutOrStdout(), "brain stopped")
	},
}

func init() {
	brainCmd.AddCommand(brainStartCmd, brainTickCmd, brainEndCmd)
	rootCmd.AddCommand(brainCmd)
}
