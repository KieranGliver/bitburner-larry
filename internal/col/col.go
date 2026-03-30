package col

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync/atomic"
	"time"

	"github.com/KieranGliver/bitburner-larry/internal/communication"
	"github.com/KieranGliver/bitburner-larry/internal/world"
)

var colRequestCounter int64

// response types

type ColResponse struct {
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
	Player  world.Player      `json:"player"`
	Servers []world.BitServer `json:"servers"`
}

type ColKillAllResponse struct {
	ID      string   `json:"id"`
	Success bool     `json:"success"`
	Killed  []string `json:"killed"`
	Error   string   `json:"error"`
}

type ColCalcResponse struct {
	ID                    string  `json:"id"`
	Success               bool    `json:"success"`
	Error                 string  `json:"error"`
	Target                string  `json:"target"`
	HackPercent           float64 `json:"hackPercent"`
	PrepWeakenThreads     int     `json:"prepWeakenThreads"`
	PrepGrowThreads       int     `json:"prepGrowThreads"`
	PrepGrowWeakenThreads int     `json:"prepGrowWeakenThreads"`
	HackThreads           int     `json:"hackThreads"`
	GrowThreads           int     `json:"growThreads"`
	WeakenHackThreads     int     `json:"weakenHackThreads"`
	WeakenGrowThreads     int     `json:"weakenGrowThreads"`
	HackTime              float64 `json:"hackTime"`
	GrowTime              float64 `json:"growTime"`
	WeakenTime            float64 `json:"weakenTime"`
}

type RunDispatchResult struct {
	Server  string `json:"server"`
	Threads int    `json:"threads"`
	PID     int    `json:"pid"`
}

type RunResult struct {
	Script           string              `json:"script"`
	ThreadsRequested int                 `json:"threads_requested"`
	ThreadsScheduled int                 `json:"threads_scheduled"`
	ThreadsRemaining int                 `json:"threads_remaining"`
	Dispatches       []RunDispatchResult `json:"dispatches"`
	Errors           []string            `json:"errors"`
}

