package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/KieranGliver/bitburner-larry/internal/brain"
	"github.com/spf13/cobra"
)

var colRequestCounter int64

// response types — only needed because we access specific fields after unmarshal

type colResponse struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
	PID     int    `json:"pid"`
	Error   string `json:"error"`
}

type colCrackResponse struct {
	ID      string   `json:"id"`
	Success bool     `json:"success"`
	Cracked []string `json:"cracked"`
	Failed  []string `json:"failed"`
	Error   string   `json:"error"`
}

type colScanResponse struct {
	ID      string            `json:"id"`
	Success bool              `json:"success"`
	Error   string            `json:"error"`
	Player  brain.Player      `json:"player"`
	Servers []brain.BitServer `json:"servers"`
}

type colKillAllResponse struct {
	ID      string   `json:"id"`
	Success bool     `json:"success"`
	Killed  []string `json:"killed"`
	Error   string   `json:"error"`
}

// colRPC sends a request to the Col inbox and waits for a response in the outbox.
func colRPC(id string, req map[string]any) (string, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	inboxPath := fmt.Sprintf("/inbox/%s.txt", id)
	outboxPath := fmt.Sprintf("/outbox/%s.txt", id)

	if err := currentConn.PushFile(ctx, "home", inboxPath, string(payload)); err != nil {
		return "", fmt.Errorf("error sending command: %w", err)
	}

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
		result, err := currentConn.GetFile(ctx, "home", outboxPath)
		if err != nil || result == "" {
			continue
		}
		_ = currentConn.DeleteFile(ctx, "home", outboxPath)
		_ = currentConn.DeleteFile(ctx, "home", inboxPath)
		return result, nil
	}
	return "", fmt.Errorf("timeout: no response from Col within 30s")
}

func colNextID() string {
	n := atomic.AddInt64(&colRequestCounter, 1)
	return fmt.Sprintf("COL-%03d", n)
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
		id := colNextID()

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

		var resp colResponse
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "error parsing response: %v\n", err)
			return
		}
		if resp.Success {
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
		id := colNextID()

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

		var resp colResponse
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "error parsing response: %v\n", err)
			return
		}
		if resp.Success {
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
		id := colNextID()

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

		var resp colKillAllResponse
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
		id := colNextID()

		targets := []string{}
		if len(args) == 1 {
			targets = []string{args[0]}
		}

		if len(targets) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "sent %s, cracking all servers...\n", id)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "sent %s, cracking %s...\n", id, targets[0])
		}
		result, err := colRPC(id, map[string]any{
			"id": id, "action": "crack", "targets": targets,
		})
		if err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), err)
			return
		}

		var resp colCrackResponse
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "error parsing response: %v\n", err)
			return
		}
		if len(resp.Cracked) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "cracked: %v\n", resp.Cracked)
		}
		if len(resp.Failed) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "failed (need more programs): %v\n", resp.Failed)
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
		id := colNextID()

		// Register HTTP channel BEFORE pushing to inbox to avoid a race.
		ch := currentConn.RegisterHTTP(id)

		server, _ := cmd.Flags().GetString("server")

		// Phase 1: col deploy's task-scan.js (scp + exec) and acks immediately.
		fmt.Fprintf(cmd.OutOrStdout(), "sent %s, scanning world (via %s)...\n", id, server)
		ackResult, err := colRPC(id, map[string]any{
			"id": id, "action": "deploy",
			"server": server, "script": "task-scan.js",
			"threads": 1, "args": []any{id},
		})
		if err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), err)
			return
		}
		var ack colResponse
		if err := json.Unmarshal([]byte(ackResult), &ack); err != nil || !ack.Success {
			msg := ack.Error
			if err != nil {
				msg = err.Error()
			}
			fmt.Fprintf(cmd.OutOrStdout(), "failed: %s\n", msg)
			return
		}

		// Phase 2: wait for task-scan.js to POST full world data to /done.
		select {
		case data := <-ch:
			var resp colScanResponse
			if err := json.Unmarshal([]byte(data), &resp); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "error parsing scan data: %v\n", err)
				return
			}
			if !resp.Success {
				fmt.Fprintf(cmd.OutOrStdout(), "failed: %s\n", resp.Error)
				return
			}
			CurrentWorld = &brain.World{Player: resp.Player, Servers: resp.Servers}
			fmt.Fprintf(cmd.OutOrStdout(), "ok: %d servers scanned\n", len(resp.Servers))
		case <-time.After(30 * time.Second):
			fmt.Fprintln(cmd.OutOrStdout(), "timeout waiting for scan data from task-scan.js")
		}
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
		id := colNextID()
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
		var ack colResponse
		if err := json.Unmarshal([]byte(ackResult), &ack); err != nil || !ack.Success {
			msg := ack.Error
			if err != nil {
				msg = err.Error()
			}
			fmt.Fprintf(cmd.OutOrStdout(), "exec failed: %s\n", msg)
			return
		}

		select {
		case <-ch:
			fmt.Fprintln(cmd.OutOrStdout(), "pong: col→HTTP round-trip ok")
		case <-time.After(10 * time.Second):
			fmt.Fprintln(cmd.OutOrStdout(), "timeout: no response from task-ping.js")
		}
	},
}

func init() {
	colExecCmd.Flags().IntP("threads", "t", 1, "number of threads")
	colDeployCmd.Flags().IntP("threads", "t", 1, "number of threads")
	colPingCmd.Flags().StringP("server", "s", "home", "server to run task-ping.js on")
	colScanCmd.Flags().StringP("server", "s", "home", "server to run task-scan.js on")
	colCmd.AddCommand(colExecCmd)
	colCmd.AddCommand(colDeployCmd)
	colCmd.AddCommand(colKillAllCmd)
	colCmd.AddCommand(colCrackCmd)
	colCmd.AddCommand(colPingCmd)
	colCmd.AddCommand(colScanCmd)
	rootCmd.AddCommand(colCmd)
}
