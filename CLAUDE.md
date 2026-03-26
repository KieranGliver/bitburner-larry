# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Project Does

**Larry** is a Bitburner game file synchronization tool with a terminal UI. It:
- Bridges local TypeScript/JavaScript development with the Bitburner game via WebSocket/JSON-RPC
- Watches local files and pushes changes to the game in real-time
- Pulls game files and Netscript type definitions on connection
- Provides a BubbleTea TUI for logs, connection status, and notes

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
cmd/main.go                       # Entry point: wires DB, TUI, HTTP server, file watchers
internal/
  app/handlers.go                 # Core sync logic: OnConnect, OnEventDist, OnEventSrc
  communication/
    http.go                       # WebSocket server on :12525
    jsonrpc.go                    # Bitburner API client (pushFile, getFile, deleteFile, etc.)
  tui/
    model.go                      # BubbleTea state machine (logsView, listView, titleView, bodyView)
    view.go                       # Lipgloss rendering
  filesync/filesync.go            # Filesystem watcher with debounce
  db/store.go                     # SQLite for notes
  logger/logger.go                # Event-based logging
  brain/world.go                  # Game state structs (currently unused)
scripts/
  src/                            # TypeScript source files (.ts, .script, .txt)
  dist/                           # Compiled JS output (synced to game)
```

**Data flow:**
1. Bitburner connects via WebSocket → `OnConnect` runs full bidirectional sync
2. Local file change in `dist/` → `OnEventDist` pushes to game via JSON-RPC
3. TypeScript change in `src/` → compiled to `dist/` → `OnEventDist` picks it up
4. All events bubble through the BubbleTea program for TUI updates

## Key Design Points

- File sync only handles `.js`, `.txt`, and `.script` extensions
- `scripts/src/` `.script` and `.txt` files are copied directly to `dist/` (not compiled); only `.ts` is compiled
- The TUI uses BubbleTea v2 (charm.land/x/bubbletea) and Lipgloss v2 import paths
- Netscript type definitions (`NetscriptDefinitions.d.ts`) are pulled from the game on each connection
- Notes are persisted in `bin/notes.db` (SQLite)