// ColRPCWith sends a request to the Col inbox and waits for a response in the outbox.
func ColRPCWith(conn *communication.BitburnerConn, id string, req map[string]any) (string, error) {
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

// ColNextID generates a unique request ID.
func ColNextID() string {
	n := atomic.AddInt64(&colRequestCounter, 1)
	return fmt.Sprintf("COL-%03d", n)
}

// TrackProcess records a newly launched process in CurrentWorld.
func TrackProcess(w *world.World, conn *communication.BitburnerConn, hostname, script string, pid, threads int, args []any) {
	if w == nil {
		return
	}
	ram, err := conn.CalculateRam(context.Background(), "home", script)
	if err != nil || ram <= 0 {
		return
	}
	w.UpdateRam(hostname, ram*float64(threads))
	w.AddProcess(hostname, world.Process{
		Pid:      uint(pid),
		Filename: script,
		Hostname: hostname,
		Threads:  uint(threads),
		Args:     args,
	})
}

// PickServer returns the hostname with the most free RAM that can fit at least ramNeeded GB.
// Returns an error if CurrentWorld is nil or no eligible server is found.
func PickServer(w *world.World, ramNeeded float64) (string, error) {
	if w == nil {
		return "", fmt.Errorf("no world data — run col scan first")
	}
	best := ""
	bestFree := 0.0
	for _, s := range w.Servers {
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

// DoScan performs a single full world scan via Col and returns the result.
// server is the host to deploy task-scan.js on; defaults to "foodnstuff" if empty.
func DoScan(conn *communication.BitburnerConn, server string) (*world.World, error) {
	if server == "" {
		server = "foodnstuff"
	}
	id := ColNextID()
	ch := conn.RegisterHTTP(id)

	ackResult, err := ColRPCWith(conn, id, map[string]any{
		"id": id, "action": "deploy",
		"server": server, "script": "task-scan.js",
		"threads": 1, "args": []any{id},
	})
	if err != nil {
		return nil, err
	}
	var ack ColResponse
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
		w := &world.World{Player: resp.Player, Servers: resp.Servers}
		return w, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for task-scan.js")
	}
}

// DoCrack cracks servers via Col and returns the cracked/failed lists.
// Pass empty targets to crack all crackable servers.
func DoCrack(conn *communication.BitburnerConn, targets []string) (cracked []string, failed []string, err error) {
	if targets == nil {
		targets = []string{}
	}
	id := ColNextID()
	result, err := ColRPCWith(conn, id, map[string]any{
		"id": id, "action": "crack", "targets": targets,
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

// DoCalc calculates hack/grow/weaken thread counts for target via Col.
// hackPercent is the fraction of max money to steal (e.g. 0.75); pass 0 to use the default (0.75).
// Scans the world first if CurrentWorld is nil.
func DoCalc(w *world.World, conn *communication.BitburnerConn, target string, hackPercent float64) (*ColCalcResponse, error) {
	if hackPercent <= 0 {
		hackPercent = 0.75
	}

	ctx := context.Background()
	ramPerThread, err := conn.CalculateRam(ctx, "home", "task-calc.js")
	if err != nil {
		return nil, fmt.Errorf("error getting RAM cost: %w", err)
	}

	if w == nil {
		return nil, fmt.Errorf("no world data — run col scan first")
	}

	host, err := PickServer(w, ramPerThread)
	if err != nil {
		return nil, err
	}

	id := ColNextID()
	ch := conn.RegisterHTTP(id)

	ackRaw, err := ColRPCWith(conn, id, map[string]any{
		"id": id, "action": "deploy",
		"server": host, "script": "task-calc.js",
		"threads": 1, "args": []any{id, target, hackPercent},
	})
	if err != nil {
		return nil, err
	}
	var ack ColResponse
	if err := json.Unmarshal([]byte(ackRaw), &ack); err != nil || !ack.Success {
		msg := ack.Error
		if err != nil {
			msg = err.Error()
		}
		return nil, fmt.Errorf("deploy failed: %s", msg)
	}
	TrackProcess(w, conn, host, "task-calc.js", ack.PID, 1, []any{id, target, hackPercent})

	select {
	case data := <-ch:
		var resp ColCalcResponse
		if err := json.Unmarshal([]byte(data), &resp); err != nil {
			return nil, fmt.Errorf("parse calc data: %w", err)
		}
		if !resp.Success {
			return nil, fmt.Errorf("calc failed: %s", resp.Error)
		}
		return &resp, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for task-calc.js")
	}
}

// DoRun spreads script across all servers with free RAM to hit the target thread count.
// Pass args as the script arguments. Returns an error if w is nil.
func DoRun(w *world.World, conn *communication.BitburnerConn, script string, threads int, args []any) (*RunResult, error) {
	ctx := context.Background()
	ramPerThread, err := conn.CalculateRam(ctx, "home", script)
	if err != nil {
		return nil, fmt.Errorf("error getting RAM cost for %s: %w", script, err)
	}
	if ramPerThread <= 0 {
		return nil, fmt.Errorf("%s reports 0 GB RAM cost", script)
	}

	if w == nil {
		return nil, fmt.Errorf("no world data — run col scan first")
	}

	type slot struct {
		hostname string
		capacity int
	}
	var slots []slot
	for _, s := range w.Servers {
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

	result := &RunResult{
		Script:           script,
		ThreadsRequested: threads,
		Dispatches:       []RunDispatchResult{},
		Errors:           []string{},
	}

	for _, d := range toDispatch {
		id := ColNextID()
		raw, err := ColRPCWith(conn, id, map[string]any{
			"id": id, "action": "deploy",
			"server": d.hostname, "script": script,
			"threads": d.threads, "args": args,
		})
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", d.hostname, err))
			continue
		}
		var resp ColResponse
		if err := json.Unmarshal([]byte(raw), &resp); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: parse error: %v", d.hostname, err))
			continue
		}
		if resp.Success {
			w.UpdateRam(d.hostname, ramPerThread*float64(d.threads))
			w.AddProcess(d.hostname, world.Process{
				Pid:      uint(resp.PID),
				Filename: script,
				Hostname: d.hostname,
				Threads:  uint(d.threads),
				Args:     args,
			})
			result.Dispatches = append(result.Dispatches, RunDispatchResult{d.hostname, d.threads, resp.PID})
			result.ThreadsScheduled += d.threads
		} else {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", d.hostname, resp.Error))
		}
	}

	result.ThreadsRemaining = threads - result.ThreadsScheduled
	return result, nil
}

// RunScanner runs a background loop that scans the world every interval and calls
// onWorld with each result. It stops when ctx is cancelled.
func RunScanner(conn *communication.BitburnerConn, ctx context.Context, interval time.Duration, onWorld func(*world.World)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w, err := DoScan(conn, "")
			if err != nil {
				continue
			}
			onWorld(w)
		}
	}
}
