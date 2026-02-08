package handlers

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func mcpFileTools(server *mcp.Server) {
	server.AddTool(&mcp.Tool{}, mcpListDirectoryTool)
}

func mcpListDirectoryTool(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	return nil, nil
}
