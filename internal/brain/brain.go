package brain

import (
	"context"
	"fmt"
	"sort"

	col "github.com/KieranGliver/bitburner-larry/internal/col"
	"github.com/KieranGliver/bitburner-larry/internal/communication"
	"github.com/KieranGliver/bitburner-larry/internal/logger"
	"github.com/KieranGliver/bitburner-larry/internal/world"
)

type BatchPlan struct {
	col.ColCalcResponse
	Pids []uint
}

type Brain struct {
	batchMap map[string]BatchPlan
	rank     func(a, b world.BitServer) bool
	onLog    func(string, logger.Level)
}

func New(rank func(a, b world.BitServer) bool, onLog func(string, logger.Level)) *Brain {
	return &Brain{
		batchMap: make(map[string]BatchPlan),
		rank:     rank,
		onLog:    onLog,
	}
}

func findByPID(procs []world.Process, id uint) (int, world.Process) {
	for i, p := range procs {
		if p.Pid == id {
			return i, p
		}
	}
	return -1, world.Process{}
}

func (b *Brain) Tick(w *world.World, conn *communication.BitburnerConn) {
	ctx := context.Background()
	// Ensure we have enough ram can run the scan script with at least one thread
	var err error
	ram, err := conn.CalculateRam(ctx, "home", "task-calc.js")
	if err != nil {
		b.onLog(fmt.Sprintf("Error on CalculateRam: %v", err), logger.ERROR)
	}
	_, err = col.PickServer(ram)
	if err != nil {
		return
	}
	// Sort servers in the world by rank function
	servers := make([]world.BitServer, len(w.Servers))
	copy(servers, w.Servers)
	sort.Slice(servers, func(i, j int) bool {
		return b.rank(servers[i], servers[j])
	})

	procs := w.GetAllProcess()
	// until we have no more threads to run
	// For ever server in the list, desc from top rank to bottom
	for _, target := range servers {
		if !target.HasAdminRights {
			continue
		}
		if uint(w.Player.Skills.Hacking) < target.RequiredHackingSkill {
			continue
		}
		batchPlan, exists := b.batchMap[target.Hostname]
		// Check if active pids are still running if > 0 continue
		if exists {
			anyRunning := false
			for _, pid := range batchPlan.Pids {
				if i, _ := findByPID(procs, pid); i != -1 {
					anyRunning = true
					break
				}
			}
			if anyRunning {
				continue
			}
		}

		pids := []uint{}
		calcResp, err := col.DoCalc(conn, target.Hostname, 0.25)
		if err != nil {
			b.onLog(fmt.Sprintf("Error on doCalc: %v", err), logger.ERROR)
			return
		}

		/**
		 *  prep-weaken:         11 threads    78.1s  (not ready)
		 *  prep-grow:            4 threads    62.5s  (not ready)
		 *  prep-grow-weaken:     1 threads    78.1s  (not ready)
		 */
		// If any prep threads exist run the prep batch and continue
		// 		col run grow.script -t 4 n00dles
		//		col run weak.script -t 12 n00dles
		// When no prep needs to be done we move onto the batch
		/**
		 *  hack:               122 threads    19.5s
		 *  weaken-hack:          5 threads    78.1s
		 */

		if calcResp.PrepGrowThreads+calcResp.PrepGrowWeakenThreads+calcResp.PrepWeakenThreads > 0 {
			growResult, err := col.DoRun(conn, "grow.script", calcResp.PrepGrowThreads, []any{target.Hostname})
			if err != nil {
				b.onLog(fmt.Sprintf("Error running grow.script on %v: %v", target.Hostname, err), logger.ERROR)
				return
			}
			weakResult, err := col.DoRun(conn, "weak.script", calcResp.PrepGrowWeakenThreads+calcResp.PrepWeakenThreads, []any{target.Hostname})
			if err != nil {
				b.onLog(fmt.Sprintf("Error running weak.script on %v: %v", target.Hostname, err), logger.ERROR)
				return
			}
			for _, d := range growResult.Dispatches {
				pids = append(pids, uint(d.PID))
			}
			for _, d := range weakResult.Dispatches {
				pids = append(pids, uint(d.PID))
			}
			if growResult.ThreadsRemaining+weakResult.ThreadsRemaining > 0 {
				break
			}
			continue
		}
		// At this point we are confident that the server is max money min security
		// We attempt to launch biggest hack
		// 		col run hack.script -t 122 n00dles
		//		col run weak.script -t 5 n00dles

		// Should be 1.75 GB weak is always higher than hack (1.70 GB)
		weakRam, err := conn.CalculateRam(ctx, "home", "weak.script")
		if err != nil {
			b.onLog(fmt.Sprintf("Error on CalculateRam: %v", err), logger.ERROR)
		}
		budget := min(w.GetThreads(weakRam), calcResp.HackThreads+calcResp.WeakenHackThreads)
		hackThreads := (budget - 1) * 25 / 26
		weakenThreads := hackThreads/25 + 1

		hackResult, err := col.DoRun(conn, "hack.script", hackThreads, []any{target.Hostname})
		if err != nil {
			b.onLog(fmt.Sprintf("Error running hack.script on %v: %v", target.Hostname, err), logger.ERROR)
			return
		}
		weakResult, err := col.DoRun(conn, "weak.script", weakenThreads, []any{target.Hostname})
		if err != nil {
			b.onLog(fmt.Sprintf("Error running weak.script on %v: %v", target.Hostname, err), logger.ERROR)
			return
		}

		for _, d := range hackResult.Dispatches {
			pids = append(pids, uint(d.PID))
		}
		for _, d := range weakResult.Dispatches {
			pids = append(pids, uint(d.PID))
		}

		b.batchMap[target.Hostname] = BatchPlan{ColCalcResponse: *calcResp, Pids: pids}
		if budget < calcResp.HackThreads+calcResp.WeakenHackThreads {
			break
		}
	}
	// IF we have enough more threads move onto the next server in the list.
	// Continue until all threads are used or no more servers in the lsit

	// What happens if the algorthim runing scripts runs out of room at any point of the algorthim?
	// Does the cycle break? Will it fix itself? Should we wait to run all the scripts we need for current batch or
	// Don't care and will do it when next tick comes on better server potentiallty.
}
