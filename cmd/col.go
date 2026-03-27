package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync/atomic"
	"time"

	"github.com/KieranGliver/bitburner-larry/internal/brain"
	"github.com/KieranGliver/bitburner-larry/internal/communication"
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
	return colRPCWith(currentConn, id, req)
}

// colRPCWith is like colRPC but uses an explicit connection.
func colRPCWith(conn *communication.BitburnerConn, id string, req map[string]any) (string, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	inboxPath := fmt.Sprintf("/inbox/%s.txt", id)
	outboxPath := fmt.Sprintf("/outbox/%s.txt", id)

	if err := conn.PushFile(ctx, "home", inboxPath, string(payload)); err != nil {
		return "", fmt.Errorf("error sending command: %w", err)
	}

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
		result, err := conn.GetFile(ctx, "home", outboxPath)
		if err != nil || result == "" {
			continue
		}
		_ = conn.DeleteFile(ctx, "home", outboxPath)
		_ = conn.DeleteFile(ctx, "home", inboxPath)
		return result, nil
	}
	return "", fmt.Errorf("timeout: no response from Col within 30s")
}

// DoScan performs a single full world scan via Col and returns the result.
func DoScan(conn *communication.BitburnerConn) (*brain.World, error) {
	id := colNextID()
	ch := conn.RegisterHTTP(id)

	ackResult, err := colRPCWith(conn, id, map[string]any{
		"id": id, "action": "deploy",
		"server": "foodnstuff", "script": "task-scan.js",
		"threads": 1, "args": []any{id},
	})
	if err != nil {
		return nil, err
	}
	var ack colResponse
	if err := json.Unmarshal([]byte(ackResult), &ack); err != nil || !ack.Success {
		msg := ack.Error
		if err != nil {
			msg = err.Error()
		}
		return nil, fmt.Errorf("deploy failed: %s", msg)
	}

	select {
	case data := <-ch:
		var resp colScanResponse
		if err := json.Unmarshal([]byte(data), &resp); err != nil {
			return nil, fmt.Errorf("parse scan data: %w", err)
		}
		if !resp.Success {
			return nil, fmt.Errorf("scan failed: %s", resp.Error)
		}
		world := &brain.World{Player: resp.Player, Servers: resp.Servers}
		CurrentWorld = world
		return world, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for task-scan.js")
	}
}

// DoCrack cracks all crackable servers via Col and returns the cracked/failed lists.
func DoCrack(conn *communication.BitburnerConn) (cracked []string, failed []string, err error) {
	id := colNextID()
	result, err := colRPCWith(conn, id, map[string]any{
		"id": id, "action": "crack", "targets": []string{},
	})
	if err != nil {
		return nil, nil, err
	}
	var resp colCrackResponse
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		return nil, nil, fmt.Errorf("parse crack response: %w", err)
	}
	return resp.Cracked, resp.Failed, nil
}

// RunScanner runs a background loop that scans the world every interval and calls
// onWorld with each result. It stops when ctx is cancelled.
func RunScanner(conn *communication.BitburnerConn, ctx context.Context, interval time.Duration, onWorld func(*brain.World)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			world, err := DoScan(conn)
			if err != nil {
				continue
			}
			onWorld(world)
		}
	}
}

// CurrentCalc holds the most recent result from "col calc".
var CurrentCalc *colCalcResponse

// pickServer returns the hostname with the most free RAM that can fit at least ramNeeded GB.
// Returns an error if CurrentWorld is nil or no eligible server is found.
func pickServer(ramNeeded float64) (string, error) {
	if CurrentWorld == nil {
		return "", fmt.Errorf("no world data — run col scan first")
	}
	best := ""
	bestFree := 0.0
	for _, s := range CurrentWorld.Servers {
		if !s.HasAdminRights {
			continue
		}
		free := s.MaxRam - s.RamUsed
		if free >= ramNeeded && free > bestFree {
			best = s.Hostname
			bestFree = free
		}
	}
	if best == "" {
		return "", fmt.Errorf("no server with %.2f GB free RAM available", ramNeeded)
	}
	return best, nil
}

