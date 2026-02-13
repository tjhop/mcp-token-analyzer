package main

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/tjhop/mcp-token-analyzer/pkg/analyzer"
)

func TestServerResult_TotalTokens(t *testing.T) {
	tests := []struct {
		name   string
		result ServerResult
		want   int
	}{
		{
			name:   "empty_result",
			result: ServerResult{},
			want:   0,
		},
		{
			name: "only_instructions",
			result: ServerResult{
				InstructionTokens: 100,
			},
			want: 100,
		},
		{
			name: "only_tools",
			result: ServerResult{
				TotalToolTokens: analyzer.ToolTokens{TotalTokens: 200},
			},
			want: 200,
		},
		{
			name: "only_prompts",
			result: ServerResult{
				TotalPromptTokens: analyzer.PromptTokens{TotalTokens: 150},
			},
			want: 150,
		},
		{
			name: "only_resources",
			result: ServerResult{
				TotalResourceTokens: analyzer.ResourceTokens{TotalTokens: 75},
			},
			want: 75,
		},
		{
			name: "all_components",
			result: ServerResult{
				InstructionTokens:   100,
				TotalToolTokens:     analyzer.ToolTokens{TotalTokens: 200},
				TotalPromptTokens:   analyzer.PromptTokens{TotalTokens: 150},
				TotalResourceTokens: analyzer.ResourceTokens{TotalTokens: 75},
			},
			want: 525,
		},
		{
			name: "realistic_values",
			result: ServerResult{
				Name:                "test-server",
				InstructionTokens:   50,
				TotalToolTokens:     analyzer.ToolTokens{Name: "TOTAL", TotalTokens: 1500},
				TotalPromptTokens:   analyzer.PromptTokens{Name: "TOTAL", TotalTokens: 300},
				TotalResourceTokens: analyzer.ResourceTokens{Name: "TOTAL", TotalTokens: 250},
			},
			want: 2100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.TotalTokens(); got != tt.want {
				t.Errorf("ServerResult.TotalTokens() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestResolveServerName(t *testing.T) {
	tests := []struct {
		name           string
		configuredName string
		serverInfo     *mcp.Implementation
		want           string
	}{
		{
			name:           "configured_name_only",
			configuredName: "my-server",
			serverInfo:     nil,
			want:           "my-server",
		},
		{
			name:           "configured_name_takes_precedence",
			configuredName: "configured",
			serverInfo:     &mcp.Implementation{Name: "server-reported"},
			want:           "configured",
		},
		{
			name:           "serverinfo_name_when_no_configured",
			configuredName: "",
			serverInfo:     &mcp.Implementation{Name: "server-reported"},
			want:           "server-reported",
		},
		{
			name:           "serverinfo_with_version",
			configuredName: "",
			serverInfo:     &mcp.Implementation{Name: "my-mcp-server", Version: "1.0.0"},
			want:           "my-mcp-server",
		},
		{
			name:           "both_empty_returns_unknown",
			configuredName: "",
			serverInfo:     nil,
			want:           unknownServerName,
		},
		{
			name:           "serverinfo_with_empty_name_falls_through",
			configuredName: "",
			serverInfo:     &mcp.Implementation{Name: "", Version: "1.0.0"},
			want:           unknownServerName,
		},
		{
			name:           "configured_empty_string_uses_serverinfo",
			configuredName: "",
			serverInfo:     &mcp.Implementation{Name: "fallback-name"},
			want:           "fallback-name",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveServerName(tt.configuredName, tt.serverInfo); got != tt.want {
				t.Errorf("resolveServerName(%q, %v) = %q, want %q", tt.configuredName, tt.serverInfo, got, tt.want)
			}
		})
	}
}
