package handlers

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) mcpFileTools(server *mcp.Server, userId string) {
	server.AddTool(&mcp.Tool{
		Name:        "ListDirectory",
		Description: "Tool to list contents of current directory",
	}, s.mcpListDirectoryTool(userId))
}

type ListDirectoryReq struct{}

func (s *Server) mcpListDirectoryTool(userId string) func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userId := userId
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		x := userId
	}
}