// trackProcess records a newly launched process in CurrentWorld: adds the process entry
// and charges its RAM against the server. Silent no-op if CurrentWorld is nil.
func trackProcess(hostname, script string, pid, threads int, args []any) {
	if CurrentWorld == nil || currentConn == nil {
		return
	}
	ram, err := currentConn.CalculateRam(context.Background(), "home", script)
	if err != nil || ram <= 0 {
		return
	}
	CurrentWorld.UpdateRam(hostname, ram*float64(threads))
	CurrentWorld.AddProcess(hostname, brain.Process{
		Pid:      uint(pid),
		Filename: script,
		Hostname: hostname,
		Threads:  uint(threads),
		Args:     args,
	})
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
			trackProcess(args[0], args[1], resp.PID, threads, scriptArgs)
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
			trackProcess(args[0], args[1], resp.PID, threads, scriptArgs)
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
		trackProcess(server, "task-ping.js", ack.PID, 1, []any{id})

		select {
		case <-ch:
			fmt.Fprintln(cmd.OutOrStdout(), "pong: col→HTTP round-trip ok")
		case <-time.After(10 * time.Second):
			fmt.Fprintln(cmd.OutOrStdout(), "timeout: no response from task-ping.js")
		}
	},
}

type runDispatchResult struct {
	Server  string `json:"server"`
	Threads int    `json:"threads"`
	PID     int    `json:"pid"`
}

type runResult struct {
	Script           string               `json:"script"`
	ThreadsRequested int                  `json:"threads_requested"`
	ThreadsScheduled int                  `json:"threads_scheduled"`
	ThreadsRemaining int                  `json:"threads_remaining"`
	Dispatches       []runDispatchResult  `json:"dispatches"`
	Errors           []string             `json:"errors"`
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

		ctx := context.Background()

		ramPerThread, err := currentConn.CalculateRam(ctx, "home", script)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "error getting RAM cost for %s: %v\n", script, err)
			return
		}
		if ramPerThread <= 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "error: %s reports 0 GB RAM cost\n", script)
			return
		}

		world := CurrentWorld
		if world == nil {
			if !asJSON {
				fmt.Fprintln(cmd.OutOrStdout(), "no world data — scanning...")
			}
			world, err = DoScan(currentConn)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "scan failed: %v\n", err)
				return
			}
		}

		type slot struct {
			hostname string
			capacity int
		}
		var slots []slot
		for _, s := range world.Servers {
			if !s.HasAdminRights {
				continue
			}
			free := s.MaxRam - s.RamUsed
			cap := int(free / ramPerThread)
			if cap > 0 {
				slots = append(slots, slot{s.Hostname, cap})
			}
		}
		sort.Slice(slots, func(i, j int) bool {
			return slots[i].capacity > slots[j].capacity
		})

		remaining := threads
		type pending struct {
			hostname string
			threads  int
		}
		var toDispatch []pending
		for _, s := range slots {
			if remaining <= 0 {
				break
			}
			t := s.capacity
			if t > remaining {
				t = remaining
			}
			toDispatch = append(toDispatch, pending{s.hostname, t})
			remaining -= t
		}

		result := runResult{
			Script:           script,
			ThreadsRequested: threads,
			Dispatches:       []runDispatchResult{},
			Errors:           []string{},
		}

		if !asJSON {
			fmt.Fprintf(cmd.OutOrStdout(), "ram cost: %.2f GB/thread\n", ramPerThread)
			if len(toDispatch) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "no servers with enough free RAM\n")
				return
			}
		} else if len(toDispatch) == 0 {
			result.ThreadsRemaining = threads
			out, _ := json.MarshalIndent(result, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return
		}

		for _, d := range toDispatch {
			id := colNextID()
			raw, err := colRPC(id, map[string]any{
				"id": id, "action": "deploy",
				"server": d.hostname, "script": script,
				"threads": d.threads, "args": scriptArgs,
			})
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", d.hostname, err))
				if !asJSON {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s (%d threads): error — %v\n", d.hostname, d.threads, err)
				}
				continue
			}
			var resp colResponse
			if err := json.Unmarshal([]byte(raw), &resp); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: parse error: %v", d.hostname, err))
				if !asJSON {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s (%d threads): parse error — %v\n", d.hostname, d.threads, err)
				}
				continue
			}
			if resp.Success {
				if CurrentWorld != nil {
					CurrentWorld.UpdateRam(d.hostname, ramPerThread*float64(d.threads))
					CurrentWorld.AddProcess(d.hostname, brain.Process{
						Pid:      uint(resp.PID),
						Filename: script,
						Hostname: d.hostname,
						Threads:  uint(d.threads),
						Args:     scriptArgs,
					})
				}
				result.Dispatches = append(result.Dispatches, runDispatchResult{d.hostname, d.threads, resp.PID})
				result.ThreadsScheduled += d.threads
				if !asJSON {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s: %d threads (pid %d)\n", d.hostname, d.threads, resp.PID)
				}
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", d.hostname, resp.Error))
				if !asJSON {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s (%d threads): failed — %s\n", d.hostname, d.threads, resp.Error)
				}
			}
		}

		result.ThreadsRemaining = threads - result.ThreadsScheduled

		if asJSON {
			out, _ := json.MarshalIndent(result, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(out))
		} else {
			if result.ThreadsRemaining > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "warning: %d/%d threads unscheduled — not enough free RAM\n", result.ThreadsRemaining, threads)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "done: %d scheduled, %d remaining\n", result.ThreadsScheduled, result.ThreadsRemaining)
		}
	},
}

