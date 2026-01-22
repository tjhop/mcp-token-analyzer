package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"syscall"

	"github.com/alecthomas/kingpin/v2"
	"github.com/aquasecurity/table"
	"github.com/tjhop/mcp-token-analyzer/pkg/analyzer"
	"github.com/tjhop/mcp-token-analyzer/pkg/mcpclient"
)

var (
	supportedMCPTransports = []string{"stdio", "http"}

	flagMCPTransport   = kingpin.Flag("mcp.transport", "Transport to use (stdio, http)").Short('t').Default("stdio").Enum(supportedMCPTransports...)
	flagMCPCommand     = kingpin.Flag("mcp.command", "Command to run (for stdio transport)").Short('c').String()
	flagMCPURL         = kingpin.Flag("mcp.url", "URL to connect to (for http transport)").Short('u').String()
	flagTokenizerModel = kingpin.Flag("tokenizer.model", "Tokenizer model to use (e.g. gpt-4, gpt-3.5-turbo)").Short('m').Default("gpt-4").String()
	// TODO (@tjhop): add `--tokenizer.list` flag to list available tokenizers/models and exit.
)

func main() {
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		// TODO (@tjhop): improve error output
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		stop()
		os.Exit(1) //nolint:gocritic
	}
}

func run(ctx context.Context) error {
	var (
		client *mcpclient.Client
		err    error
	)

	switch *flagMCPTransport {
	case "stdio":
		if *flagMCPCommand == "" {
			return errors.New("--mcp.command is required for stdio transport")
		}
		client, err = mcpclient.NewStdioClient(ctx, *flagMCPCommand)
	case "http":
		if *flagMCPURL == "" {
			return errors.New("--mcp.url is required for http transport")
		}
		client, err = mcpclient.NewHTTPClient(ctx, *flagMCPURL)
	}

	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}
	defer client.Close()

	counter, err := analyzer.NewTokenCounter(*flagTokenizerModel)
	if err != nil {
		return fmt.Errorf("error creating counter: %w", err)
	}

	initResp := client.InitializeResult()
	instrx := initResp.Instructions

	initName := "<unset>"
	if initResp.ServerInfo != nil {
		initName = initResp.ServerInfo.Name
	}

	fmt.Println("MCP Server Instructions Analysis")
	instrxTable := table.New(os.Stdout)
	instrxTable.SetHeaders("MCP Server Name", "Instructions tokens")
	instrxTable.AddRow(initName, strconv.Itoa(len(counter.Encode(instrx, nil, nil))))
	instrxTable.Render()

	toolsResp, err := client.ListTools(ctx, nil)
	if err != nil {
		return fmt.Errorf("error listing tools: %w", err)
	}
	tools := toolsResp.Tools

	toolStats := make([]analyzer.ToolTokens, 0, len(tools))
	totalStats := analyzer.ToolTokens{Name: "TOTAL"}

	for _, tool := range tools {
		stats, err := counter.AnalyzeTool(tool)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error analyzing tool %s: %v\n", tool.Name, err)
			continue
		}
		toolStats = append(toolStats, stats)
		totalStats.NameTokens += stats.NameTokens
		totalStats.DescTokens += stats.DescTokens
		totalStats.SchemaTokens += stats.SchemaTokens
		totalStats.TotalTokens += stats.TotalTokens
	}

	sort.Slice(toolStats, func(i, j int) bool {
		return toolStats[i].TotalTokens > toolStats[j].TotalTokens
	})

	fmt.Println("MCP Tool Analysis")
	toolTable := table.New(os.Stdout)
	toolTable.SetHeaders("Tool Name", "Name Tokens", "Desc Tokens", "Schema Tokens", "Total Tokens")
	for _, stats := range toolStats {
		toolTable.AddRow(
			stats.Name,
			strconv.Itoa(stats.NameTokens),
			strconv.Itoa(stats.DescTokens),
			strconv.Itoa(stats.SchemaTokens),
			strconv.Itoa(stats.TotalTokens),
		)
	}
	toolTable.AddFooters(
		totalStats.Name,
		strconv.Itoa(totalStats.NameTokens),
		strconv.Itoa(totalStats.DescTokens),
		strconv.Itoa(totalStats.SchemaTokens),
		strconv.Itoa(totalStats.TotalTokens),
	)
	toolTable.Render()

	return nil
}
