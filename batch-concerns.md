# Batch Calculation Concerns

## Thread count accuracy

`task-calc.ts` snapshots `hackAnalyze` and `growthAnalyze` at call time. Both values depend on the server's security level when the script runs. If the server is at min security when `task-calc` runs, the thread counts are exact. If security has drifted up by the time the actual batch scripts execute, each hack thread steals slightly less than expected.

**Effect:** server sits slightly above 25% money after hack. Grow was sized for a 25% → 100% (4×) recovery, so it over-provisions slightly — but Bitburner caps grow at `maxMoney`, so the server still lands at max. Not a cascade failure on its own.

**Fixed:** `growMult` now uses `actualHackFrac = hackThreads * ns.hackAnalyze(target)` instead of the raw `hackPercent`, so grow correctly compensates for what `Math.ceil` on `hackThreads` actually steals.

## Timing drift (the real risk)

Hack, grow, and weaken times all scale with security. The calc snapshots these once and they are used as fixed offsets when scheduling the batch. If security is elevated at dispatch time, all three scripts take longer — but the absolute differences shift, so a batch scheduled with stale timing can land operations out of order (e.g. grow before weaken-hack).

**Weaken thread counts are safe:** `hackAnalyzeSecurity(threads)` and `growthAnalyzeSecurity(threads)` return fixed deltas based on thread count alone, independent of security level. Weakens will always exactly cancel the security added by hack and grow regardless of starting security.

## Early cycle cascade ("slow third cycle")

If a weaken from cycle N is late and security hasn't fully resolved before cycle N+1's hack lands:

1. Hack steals less than expected (elevated security reduces per-thread steal)
2. Grow and weaken timing windows shift relative to pre-calculated offsets
3. Operations from overlapping batches can land out of order
4. Each affected cycle leaves the server in a slightly wrong state, compounding over subsequent batches

This is most likely in the first few cycles before the pipeline stabilizes.

## What is not accounted for

- Thread counts and timings are calculated once at dispatch, not recalculated per batch
- No timing buffer is added to weaken offsets to absorb minor drift
- No detection of out-of-order landings or automatic re-prep

## Potential mitigations

- Recalculate `hackTime`/`growTime`/`weakenTime` at actual dispatch time rather than once upfront
- Add a small buffer (e.g. +200ms) to weaken offsets to absorb drift
- Re-prep the server (run prep-weaken + prep-grow cycle) if a batch lands detectably out of order