type colCalcResponse struct {
	ID                string  `json:"id"`
	Success           bool    `json:"success"`
	Error             string  `json:"error"`
	Target            string  `json:"target"`
	HackPercent       float64 `json:"hackPercent"`
	PrepWeakenThreads     int     `json:"prepWeakenThreads"`
	PrepGrowThreads       int     `json:"prepGrowThreads"`
	PrepGrowWeakenThreads int     `json:"prepGrowWeakenThreads"`
	HackThreads       int     `json:"hackThreads"`
	GrowThreads       int     `json:"growThreads"`
	WeakenHackThreads int     `json:"weakenHackThreads"`
	WeakenGrowThreads int     `json:"weakenGrowThreads"`
	HackTime          float64 `json:"hackTime"`
	GrowTime          float64 `json:"growTime"`
	WeakenTime        float64 `json:"weakenTime"`
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

		id := colNextID()
		ch := currentConn.RegisterHTTP(id)

		ctx := context.Background()
		ramPerThread, err := currentConn.CalculateRam(ctx, "home", "task-calc.js")
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "error getting RAM cost: %v\n", err)
			return
		}

		if CurrentWorld == nil {
			if !asJSON {
				fmt.Fprintln(cmd.OutOrStdout(), "no world data — scanning...")
			}
			if _, err := DoScan(currentConn); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "scan failed: %v\n", err)
				return
			}
		}

		host, err := pickServer(ramPerThread)
		if err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), err)
			return
		}

		if !asJSON {
			fmt.Fprintf(cmd.OutOrStdout(), "sent %s, calculating for %s (via %s)...\n", id, target, host)
		}
		ackRaw, err := colRPC(id, map[string]any{
			"id": id, "action": "deploy",
			"server": host, "script": "task-calc.js",
			"threads": 1, "args": []any{id, target, hackPercent},
		})
		if err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), err)
			return
		}
		var ack colResponse
		if err := json.Unmarshal([]byte(ackRaw), &ack); err != nil || !ack.Success {
			msg := ack.Error
			if err != nil {
				msg = err.Error()
			}
			fmt.Fprintf(cmd.OutOrStdout(), "deploy failed: %s\n", msg)
			return
		}
		trackProcess(host, "task-calc.js", ack.PID, 1, []any{id, target, hackPercent})

		select {
		case data := <-ch:
			var resp colCalcResponse
			if err := json.Unmarshal([]byte(data), &resp); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "error parsing calc data: %v\n", err)
				return
			}
			if !resp.Success {
				fmt.Fprintf(cmd.OutOrStdout(), "failed: %s\n", resp.Error)
				return
			}
			CurrentCalc = &resp
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
		case <-time.After(30 * time.Second):
			fmt.Fprintln(cmd.OutOrStdout(), "timeout waiting for task-calc.js")
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
