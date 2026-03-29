# Larry

Bitburner file sync tool — watches local TS/JS files and pushes changes to the game via WebSocket, with a BubbleTea TUI.

## What it does

- Watches your local `scripts/` directory and pushes changes to Bitburner automatically
- Pulls Netscript type definitions from the game so you get TypeScript autocomplete
- Seeds your local environment with existing game files on connection
- Provides a TUI with tabs for sync logs, notes, server browser, and an in-game terminal

## Requirements

- Go 1.21+
- Node.js (for TypeScript compilation)
- Bitburner with the Remote API enabled (default port 12525)

## Setup

```bash
# Install TypeScript dependencies
npm --prefix scripts install
```

## Usage

```bash
make dev    # Start Larry + TypeScript compiler in watch mode
```

Or separately:

```bash
make run                          # Run Larry only
npm --prefix scripts run watch    # TypeScript watch mode only
```

Once running, connect Bitburner to Hostname: `localhost` Port: `12525`. Larry will automatically sync all files.

## Writing Scripts

Put TypeScript or JavaScript files in `scripts/src/`. They compile to `scripts/dist/` and are pushed to the game automatically.

`.script` and `.txt` files in `scripts/src/` are copied directly to `scripts/dist/` without compilation.

## TUI

Larry has three tabs (cycle with `Tab`) plus a terminal overlay:

| Tab | Description |
|-----|-------------|
| Logs | Real-time sync event log |
| Notes | Persistent notes (saved to SQLite) |
| Servers | Browse game servers |

### Keybindings

| Key | Action |
|-----|--------|
| Tab | Cycle tabs (Logs → Notes → Servers) |
| k / up | Scroll up |
| j / down | Scroll down |
| Enter | Select |
| Esc | Go back |
| Ctrl+S | Save |
| Ctrl+T | Open / close terminal |
| q | Quit |

## Col (Child of Larry)

Col is a game-side daemon (`col.js`) that must be running in Bitburner for the `col` and `brain` terminal commands to work. Start it once from the Bitburner terminal:

```
run col.js
```

It will keep running in the background. After connecting Col will automatically scan the state of the bitburner world every 5 seconds. You can then use `col run -t 10 hack.script n00dles`, `col calc`, `brain start`, etc. from Larry's terminal.

### Terminal

Press `Ctrl+T` from anywhere to open the terminal overlay. It supports command history (up/down arrows) and persists history across sessions in `bin/.cmdlog`.

| Key | Action |
|-----|--------|
| Enter | Run command |
| ↑ / ↓ | Navigate history |
| Ctrl+D | View last output in log detail |
| Ctrl+C | Close terminal |

#### All commands

**General**

| Command | Description |
|---------|-------------|
| `ping` | Show connection status |
| `servers` | List all servers with admin/purchased flags |
| `files <server>` | List files on a server |
| `ram <file> <server>` | Show RAM cost of a script |

**Col** — requires `col.js` running on `home` in Bitburner

| Command | Description |
|---------|-------------|
| `col scan [-s server]` | Scan all servers and update world state |
| `col crack [server]` | Gain admin rights on crackable servers (all if omitted) |
| `col calc <target> [--hack-percent N]` | Calculate hack/grow/weaken thread counts and timings |
| `col run <script> -t N [args...]` | Spread a script across all servers to hit N total threads |
| `col exec <server> <script> -t N [args...]` | Run a script already on a server |
| `col deploy <server> <script> -t N [args...]` | Copy a script to a server and run it |
| `col killall [server]` | Kill all scripts on a server (or all servers) |
| `col ping [-s server]` | Test the Col → HTTP round-trip |

**Brain** — autonomous hacking loop, requires a prior `col scan`

| Command | Description |
|---------|-------------|
| `brain start` | Start the hacking brain loop (ticks every 10s) |
| `brain tick` | Run a single brain tick immediately |
| `brain end` | Stop the brain loop (state is preserved) |

## Build

```bash
make build   # Compile to bin/larry
make fmt     # Format Go code
make lint    # Run go vet
