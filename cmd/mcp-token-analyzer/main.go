package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"

	"github.com/alecthomas/kingpin/v2"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"

	"github.com/tjhop/mcp-token-analyzer/pkg/analyzer"
	"github.com/tjhop/mcp-token-analyzer/pkg/config"
	"github.com/tjhop/mcp-token-analyzer/pkg/mcpclient"
)

const (
	tableLabelTotal          = "TOTAL"
	maxConcurrentConnections = 10
	unknownServerName        = "<unknown>"
)

var (
	supportedMCPTransports = []string{string(config.TransportStdio), string(config.TransportHTTP), string(config.TransportStreamableHTTP)}

	// Flags for ad-hoc connections to individual MCP servers.
	flagMCPTransport   = kingpin.Flag("mcp.transport", "Transport to use (stdio, http, streamable-http)").Short('t').Default("stdio").Enum(supportedMCPTransports...)
	flagMCPCommand     = kingpin.Flag("mcp.command", "Command to run (for stdio transport)").Short('c').String()
	flagMCPURL         = kingpin.Flag("mcp.url", "URL to connect to (for http transport)").Short('u').String()
	flagTokenizerModel = kingpin.Flag("tokenizer.model", "Tokenizer model to use (e.g. gpt-4, gpt-3.5-turbo)").Short('m').Default("gpt-4").String()
	// TODO (@tjhop): add `--tokenizer.list` flag to list available tokenizers/models and exit.

	// Flags for working with mcp.json config files.
	// Allows for connecting to multiple MCP servers at once.
	flagConfigFile   = kingpin.Flag("config", "Path to mcp.json config file").Short('f').String()
	flagServer       = kingpin.Flag("server", "Analyze only this named server from config").Short('s').String()
	flagDetail       = kingpin.Flag("detail", "Show detailed per-server tables").Bool()
	flagContextLimit = kingpin.Flag("limit", "Context window limit for percentage calculation").Int()
)

// ServerResult holds the analysis results for a single MCP server.
type ServerResult struct {
	Name                string
	Error               error
	InstructionTokens   int
	TotalToolTokens     analyzer.ToolTokens
	TotalPromptTokens   analyzer.PromptTokens
	TotalResourceTokens analyzer.ResourceTokens

	// Per-component stats for --detail mode
	ToolStats     []analyzer.ToolTokens
	PromptStats   []analyzer.PromptTokens
	ResourceStats []analyzer.ResourceTokens
}

// TotalTokens returns the grand total of all tokens for this server.
func (r *ServerResult) TotalTokens() int {
	return r.InstructionTokens + r.TotalToolTokens.TotalTokens + r.TotalPromptTokens.TotalTokens + r.TotalResourceTokens.TotalTokens
}

// logAnalysisError prints an error message for a failed component analysis.
func logAnalysisError(componentType, name string, err error) {
	fmt.Fprintf(os.Stderr, "Error analyzing %s %s: %v\n", componentType, name, err)
}

// resolveServerName determines the display name for a server, preferring
// the configured name over the server-reported name.
func resolveServerName(configuredName string, serverInfo *mcp.Implementation) string {
	switch {
	case configuredName != "":
		return configuredName
	case serverInfo != nil:
		return serverInfo.Name
	default:
		return unknownServerName
	}
}

func main() {
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error running mcp-token-analyzer: %v\n", err)
		stop()
		os.Exit(1) //nolint:gocritic
	}
}

func run(ctx context.Context) error {
	counter, err := analyzer.NewTokenCounter(*flagTokenizerModel)
	if err != nil {
		return fmt.Errorf("failed to initialize token counter: %w", err)
	}

	cfg, configDir, err := loadOrBuildConfig()
	if err != nil {
		return err
	}

	// Unified processing pipeline for both ad-hoc and file-based configs
	cfg.InferDefaults()
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	for _, warning := range cfg.Warnings() {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", warning)
	}

	return runAnalysis(ctx, cfg, configDir, counter)
}

