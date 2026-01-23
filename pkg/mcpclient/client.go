// Package mcpclient provides MCP client implementations for connecting to
// MCP servers via stdio (command execution) and HTTP transports.
package mcpclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/tjhop/mcp-token-analyzer/internal/version"
)

// Client wraps an MCP client session and provides a unified interface for MCP operations.
type Client struct {
	*mcp.ClientSession
}

func newMCPClient() *mcp.Client {
	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "mcp-token-analyzer",
		Version: version.Version,
	}, nil)

	return mcpClient
}

// NewStdioClient creates an MCP client that connects via stdio by executing the given command.
func NewStdioClient(ctx context.Context, command string) (*Client, error) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil, errors.New("MCP command for stdio transport cannot be empty")
	}

	cmd := exec.Command(parts[0], parts[1:]...)

	transport := &mcp.CommandTransport{
		Command: cmd,
	}

	mcpClient := newMCPClient()
	session, err := mcpClient.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect MCP client and establish session: %w", err)
	}

	return &Client{session}, nil
}

// NewHTTPClient creates an MCP client that connects via HTTP to the given URL.
// TODO (@tjhop): allow configuring HTTP client attributes and auth things like basic auth, tls etc?
func NewHTTPClient(ctx context.Context, url string) (*Client, error) {
	transport := &mcp.StreamableClientTransport{
		Endpoint:   url,
		HTTPClient: http.DefaultClient,
	}

	mcpClient := newMCPClient()
	session, err := mcpClient.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect MCP client and establish session: %w", err)
	}

	return &Client{session}, nil
}

// Close terminates the MCP client session.
func (c *Client) Close() error {
	return c.ClientSession.Close()
}
