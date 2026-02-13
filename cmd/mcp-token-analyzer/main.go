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
	flagContextLimit = kingpin.Flag("limit", "Optional context window limit for percentage calculation").Int()
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
	case serverInfo != nil && serverInfo.Name != "":
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
			return nil, errors.New("--mcp.command is required for stdio transport in single-server mode")
		}
		// Parse command string into command + args
		parts := strings.Fields(*flagMCPCommand)
		srv.Command = parts[0]
		if len(parts) > 1 {
			srv.Args = parts[1:]
		}
	case "http", "streamable-http":
		if *flagMCPURL == "" {
			return nil, errors.New("--mcp.url is required for http transport in single-server mode")
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
	servers := cfg.MergedServers()

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

	results := connectAndAnalyzeAll(ctx, servers, configDir, counter)

	// Single-server results always include detail tables; multi-server
	// results include them only when explicitly requested via --detail.
	if len(results) == 1 || *flagDetail {
		renderDetailTables(results)
	}

	renderSummary(results)

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
// Name resolution is handled here: the configured name (map key) takes
// precedence over the server-reported name from the init response. When
// running ad-hoc (empty map key), the server-reported name is used as fallback.
func analyzeServer(ctx context.Context, name string, srv *config.ServerConfig, configDir string, counter *analyzer.TokenCounter) *ServerResult {
	client, err := mcpclient.NewClientFromConfig(ctx, srv, configDir)
	if err != nil {
		return &ServerResult{Name: resolveServerName(name, nil), Error: err}
	}
	defer client.Close()

	result := analyzeClient(ctx, client, counter)

	// Resolve the final display name from the configured name and
	// whatever the server reported during initialization.
	initResp := client.InitializeResult()
	var serverInfo *mcp.Implementation
	if initResp != nil {
		serverInfo = initResp.ServerInfo
	}
	result.Name = resolveServerName(name, serverInfo)

	return result
}

// analyzeClient performs the analysis on an already-connected client.
func analyzeClient(ctx context.Context, client *mcpclient.Client, counter *analyzer.TokenCounter) *ServerResult {
	result := &ServerResult{}

	initResp := client.InitializeResult()
	if initResp == nil {
		result.Error = errors.New("MCP session not initialized")
		return result
	}

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
// Uses the SDK's paginating iterator to ensure all tools are retrieved.
// Not all servers support tools, so listing errors are logged as warnings.
//
// Note: some variations of mcp.json format have the concept of allowed/denied
// tools. If/when that is standardized, this project may support it. For now,
// we always attempt to load and analyze all tools.
func analyzeTools(ctx context.Context, client *mcpclient.Client, counter *analyzer.TokenCounter) ([]analyzer.ToolTokens, analyzer.ToolTokens) {
	total := analyzer.ToolTokens{Name: tableLabelTotal}

	var stats []analyzer.ToolTokens
	for tool, err := range client.Tools(ctx, nil) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to list tools for %s: %v\n", client.Name, err)
			break
		}
		toolStats, err := counter.AnalyzeTool(tool)
		if err != nil {
			logAnalysisError("tool", tool.Name, err)
			continue
		}
		stats = append(stats, toolStats)
		total.Add(toolStats)
	}

	return stats, total
}

// analyzePrompts lists and analyzes all prompts from the server.
// Uses the SDK's paginating iterator to ensure all prompts are retrieved.
// Not all servers support prompts, so listing errors are logged as warnings.
func analyzePrompts(ctx context.Context, client *mcpclient.Client, counter *analyzer.TokenCounter) ([]analyzer.PromptTokens, analyzer.PromptTokens) {
	total := analyzer.PromptTokens{Name: tableLabelTotal}

	var stats []analyzer.PromptTokens
	for prompt, err := range client.Prompts(ctx, nil) {
		if err != nil {
			// Known issue: servers that don't implement prompts/resources
			// produce noisy "Method not found" warnings here. Ideally
			// these would be debug-level log messages, but that requires
			// migrating to a structured logger (e.g., log/slog). For now,
			// they remain as stderr warnings.
			fmt.Fprintf(os.Stderr, "Warning: failed to list prompts for %s: %v\n", client.Name, err)
			break
		}
		promptStats, err := counter.AnalyzePrompt(prompt)
		if err != nil {
			logAnalysisError("prompt", prompt.Name, err)
			continue
		}
		stats = append(stats, promptStats)
		total.Add(promptStats)
	}

	return stats, total
}

// analyzeResources lists and analyzes all resources and resource templates from the server.
// Uses the SDK's paginating iterators to ensure all items are retrieved.
// Not all servers support resources, so listing errors are logged as warnings.
func analyzeResources(ctx context.Context, client *mcpclient.Client, counter *analyzer.TokenCounter) ([]analyzer.ResourceTokens, analyzer.ResourceTokens) {
	total := analyzer.ResourceTokens{Name: tableLabelTotal}

	var stats []analyzer.ResourceTokens
	for resource, err := range client.Resources(ctx, nil) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to list resources for %s: %v\n", client.Name, err)
			break
		}
		resourceStats, err := counter.AnalyzeResource(resource)
		if err != nil {
			logAnalysisError("resource", resource.Name, err)
			continue
		}
		stats = append(stats, resourceStats)
		total.Add(resourceStats)
	}

	for template, err := range client.ResourceTemplates(ctx, nil) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to list resource templates for %s: %v\n", client.Name, err)
			break
		}
		templateStats, err := counter.AnalyzeResourceTemplate(template)
		if err != nil {
			logAnalysisError("resource template", template.Name, err)
			continue
		}
		stats = append(stats, templateStats)
		total.Add(templateStats)
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

	// Using errgroup.Group over sync.WaitGroup for built in concurrency
	// limiter.
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