// loadOrBuildConfig returns a Config from either a file or CLI flags.
// It also returns the config directory for resolving relative paths (empty for ad-hoc mode).
func loadOrBuildConfig() (*config.Config, string, error) {
	if *flagConfigFile != "" {
		cfg, err := config.LoadConfig(*flagConfigFile)
		if err != nil {
			return nil, "", fmt.Errorf("failed to load config: %w", err)
		}
		return cfg, filepath.Dir(*flagConfigFile), nil
	}

	// Build in-memory config from CLI flags
	cfg, err := buildConfigFromFlags()
	if err != nil {
		return nil, "", err
	}
	return cfg, "", nil // Empty configDir - relative paths resolve from CWD
}

// buildConfigFromFlags creates a Config from CLI flags.
// The resulting Config goes through the same InferDefaults/Validate pipeline as file-based configs.
func buildConfigFromFlags() (*config.Config, error) {
	srv := &config.ServerConfig{}

	switch *flagMCPTransport {
	case "stdio":
		if *flagMCPCommand == "" {
			return nil, errors.New("--mcp.command is required for stdio transport")
		}
		// Parse command string into command + args
		parts := strings.Fields(*flagMCPCommand)
		srv.Command = parts[0]
		if len(parts) > 1 {
			srv.Args = parts[1:]
		}
	case "http", "streamable-http":
		if *flagMCPURL == "" {
			return nil, errors.New("--mcp.url is required for http transport")
		}
		srv.URL = *flagMCPURL
	}

	// Return as Config - Name is set from the map key, Type is inferred by InferDefaults()
	return &config.Config{
		MCPServers: map[string]*config.ServerConfig{
			"": srv,
		},
	}, nil
}

// runAnalysis performs server analysis on the given config.
// This is the unified analysis path for both ad-hoc and file-based configs.
func runAnalysis(ctx context.Context, cfg *config.Config, configDir string, counter *analyzer.TokenCounter) error {
	servers := cfg.GetServers()

	// Filter to single server if specified
	if *flagServer != "" {
		srv, ok := servers[*flagServer]
		if !ok {
			return fmt.Errorf("server %q not found in config", *flagServer)
		}
		servers = map[string]*config.ServerConfig{*flagServer: srv}
	}

	if len(servers) == 0 {
		return errors.New("no servers to analyze")
	}

	// If single server (either filtered or only one in config), use single-server output
	if len(servers) == 1 {
		for name, srv := range servers {
			result := analyzeServer(ctx, name, srv, configDir, counter)
			if result.Error != nil {
				return fmt.Errorf("server %q: %w", name, result.Error)
			}
			renderSingleServerOutput(result)
			return nil
		}
	}

	// Multi-server mode: analyze all servers in parallel
	results := connectAndAnalyzeAll(ctx, servers, configDir, counter)

	// Render group summary
	renderGroupSummary(results)

	// Render detailed tables if requested
	if *flagDetail {
		renderDetailedTables(results)
	}

	// Report failure if any servers had errors, so the CLI exits with non-zero status.
	var failCount int
	for _, r := range results {
		if r.Error != nil {
			failCount++
		}
	}
	if failCount > 0 {
		return fmt.Errorf("%d of %d servers failed analysis", failCount, len(results))
	}

	return nil
}

// analyzeServer connects to a server and analyzes it.
func analyzeServer(ctx context.Context, name string, srv *config.ServerConfig, configDir string, counter *analyzer.TokenCounter) *ServerResult {
	client, err := mcpclient.NewClientFromConfig(ctx, srv, configDir)
	if err != nil {
		return &ServerResult{Name: name, Error: err}
	}
	defer client.Close()

	result := analyzeClient(ctx, client, counter)
	result.Name = name
	return result
}

// analyzeClient performs the analysis on an already-connected client.
func analyzeClient(ctx context.Context, client *mcpclient.Client, counter *analyzer.TokenCounter) *ServerResult {
	result := &ServerResult{}

	// Get server name and instructions from the initialize response.
	// The SDK should always return a non-nil result after successful connection,
	// but we check defensively to avoid panics.
	initResp := client.InitializeResult()
	if initResp == nil {
		result.Name = resolveServerName(client.Name, nil)
		result.Error = errors.New("MCP session not initialized")
		return result
	}

	result.Name = resolveServerName(client.Name, initResp.ServerInfo)

	// Instructions
	result.InstructionTokens = counter.CountTokens(initResp.Instructions)

	// Analyze each component - all are optional since not all servers support all capabilities
	result.ToolStats, result.TotalToolTokens = analyzeTools(ctx, client, counter)
	result.PromptStats, result.TotalPromptTokens = analyzePrompts(ctx, client, counter)
	result.ResourceStats, result.TotalResourceTokens = analyzeResources(ctx, client, counter)

	return result
}

