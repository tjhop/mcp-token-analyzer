package mcpclient

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/tjhop/mcp-token-analyzer/pkg/config"
)

// mockRoundTripper captures the request it receives for inspection.
type mockRoundTripper struct {
	received *http.Request
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.received = req
	return &http.Response{StatusCode: http.StatusOK}, nil
}

func TestHeaderRoundTripper(t *testing.T) {
	t.Run("headers_injected", func(t *testing.T) {
		mock := &mockRoundTripper{}
		rt := &headerRoundTripper{
			base: mock,
			headers: map[string]string{
				"Authorization": "Bearer token123",
				"X-Custom":      "custom-value",
			},
		}

		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		_, err = rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("RoundTrip returned error: %v", err)
		}

		if mock.received == nil {
			t.Fatal("base round tripper did not receive a request")
		}
		if got := mock.received.Header.Get("Authorization"); got != "Bearer token123" {
			t.Errorf("expected Authorization header %q, got %q", "Bearer token123", got)
		}
		if got := mock.received.Header.Get("X-Custom"); got != "custom-value" {
			t.Errorf("expected X-Custom header %q, got %q", "custom-value", got)
		}
	})

	t.Run("original_request_not_mutated", func(t *testing.T) {
		mock := &mockRoundTripper{}
		rt := &headerRoundTripper{
			base: mock,
			headers: map[string]string{
				"X-Injected": "injected-value",
			},
		}

		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		_, err = rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("RoundTrip returned error: %v", err)
		}

		// The original request should not have the injected header.
		if got := req.Header.Get("X-Injected"); got != "" {
			t.Errorf("original request was mutated: X-Injected = %q, want empty", got)
		}

		// The cloned request sent to base should be a different pointer.
		if mock.received == req {
			t.Error("base received the original request pointer, expected a clone")
		}
	})

	t.Run("nil_headers_map", func(t *testing.T) {
		mock := &mockRoundTripper{}
		rt := &headerRoundTripper{
			base:    mock,
			headers: nil,
		}

		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		_, err = rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("RoundTrip with nil headers returned error: %v", err)
		}
		if mock.received == nil {
			t.Fatal("base round tripper did not receive a request")
		}
	})

	t.Run("empty_headers_map", func(t *testing.T) {
		mock := &mockRoundTripper{}
		rt := &headerRoundTripper{
			base:    mock,
			headers: map[string]string{},
		}

		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		_, err = rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("RoundTrip with empty headers returned error: %v", err)
		}
		if mock.received == nil {
			t.Fatal("base round tripper did not receive a request")
		}
	})
}

func TestNewStdioClient_EmptyCommand(t *testing.T) {
	_, err := NewStdioClient(context.Background(), "", nil, nil)
	if err == nil {
		t.Fatal("expected error for empty command")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("expected error containing 'cannot be empty', got: %v", err)
	}
}

func TestNewStdioClient_InvalidCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultHTTPTimeout)
	defer cancel()

	_, err := NewStdioClient(ctx, "/nonexistent/binary/path", nil, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent command")
	}
	if !strings.Contains(err.Error(), "failed to connect") {
		t.Errorf("expected error containing 'failed to connect', got: %v", err)
	}
}

func TestNewHTTPClient_InvalidEndpoint(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultHTTPTimeout)
	defer cancel()

	_, err := NewHTTPClient(ctx, "http://127.0.0.1:1/nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for invalid endpoint")
	}
	if !strings.Contains(err.Error(), "failed to connect to MCP server") {
		t.Errorf("expected error containing 'failed to connect to MCP server', got: %v", err)
	}
}

func TestNewClientFromConfig_NilConfig(t *testing.T) {
	_, err := NewClientFromConfig(context.Background(), nil, "")
	if err == nil {
		t.Fatal("expected error for nil config")
	}
	if !strings.Contains(err.Error(), "server configuration is nil") {
		t.Errorf("expected error containing 'server configuration is nil', got: %v", err)
	}
}

func TestNewClientFromConfig_UnsupportedTransport(t *testing.T) {
	srv := &config.ServerConfig{
		Name: "test",
		Type: config.Transport("websocket"),
	}

	_, err := NewClientFromConfig(context.Background(), srv, "")
	if err == nil {
		t.Fatal("expected error for unsupported transport")
	}
	if !strings.Contains(err.Error(), "unsupported transport type") {
		t.Errorf("expected error containing 'unsupported transport type', got: %v", err)
	}
}

func TestNewClientFromConfig_StdioRouting(t *testing.T) {
	// A stdio config with an empty command should route through NewStdioClient
	// and return the "cannot be empty" error, proving the routing works.
	srv := &config.ServerConfig{
		Name:    "test-stdio",
		Type:    config.TransportStdio,
		Command: "",
	}

	_, err := NewClientFromConfig(context.Background(), srv, "")
	if err == nil {
		t.Fatal("expected error for stdio with empty command")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("expected error containing 'cannot be empty', got: %v", err)
	}
}

func TestNewClientFromConfig_HTTPRouting(t *testing.T) {
	// An HTTP config that can't connect should route through NewHTTPClient
	// and the error should mention the endpoint URL.
	srv := &config.ServerConfig{
		Name: "test-http",
		Type: config.TransportHTTP,
		URL:  "http://127.0.0.1:1/nonexistent",
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultHTTPTimeout)
	defer cancel()

	_, err := NewClientFromConfig(ctx, srv, "")
	if err == nil {
		t.Fatal("expected error for HTTP connection failure")
	}
	if !strings.Contains(err.Error(), "127.0.0.1:1/nonexistent") {
		t.Errorf("expected error mentioning the endpoint URL, got: %v", err)
	}
}

func TestNewClientFromConfig_EnvResolutionError(t *testing.T) {
	configDir := t.TempDir()

	srv := &config.ServerConfig{
		Name:    "test-env",
		Type:    config.TransportStdio,
		Command: "echo",
		EnvFile: "../../etc/shadow",
	}

	_, err := NewClientFromConfig(context.Background(), srv, configDir)
	if err == nil {
		t.Fatal("expected error for envFile path traversal")
	}
	if !strings.Contains(err.Error(), "failed to resolve environment") {
		t.Errorf("expected error containing 'failed to resolve environment', got: %v", err)
	}
}

func TestNewStdioClient_OptionsName(t *testing.T) {
	t.Run("nil_opts_does_not_panic", func(t *testing.T) {
		// A nonexistent command will fail at Connect, but nil opts
		// should not cause a panic before that point.
		_, err := NewStdioClient(context.Background(), "/nonexistent/binary/path", nil, nil)
		if err == nil {
			t.Fatal("expected error for nonexistent command")
		}
	})

	t.Run("opts_with_name_does_not_affect_error", func(t *testing.T) {
		opts := &ClientOptions{
			Name: "my-server",
			Env:  map[string]string{"FOO": "bar"},
		}

		_, err := NewStdioClient(context.Background(), "/nonexistent/binary/path", nil, opts)
		if err == nil {
			t.Fatal("expected error for nonexistent command")
		}
		if !strings.Contains(err.Error(), "failed to connect") {
			t.Errorf("expected error containing 'failed to connect', got: %v", err)
		}
	})
}
