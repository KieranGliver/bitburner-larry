package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	larcmd "github.com/KieranGliver/bitburner-larry/cmd"
	"github.com/KieranGliver/bitburner-larry/internal/app"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type McpServer struct {
	appState *app.AppState
	onCall   func(input, result string)
}

func New(fn func(input, result string), as *app.AppState) *McpServer {
	s := &McpServer{onCall: fn, appState: as}
	return s
}

func (s *McpServer) run(input string) string {
	result := larcmd.ExecuteCommand(input, s.appState)
	if s.onCall != nil {
		s.onCall(input, result)
	}
	return result
}

func (s *McpServer) Serve(port string) {
	mcpSrv := server.NewMCPServer("bitburner-larry", "1.0.0")

	mcpSrv.AddTool(mcp.NewTool("ping",
		mcp.WithDescription("Check if Bitburner is connected"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(s.run("ping")), nil
	})

	mcpSrv.AddTool(mcp.NewTool("servers",
		mcp.WithDescription("List all servers in the Bitburner game"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(s.run("servers")), nil
	})

	mcpSrv.AddTool(mcp.NewTool("files",
		mcp.WithDescription("List files on a Bitburner server"),
		mcp.WithString("server", mcp.Required(), mcp.Description("Server hostname")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		serverName := mcp.ParseString(req, "server", "")
		return mcp.NewToolResultText(s.run("files " + serverName)), nil
	})

	mcpSrv.AddTool(mcp.NewTool("ram",
		mcp.WithDescription("Calculate RAM cost of a script on a server"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Script filename")),
		mcp.WithString("server", mcp.Required(), mcp.Description("Server hostname")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		file := mcp.ParseString(req, "file", "")
		serverName := mcp.ParseString(req, "server", "")
		return mcp.NewToolResultText(s.run(fmt.Sprintf("ram %s %s", file, serverName))), nil
	})

	mcpSrv.AddTool(mcp.NewTool("scan",
		mcp.WithDescription("Collect full world state from Bitburner via Col and cache it"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(s.run("col scan foodnstuff")), nil
	})

	mcpSrv.AddTool(mcp.NewTool("world",
		mcp.WithDescription("Return the cached world state (player stats + all servers) as JSON; run scan first to populate it"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		world := s.appState.World()
		if world == nil {
			return mcp.NewToolResultText("no world data — run scan first"), nil
		}
		out, err := json.MarshalIndent(world, "", "  ")
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("error serializing world: %v", err)), nil
		}
		result := string(out)
		if s.onCall != nil {
			s.onCall("world", result)
		}
		return mcp.NewToolResultText(result), nil
	})

	mcpSrv.AddTool(mcp.NewTool("calc",
		mcp.WithDescription("Calculate hack/grow/weaken thread counts and timings for a target server"),
		mcp.WithString("target", mcp.Required(), mcp.Description("Target server hostname")),
		mcp.WithString("hack_percent", mcp.Description("Fraction of max money to steal per hack (default 0.75)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		target := mcp.ParseString(req, "target", "")
		hackPercent := mcp.ParseString(req, "hack_percent", "0.75")
		return mcp.NewToolResultText(s.run(fmt.Sprintf("col calc %s --hack-percent %s --json", target, hackPercent))), nil
	})

	mcpSrv.AddTool(mcp.NewTool("run",
		mcp.WithDescription("Spread a script across all available servers to hit a target thread count"),
		mcp.WithString("script", mcp.Required(), mcp.Description("Script filename (e.g. hack.js)")),
		mcp.WithString("threads", mcp.Required(), mcp.Description("Total number of threads to spread across servers")),
		mcp.WithString("args", mcp.Description("Space-separated extra arguments to pass to the script")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		script := mcp.ParseString(req, "script", "")
		threads := mcp.ParseString(req, "threads", "1")
		args := mcp.ParseString(req, "args", "")
		cmd := fmt.Sprintf("col run %s --threads %s --json", script, threads)
		if args != "" {
			cmd += " " + args
		}
		return mcp.NewToolResultText(s.run(cmd)), nil
	})

	mcpSrv.AddTool(mcp.NewTool("exec",
		mcp.WithDescription("Execute a script on a specific server via Col"),
		mcp.WithString("server", mcp.Required(), mcp.Description("Server hostname to run the script on")),
		mcp.WithString("script", mcp.Required(), mcp.Description("Script filename")),
		mcp.WithString("threads", mcp.Description("Number of threads (default 1)")),
		mcp.WithString("args", mcp.Description("Space-separated extra arguments to pass to the script")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		serverName := mcp.ParseString(req, "server", "")
		script := mcp.ParseString(req, "script", "")
		threads := mcp.ParseString(req, "threads", "1")
		args := mcp.ParseString(req, "args", "")
		cmd := fmt.Sprintf("col exec %s %s --threads %s", serverName, script, threads)
		if args != "" {
			cmd += " " + args
		}
		return mcp.NewToolResultText(s.run(cmd)), nil
	})

	mcpSrv.AddTool(mcp.NewTool("crack",
		mcp.WithDescription("Crack servers to gain admin rights via Col; omit server to crack all crackable servers"),
		mcp.WithString("server", mcp.Description("Server hostname to crack (omit to crack all)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		serverName := mcp.ParseString(req, "server", "")
		cmd := "col crack"
		if serverName != "" {
			cmd += " " + serverName
		}
		return mcp.NewToolResultText(s.run(cmd)), nil
	})

	mcpSrv.AddTool(mcp.NewTool("killall",
		mcp.WithDescription("Kill all scripts on a server (or every server) via Col; col.js on home is always preserved"),
		mcp.WithString("server", mcp.Description("Server hostname (omit to kill on all servers)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		serverName := mcp.ParseString(req, "server", "")
		cmd := "col killall"
		if serverName != "" {
			cmd += " " + serverName
		}
		return mcp.NewToolResultText(s.run(cmd)), nil
	})

	httpSrv := server.NewStreamableHTTPServer(mcpSrv, server.WithStateLess(true))
	if err := httpSrv.Start(":" + port); err != nil {
		fmt.Printf("MCP server error: %v\n", err)
	}
}