// analyzeTools lists and analyzes all tools from the server.
// Returns the per-tool stats and the accumulated totals.
// Not all servers support tools, so ListTools errors are logged as warnings.
func analyzeTools(ctx context.Context, client *mcpclient.Client, counter *analyzer.TokenCounter) ([]analyzer.ToolTokens, analyzer.ToolTokens) {
	total := analyzer.ToolTokens{Name: tableLabelTotal}

	toolsResp, err := client.ListTools(ctx, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to list tools for %s: %v\n", client.Name, err)
		return nil, total
	}

	var stats []analyzer.ToolTokens
	for _, tool := range toolsResp.Tools {
		s, err := counter.AnalyzeTool(tool)
		if err != nil {
			logAnalysisError("tool", tool.Name, err)
			continue
		}
		stats = append(stats, s)
		total.Add(s)
	}

	return stats, total
}

// analyzePrompts lists and analyzes all prompts from the server.
// Not all servers support prompts, so ListPrompts errors are logged as warnings.
func analyzePrompts(ctx context.Context, client *mcpclient.Client, counter *analyzer.TokenCounter) ([]analyzer.PromptTokens, analyzer.PromptTokens) {
	total := analyzer.PromptTokens{Name: tableLabelTotal}

	promptsResp, err := client.ListPrompts(ctx, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to list prompts for %s: %v\n", client.Name, err)
		return nil, total
	}

	var stats []analyzer.PromptTokens
	for _, prompt := range promptsResp.Prompts {
		s, err := counter.AnalyzePrompt(prompt)
		if err != nil {
			logAnalysisError("prompt", prompt.Name, err)
			continue
		}
		stats = append(stats, s)
		total.Add(s)
	}

	return stats, total
}

// analyzeResources lists and analyzes all resources and resource templates from the server.
// Not all servers support resources, so ListResources errors are logged as warnings.
func analyzeResources(ctx context.Context, client *mcpclient.Client, counter *analyzer.TokenCounter) ([]analyzer.ResourceTokens, analyzer.ResourceTokens) {
	total := analyzer.ResourceTokens{Name: tableLabelTotal}

	resourcesResp, err := client.ListResources(ctx, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to list resources for %s: %v\n", client.Name, err)
		return nil, total
	}

	var stats []analyzer.ResourceTokens
	for _, resource := range resourcesResp.Resources {
		s, err := counter.AnalyzeResource(resource)
		if err != nil {
			logAnalysisError("resource", resource.Name, err)
			continue
		}
		stats = append(stats, s)
		total.Add(s)
	}

	// Also list templates
	templatesResp, err := client.ListResourceTemplates(ctx, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to list resource templates for %s: %v\n", client.Name, err)
		return stats, total
	}

	for _, template := range templatesResp.ResourceTemplates {
		s, err := counter.AnalyzeResourceTemplate(template)
		if err != nil {
			logAnalysisError("resource template", template.Name, err)
			continue
		}
		stats = append(stats, s)
		total.Add(s)
	}

	return stats, total
}

// connectAndAnalyzeAll connects to all servers in parallel and returns results.
// The servers map and its ServerConfig values are treated as read-only; concurrent
// goroutines only read configuration data, never modify it.
func connectAndAnalyzeAll(ctx context.Context, servers map[string]*config.ServerConfig, configDir string, counter *analyzer.TokenCounter) []*ServerResult {
	var (
		results []*ServerResult
		mu      sync.Mutex
	)

	var g errgroup.Group
	g.SetLimit(maxConcurrentConnections)

	for name, srv := range servers {
		g.Go(func() error {
			result := analyzeServer(ctx, name, srv, configDir, counter)
			mu.Lock()
			defer mu.Unlock()
			results = append(results, result)
			// Don't return error - we want to continue analyzing other servers
			return nil
		})
	}

	// Errors are captured in individual results, not returned via errgroup.
	// Each goroutine returns nil to allow all servers to be processed.
	_ = g.Wait()

	// Sort results by name for consistent output
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	return results
}
