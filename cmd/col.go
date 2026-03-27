package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	col "github.com/KieranGliver/bitburner-larry/internal/col"
	"github.com/spf13/cobra"
)

// colRPC is a convenience wrapper that uses the package-level currentConn.
func colRPC(id string, req map[string]any) (string, error) {
	return col.ColRPCWith(currentConn, id, req)
}

var colCmd = &cobra.Command{
	Use:   "col",
	Short: "Send commands to the Col daemon in Bitburner",
}

var colExecCmd = &cobra.Command{
	Use:   "exec <server> <script> [args...]",
	Short: "Execute a script on a server via Col",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if currentConn == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "not connected to Bitburner")
			return
		}
		threads, _ := cmd.Flags().GetInt("threads")
		id := col.ColNextID()

		scriptArgs := make([]any, len(args)-2)
		for i, a := range args[2:] {
			scriptArgs[i] = a
		}

		fmt.Fprintf(cmd.OutOrStdout(), "sent %s, waiting for response...\n", id)
		result, err := colRPC(id, map[string]any{
			"id": id, "action": "exec",
			"server": args[0], "script": args[1],
			"threads": threads, "args": scriptArgs,
		})
		if err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), err)
			return
		}

		var resp col.ColResponse
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "error parsing response: %v\n", err)
			return
		}
		if resp.Success {
			col.TrackProcess(currentConn, args[0], args[1], resp.PID, threads, scriptArgs)
			fmt.Fprintf(cmd.OutOrStdout(), "ok pid=%d\n", resp.PID)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "failed: %s\n", resp.Error)
		}
	},
}

var colDeployCmd = &cobra.Command{
	Use:   "deploy <server> <script> [args...]",
	Short: "Copy a script to a server and execute it via Col",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if currentConn == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "not connected to Bitburner")
			return
		}
		threads, _ := cmd.Flags().GetInt("threads")
		id := col.ColNextID()

		scriptArgs := make([]any, len(args)-2)
		for i, a := range args[2:] {
			scriptArgs[i] = a
		}

		fmt.Fprintf(cmd.OutOrStdout(), "sent %s, deploying %s to %s...\n", id, args[1], args[0])
		result, err := colRPC(id, map[string]any{
			"id": id, "action": "deploy",
			"server": args[0], "script": args[1],
			"threads": threads, "args": scriptArgs,
		})
		if err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), err)
			return
		}

		var resp col.ColResponse
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "error parsing response: %v\n", err)
			return
		}
		if resp.Success {
			col.TrackProcess(currentConn, args[0], args[1], resp.PID, threads, scriptArgs)
			fmt.Fprintf(cmd.OutOrStdout(), "ok pid=%d\n", resp.PID)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "failed: %s\n", resp.Error)
		}
	},
}

var colKillAllCmd = &cobra.Command{
	Use:   "killall [server]",
	Short: "Kill all scripts on a server (or every server) via Col; col.js on home is always preserved",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if currentConn == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "not connected to Bitburner")
			return
		}
		id := col.ColNextID()

		servers := []string{}
		if len(args) == 1 {
			servers = []string{args[0]}
		}

		if len(servers) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "sent %s, killing all scripts on every server...\n", id)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "sent %s, killing all scripts on %s...\n", id, servers[0])
		}
		result, err := colRPC(id, map[string]any{
			"id": id, "action": "killall", "servers": servers,
		})
		if err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), err)
			return
		}

		var resp col.ColKillAllResponse
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "error parsing response: %v\n", err)
			return
		}
		if !resp.Success {
			fmt.Fprintf(cmd.OutOrStdout(), "failed: %s\n", resp.Error)
		} else if len(resp.Killed) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "ok: stopped scripts on %v\n", resp.Killed)
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "ok: nothing was running")
		}
	},
}

var colCrackCmd = &cobra.Command{
	Use:   "crack [server]",
	Short: "Crack servers via Col (omit server to crack all)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if currentConn == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "not connected to Bitburner")
			return
		}
		var targets []string
		if len(args) == 1 {
			targets = []string{args[0]}
		}
		cracked, failed, err := col.DoCrack(currentConn, targets)
		if err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), err)
			return
		}
		if len(cracked) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "cracked: %v\n", cracked)
		}
		if len(failed) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "failed (need more programs): %v\n", failed)
		}
	},
}

var colScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Collect full world state from Bitburner via Col",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if currentConn == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "not connected to Bitburner")
			return
		}
		server, _ := cmd.Flags().GetString("server")
		fmt.Fprintf(cmd.OutOrStdout(), "scanning world (via %s)...\n", server)
		w, err := col.DoScan(currentConn, server)
		if err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), err)
			return
		}
		fmt.Fprintf(cmd.OutOrStdout(), "ok: %d servers scanned\n", len(w.Servers))
	},
}

var colPingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Test the col→HTTP round-trip by exec'ing task-ping.js and waiting for /done",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if currentConn == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "not connected to Bitburner")
			return
		}
		id := col.ColNextID()
		ch := currentConn.RegisterHTTP(id)

		server, _ := cmd.Flags().GetString("server")

		fmt.Fprintf(cmd.OutOrStdout(), "sent %s, waiting for pong (via %s)...\n", id, server)
		ackResult, err := colRPC(id, map[string]any{
			"id": id, "action": "deploy",
			"server": server, "script": "task-ping.js",
			"threads": 1, "args": []any{id},
		})
		if err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), err)
			return
		}
		var ack col.ColResponse
		if err := json.Unmarshal([]byte(ackResult), &ack); err != nil || !ack.Success {
			msg := ack.Error
			if err != nil {
				msg = err.Error()
			}
			fmt.Fprintf(cmd.OutOrStdout(), "exec failed: %s\n", msg)
			return
		}
		col.TrackProcess(currentConn, server, "task-ping.js", ack.PID, 1, []any{id})

		select {
		case <-ch:
			fmt.Fprintln(cmd.OutOrStdout(), "pong: col→HTTP round-trip ok")
		case <-time.After(10 * time.Second):
			fmt.Fprintln(cmd.OutOrStdout(), "timeout: no response from task-ping.js")
		}
	},
}

var colRunCmd = &cobra.Command{
	Use:   "run <script> [args...]",
	Short: "Spread a script across all available servers to hit a target thread count",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if currentConn == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "not connected to Bitburner")
			return
		}
		threads, _ := cmd.Flags().GetInt("threads")
		asJSON, _ := cmd.Flags().GetBool("json")
		script := args[0]
		scriptArgs := make([]any, len(args)-1)
		for i, a := range args[1:] {
			scriptArgs[i] = a
		}

		result, err := col.DoRun(currentConn, script, threads, scriptArgs)
		if err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), err)
			return
		}

		if asJSON {
			out, _ := json.MarshalIndent(result, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(out))
		} else {
			if len(result.Dispatches) == 0 && len(result.Errors) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no servers with enough free RAM")
				return
			}
			for _, d := range result.Dispatches {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s: %d threads (pid %d)\n", d.Server, d.Threads, d.PID)
			}
			for _, e := range result.Errors {
				fmt.Fprintf(cmd.OutOrStdout(), "  error: %s\n", e)
			}
			if result.ThreadsRemaining > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "warning: %d/%d threads unscheduled — not enough free RAM\n", result.ThreadsRemaining, threads)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "done: %d scheduled, %d remaining\n", result.ThreadsScheduled, result.ThreadsRemaining)
		}
	},
}

var colCalcCmd = &cobra.Command{
	Use:   "calc <target>",
	Short: "Calculate hack/grow/weaken thread counts and timings for a target via Col",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if currentConn == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "not connected to Bitburner")
			return
		}
		target := args[0]
		hackPercent, _ := cmd.Flags().GetFloat64("hack-percent")
		asJSON, _ := cmd.Flags().GetBool("json")

		resp, err := col.DoCalc(currentConn, target, hackPercent)
		if err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), err)
			return
		}
		if asJSON {
			out, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(out))
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "target: %s  (hack %.0f%%)\n", resp.Target, resp.HackPercent*100)
			if resp.PrepWeakenThreads > 0 || resp.PrepGrowThreads > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "  prep-weaken:       %4d threads  %6.1fs  (not ready)\n", resp.PrepWeakenThreads, resp.WeakenTime/1000)
				fmt.Fprintf(cmd.OutOrStdout(), "  prep-grow:         %4d threads  %6.1fs  (not ready)\n", resp.PrepGrowThreads, resp.GrowTime/1000)
				fmt.Fprintf(cmd.OutOrStdout(), "  prep-grow-weaken:  %4d threads  %6.1fs  (not ready)\n", resp.PrepGrowWeakenThreads, resp.WeakenTime/1000)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "  prep: ready")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  hack:              %4d threads  %6.1fs\n", resp.HackThreads, resp.HackTime/1000)
			fmt.Fprintf(cmd.OutOrStdout(), "  weaken-hack:       %4d threads  %6.1fs\n", resp.WeakenHackThreads, resp.WeakenTime/1000)
			fmt.Fprintf(cmd.OutOrStdout(), "  grow:              %4d threads  %6.1fs\n", resp.GrowThreads, resp.GrowTime/1000)
			fmt.Fprintf(cmd.OutOrStdout(), "  weaken-grow:       %4d threads  %6.1fs\n", resp.WeakenGrowThreads, resp.WeakenTime/1000)
		}
	},
}

func init() {
	colExecCmd.Flags().IntP("threads", "t", 1, "number of threads")
	colDeployCmd.Flags().IntP("threads", "t", 1, "number of threads")
	colRunCmd.Flags().IntP("threads", "t", 1, "total threads to spread across servers")
	colRunCmd.Flags().Bool("json", false, "output machine-readable JSON")
	colCalcCmd.Flags().Float64("hack-percent", 0.75, "fraction of max money to steal per hack")
	colCalcCmd.Flags().Bool("json", false, "output machine-readable JSON")
	colPingCmd.Flags().StringP("server", "s", "home", "server to run task-ping.js on")
	colScanCmd.Flags().StringP("server", "s", "home", "server to run task-scan.js on")
	colCmd.AddCommand(colExecCmd)
	colCmd.AddCommand(colDeployCmd)
	colCmd.AddCommand(colRunCmd)
	colCmd.AddCommand(colCalcCmd)
	colCmd.AddCommand(colKillAllCmd)
	colCmd.AddCommand(colCrackCmd)
	colCmd.AddCommand(colPingCmd)
	colCmd.AddCommand(colScanCmd)
	rootCmd.AddCommand(colCmd)
}
