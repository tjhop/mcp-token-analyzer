// Package mcpclient provides MCP client implementations for connecting to
// MCP servers via stdio (command execution) and HTTP transports.
package mcpclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/tjhop/mcp-token-analyzer/internal/version"
	"github.com/tjhop/mcp-token-analyzer/pkg/config"
)

// Default timeouts for HTTP connections.
const (
	defaultHTTPTimeout = 30 * time.Second
)

// ClientOptions holds optional configuration for creating MCP clients.
type ClientOptions struct {
	Name    string            // Server identifier for multi-server output
	Env     map[string]string // Environment variables for stdio processes
	Headers map[string]string // HTTP headers for HTTP transport
}

// Client wraps an MCP client session and provides a unified interface for MCP operations.
type Client struct {
	*mcp.ClientSession
	Name string // Server identifier for multi-server output
}

func newMCPClient() *mcp.Client {
	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "mcp-token-analyzer",
		Version: version.Version,
	}, nil)

	return mcpClient
}

// NewStdioClient creates an MCP client that connects via stdio by executing the given command.
// The command and args are provided separately. Pass nil for opts if no options are needed.
func NewStdioClient(ctx context.Context, command string, args []string, opts *ClientOptions) (*Client, error) {
	if command == "" {
		return nil, errors.New("MCP command for stdio transport cannot be empty")
	}

	cmd := exec.Command(command, args...)

	// Set environment variables if provided
	if opts != nil && len(opts.Env) > 0 {
		// Start with current environment and add/override with opts.Env
		cmd.Env = os.Environ()
		for k, v := range opts.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	transport := &mcp.CommandTransport{
		Command: cmd,
	}

	mcpClient := newMCPClient()
	session, err := mcpClient.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect MCP client and establish session: %w", err)
	}

	client := &Client{ClientSession: session}
	if opts != nil {
		client.Name = opts.Name
	}

	return client, nil
}

// headerRoundTripper wraps an http.RoundTripper to add custom headers to all requests.
type headerRoundTripper struct {
	base    http.RoundTripper
	headers map[string]string
}

func (h *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid mutating the original
	reqCopy := req.Clone(req.Context())
	for k, v := range h.headers {
		reqCopy.Header.Set(k, v)
	}
	return h.base.RoundTrip(reqCopy)
}

// NewHTTPClient creates an MCP client that connects via HTTP to the given URL.
// Pass nil for opts if no options are needed.
func NewHTTPClient(ctx context.Context, url string, opts *ClientOptions) (*Client, error) {
	var headers map[string]string
	if opts != nil {
		headers = opts.Headers
	}

	httpClient := &http.Client{
		Transport: &headerRoundTripper{
			base:    http.DefaultTransport,
			headers: headers,
		},
		Timeout: defaultHTTPTimeout,
	}

	transport := &mcp.StreamableClientTransport{
		Endpoint:   url,
		HTTPClient: httpClient,
	}

	mcpClient := newMCPClient()
	session, err := mcpClient.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server at %s: %w", url, err)
	}

	client := &Client{ClientSession: session}
	if opts != nil {
		client.Name = opts.Name
	}

	return client, nil
}

// NewClientFromConfig creates an MCP client from a server configuration.
// The configDir is used to resolve relative paths in the configuration (e.g., envFile).
func NewClientFromConfig(ctx context.Context, srv *config.ServerConfig, configDir string) (*Client, error) {
	if srv == nil {
		return nil, errors.New("server configuration is nil")
	}

	// Resolve environment variables
	env, err := config.MergeServerEnv(srv, configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve environment: %w", err)
	}

	opts := &ClientOptions{
		Name:    srv.Name,
		Env:     env,
		Headers: srv.Headers,
	}

	switch srv.Type {
	case config.TransportStdio:
		return NewStdioClient(ctx, srv.Command, srv.Args, opts)
	case config.TransportHTTP:
		return NewHTTPClient(ctx, srv.URL, opts)
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", srv.Type)
	}
}

// Close terminates the MCP client session.
func (c *Client) Close() error {
	return c.ClientSession.Close()
}
