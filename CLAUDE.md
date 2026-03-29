# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Project Does

**Larry** is a Bitburner game file synchronization tool with a terminal UI. It:
- Bridges local TypeScript/JavaScript development with the Bitburner game via WebSocket/JSON-RPC
- Watches local files and pushes changes to the game in real-time
- Pulls game files and Netscript type definitions on connection
- Provides a BubbleTea TUI for logs, notes, server browsing, and an in-game terminal

## Commands

```bash
make build   # Compile Go binary to bin/larry
make run     # Build and run
make dev     # TypeScript watch + run binary together
make fmt     # Format Go code
make lint    # go vet

# TypeScript only (scripts in scripts/src/ compile to scripts/dist/)
npm --prefix scripts run watch
```

## Architecture

```
main.go                           # Entry point: wires DB, TUI, WebSocket server, file watchers
cmd/
  root.go                         # Cobra CLI root; ExecuteCommand runs commands from the TUI terminal
  brain.go                        # brain start/tick/end subcommands
  col.go                          # col (batch calc) subcommand
  files.go                        # files subcommand
  ram.go                          # ram subcommand
  servers.go                      # servers subcommand
  ping.go                         # ping subcommand
internal/
  app/handlers.go                 # Core sync logic: OnConnect, OnEventDist, OnEventSrc
  communication/
    websocket.go                  # WebSocket server on :12525
    jsonrpc.go                    # Bitburner API client (pushFile, getFile, deleteFile, etc.)
  tui/
    model.go                      # BubbleTea state machine and shared model
    view.go                       # Lipgloss rendering
    tab_logs.go                   # Logs tab
    tab_notes.go                  # Notes tab
    tab_servers.go                # Servers tab
    tab_terminal.go               # Terminal tab
  filesync/filesync.go            # Filesystem watcher with debounce
  db/store.go                     # SQLite for notes
  logger/logger.go                # Event-based logging
  col/col.go                      # Col RPC client: scan, calc, crack, run, deploy via inbox/outbox protocol
  brain/brain.go                  # Batch hacking orchestration (BatchPlan, Brain)
  world/world.go                  # Game state structs (servers, player)
  mcp/server.go                   # MCP server exposing Larry tools to LLMs
scripts/
  src/                            # TypeScript source files (.ts, .script, .txt)
  dist/                           # Compiled JS output (synced to game)
```

**Data flow:**
1. Bitburner connects via WebSocket → `OnConnect` runs full bidirectional sync
2. Local file change in `dist/` → `OnEventDist` pushes to game via JSON-RPC
3. TypeScript change in `src/` → compiled to `dist/` → `OnEventDist` picks it up
4. All events bubble through the BubbleTea program for TUI updates

## Col — Game-Side RPC Protocol

Col is a Bitburner script (`col.js`) running on `home` that acts as a command controller. Communication uses a file-based inbox/outbox pattern:

1. Larry writes a JSON request to `/inbox/<id>.txt` on `home` via `PushFile`
2. Col picks it up, executes the action, writes result to `/outbox/<id>.txt`
3. Larry polls `GetFile` until the outbox file appears (30s timeout), then deletes both files

For actions returning large payloads (scan, calc), Col deploys a short-lived task script that POSTs results back to Larry's HTTP server. Larry pre-registers a channel via `conn.RegisterHTTP(id)` and waits on it.

**Go-side functions in `internal/col/col.go`:**
- `DoScan` — deploys `task-scan.js`, waits for HTTP callback, stores result in `CurrentWorld`
- `DoCalc` — deploys `task-calc.js`, returns thread/timing plan; auto-scans if `CurrentWorld` is nil
- `DoCrack` — inbox/outbox RPC to gain admin rights on servers
- `DoRun` — spreads a script across all servers with free RAM to hit a target thread count
- `RunScanner` — background ticker that calls `DoScan` on an interval
- `PickServer` — returns the server with most free RAM from `CurrentWorld`
- `TrackProcess` — updates `CurrentWorld` RAM usage and process list after launching a script

**Col actions (sent in the JSON `"action"` field):**
- `deploy` — copy a script to a server and run it
- `exec` — run a script already on a server
- `killall` — kill all scripts on one or all servers (preserves `col.js` on home)
- `crack` — nuke/BruteSSH/etc. to gain admin rights

## Key Design Points

- File sync only handles `.js`, `.txt`, and `.script` extensions
- `scripts/src/` `.script` and `.txt` files are copied directly to `dist/` (not compiled); only `.ts` is compiled
- The TUI uses BubbleTea v2 (charm.land/x/bubbletea) and Lipgloss v2 import paths
- Netscript type definitions (`NetscriptDefinitions.d.ts`) are pulled from the game on each connection
- Notes are persisted in `bin/notes.db` (SQLite)
