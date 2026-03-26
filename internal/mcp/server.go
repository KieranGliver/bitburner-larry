package mcpserver

import (
	"context"
	"fmt"
	"sync"

	larcmd "github.com/KieranGliver/bitburner-larry/cmd"
	"github.com/KieranGliver/bitburner-larry/internal/communication"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type McpServer struct {
	conn   *communication.BitburnerConn
	onCall func(input, result string)
	mu     sync.RWMutex
}

func New(fn func(input, result string)) *McpServer {
	s := &McpServer{onCall: fn}
	return s
}

func (s *McpServer) SetConn(conn *communication.BitburnerConn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conn = conn
}

func (s *McpServer) run(input string) string {
	s.mu.RLock()
	conn := s.conn
	onCall := s.onCall
	s.mu.RUnlock()

	result := larcmd.ExecuteCommand(input, conn)
	if onCall != nil {
		onCall(input, result)
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

	httpSrv := server.NewStreamableHTTPServer(mcpSrv, server.WithStateLess(true))
	if err := httpSrv.Start(":" + port); err != nil {
		fmt.Printf("MCP server error: %v\n", err)
	}
}
