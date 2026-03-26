# Larry

A local development environment for [Bitburner](https://github.com/bitburner-official/bitburner-src) with real-time file synchronization and a terminal UI.

## What it does

- Watches your local `scripts/` directory and pushes changes to Bitburner automatically
- Pulls Netscript type definitions from the game so you get TypeScript autocomplete
- Seeds your local environment with existing game files on connection
- Provides a TUI for viewing sync logs and managing notes

## Requirements

- Go 1.21+
- Node.js (for TypeScript compilation)
- Bitburner with the [Remote API](https://github.com/bitburner-official/bitburner-src/blob/dev/markdown/bitburner.ns.getporthandle.md) enabled

## Setup

```bash
# Install TypeScript dependencies
npm --prefix scripts install
```

In Bitburner, enable the remote API server at the default port (12525).

## Usage

```bash
make dev    # Start Larry + TypeScript compiler in watch mode
```

Or separately:

```bash
make run                          # Run Larry only
npm --prefix scripts run watch    # TypeScript watch mode only
```

Once running, connect Bitburner to `ws://localhost:12525`. Larry will automatically sync all files.

## Writing Scripts

Put TypeScript or Javascript files in `scripts/src/`. They compile to `scripts/dist/` and are pushed to the game automatically.

`.script` and `.txt` files in `scripts/src/` are copied directly to `scripts/dist/` without compilation.

### TUI Keybindings

| Key | Action |
|-----|--------|
| Tab | Switch between Logs and Notes |
| k / up | Scroll up |
| j / down | Scroll down |
| Ctrl+S | Save note |
| q | Quit |

## Build

```bash
make build   # Compile to bin/larry
make fmt     # Format Go code
make lint    # Run go vet
