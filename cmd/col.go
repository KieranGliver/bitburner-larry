package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"
)

var colRequestCounter int64

type colExecRequest struct {
	ID      string                   `json:"id"`
	Action  string                   `json:"action"`
	Server  string                   `json:"server"`
	Script  string                   `json:"script"`
	Threads int                      `json:"threads"`
	Args    []interface{}            `json:"args"`
}

type colResponse struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
	PID     int    `json:"pid"`
	Error   string `json:"error"`
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
		n := atomic.AddInt64(&colRequestCounter, 1)
		id := fmt.Sprintf("COL-%03d", n)

		scriptArgs := make([]interface{}, len(args)-2)
		for i, a := range args[2:] {
			scriptArgs[i] = a
		}

		req := colExecRequest{
			ID:      id,
			Action:  "exec",
			Server:  args[0],
			Script:  args[1],
			Threads: threads,
			Args:    scriptArgs,
		}

		payload, err := json.Marshal(req)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "error: %v\n", err)
			return
		}

		ctx := context.Background()
		inboxPath := fmt.Sprintf("/inbox/%s.txt", id)
		outboxPath := fmt.Sprintf("/outbox/%s.txt", id)

		if err := currentConn.PushFile(ctx, "home", inboxPath, string(payload)); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "error sending command: %v\n", err)
			return
		}

		fmt.Fprintf(cmd.OutOrStdout(), "sent %s, waiting for response...\n", id)

		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			time.Sleep(500 * time.Millisecond)

			result, err := currentConn.GetFile(ctx, "home", outboxPath)
			if err != nil || result == "" {
				continue
			}

			var resp colResponse
			if err := json.Unmarshal([]byte(result), &resp); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "error parsing response: %v\n", err)
				return
			}

			_ = currentConn.DeleteFile(ctx, "home", outboxPath)

			if resp.Success {
				fmt.Fprintf(cmd.OutOrStdout(), "ok pid=%d\n", resp.PID)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "failed: %s\n", resp.Error)
			}
			return
		}

		fmt.Fprintln(cmd.OutOrStdout(), "timeout: no response from Col within 30s")
	},
}

func init() {
	colExecCmd.Flags().IntP("threads", "t", 1, "number of threads")
	colCmd.AddCommand(colExecCmd)
	rootCmd.AddCommand(colCmd)
}
