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

const (
	tableLabelTotal = "TOTAL"
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
		fmt.Fprintf(os.Stderr, "error running mcp-token-analyzer: %v\n", err)
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

	// Tokenize and report system instructions.
	initResp := client.InitializeResult()
	instrx := initResp.Instructions

	initName := "<unset>"
	if initResp.ServerInfo != nil {
		initName = initResp.ServerInfo.Name
	}

	fmt.Printf("\nMCP Server Instructions Analysis\n")
	instrxTable := table.New(os.Stdout)
	instrxTable.SetHeaders("MCP Server Name", "Instructions tokens")
	instrxTokens := len(counter.Encode(instrx, nil, nil))
	instrxTable.AddRow(initName, strconv.Itoa(instrxTokens))
	instrxTable.Render()

	// Tokenize and report tools.
	fmt.Printf("\nMCP Tool Analysis\n")
	totalToolStats := analyzer.ToolTokens{Name: tableLabelTotal}
	toolTable := table.New(os.Stdout)
	toolTable.SetHeaders("Tool Name", "Name Tokens", "Desc Tokens", "Schema Tokens", "Total Tokens")

	toolsResp, err := client.ListTools(ctx, nil)
	if err != nil {
		return fmt.Errorf("error listing tools: %w", err)
	}

	if toolsResp != nil {
		tools := toolsResp.Tools
		toolStats := make([]analyzer.ToolTokens, 0, len(tools))

		for _, tool := range tools {
			stats, err := counter.AnalyzeTool(tool)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error analyzing tool %s: %v\n", tool.Name, err)
				continue
			}
			toolStats = append(toolStats, stats)
			totalToolStats.NameTokens += stats.NameTokens
			totalToolStats.DescTokens += stats.DescTokens
			totalToolStats.SchemaTokens += stats.SchemaTokens
			totalToolStats.TotalTokens += stats.TotalTokens
		}

		// Sort in descending order of total tokens for output.
		sort.Slice(toolStats, func(i, j int) bool {
			return toolStats[i].TotalTokens > toolStats[j].TotalTokens
		})

		for _, stats := range toolStats {
			toolTable.AddRow(
				stats.Name,
				strconv.Itoa(stats.NameTokens),
				strconv.Itoa(stats.DescTokens),
				strconv.Itoa(stats.SchemaTokens),
				strconv.Itoa(stats.TotalTokens),
			)
		}
	}

	toolTable.AddFooters(
		totalToolStats.Name,
		strconv.Itoa(totalToolStats.NameTokens),
		strconv.Itoa(totalToolStats.DescTokens),
		strconv.Itoa(totalToolStats.SchemaTokens),
		strconv.Itoa(totalToolStats.TotalTokens),
	)
	toolTable.Render()

	// Tokenize and report prompts.
	fmt.Printf("\nMCP Prompt Analysis\n")
	totalPromptStats := analyzer.PromptTokens{Name: tableLabelTotal}
	promptTable := table.New(os.Stdout)
	promptTable.SetHeaders("Prompt Name", "Name Tokens", "Desc Tokens", "Args Tokens", "Total Tokens")

	promptsResp, err := client.ListPrompts(ctx, nil)
	if err != nil {
		return fmt.Errorf("error listing prompts: %w", err)
	}

	if promptsResp != nil {
		prompts := promptsResp.Prompts
		promptStats := make([]analyzer.PromptTokens, 0, len(prompts))

		for _, prompt := range prompts {
			stats, err := counter.AnalyzePrompt(prompt)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error analyzing prompt %s: %v\n", prompt.Name, err)
				continue
			}
			promptStats = append(promptStats, stats)
			totalPromptStats.NameTokens += stats.NameTokens
			totalPromptStats.DescTokens += stats.DescTokens
			totalPromptStats.ArgsTokens += stats.ArgsTokens
			totalPromptStats.TotalTokens += stats.TotalTokens
		}

		// Sort in descending order of total tokens for output.
		sort.Slice(promptStats, func(i, j int) bool {
			return promptStats[i].TotalTokens > promptStats[j].TotalTokens
		})

		for _, stats := range promptStats {
			promptTable.AddRow(
				stats.Name,
				strconv.Itoa(stats.NameTokens),
				strconv.Itoa(stats.DescTokens),
				strconv.Itoa(stats.ArgsTokens),
				strconv.Itoa(stats.TotalTokens),
			)
		}
	}

	promptTable.AddFooters(
		totalPromptStats.Name,
		strconv.Itoa(totalPromptStats.NameTokens),
		strconv.Itoa(totalPromptStats.DescTokens),
		strconv.Itoa(totalPromptStats.ArgsTokens),
		strconv.Itoa(totalPromptStats.TotalTokens),
	)
	promptTable.Render()

	// Tokenize and report resources and resource templates.
	fmt.Printf("\nMCP Resource Analysis\n")
	resourceStats := make([]analyzer.ResourceTokens, 0)
	totalResourceStats := analyzer.ResourceTokens{Name: tableLabelTotal}
	resourceTable := table.New(os.Stdout)
	resourceTable.SetHeaders("Resource Name", "Name Tokens", "URI Tokens", "Desc Tokens", "Total Tokens")

	resourcesResp, err := client.ListResources(ctx, nil)
	if err != nil {
		return fmt.Errorf("error listing resources: %w", err)
	}

	if resourcesResp != nil {
		resources := resourcesResp.Resources

		for _, resource := range resources {
			stats, err := counter.AnalyzeResource(resource)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error analyzing resource %s: %v\n", resource.Name, err)
				continue
			}
			resourceStats = append(resourceStats, stats)
			totalResourceStats.NameTokens += stats.NameTokens
			totalResourceStats.URITokens += stats.URITokens
			totalResourceStats.DescTokens += stats.DescTokens
			totalResourceStats.TotalTokens += stats.TotalTokens
		}
	}

	templatesResp, err := client.ListResourceTemplates(ctx, nil)
	if err != nil {
		return fmt.Errorf("error listing resource templates: %w", err)
	}

	if templatesResp != nil {
		for _, template := range templatesResp.ResourceTemplates {
			stats, err := counter.AnalyzeResourceTemplate(template)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error analyzing resource template %s: %v\n", template.Name, err)
				continue
			}
			resourceStats = append(resourceStats, stats)
			totalResourceStats.NameTokens += stats.NameTokens
			totalResourceStats.URITokens += stats.URITokens
			totalResourceStats.DescTokens += stats.DescTokens
			totalResourceStats.TotalTokens += stats.TotalTokens
		}
	}

	sort.Slice(resourceStats, func(i, j int) bool {
		return resourceStats[i].TotalTokens > resourceStats[j].TotalTokens
	})

	for _, stats := range resourceStats {
		resourceTable.AddRow(
			stats.Name,
			strconv.Itoa(stats.NameTokens),
			strconv.Itoa(stats.URITokens),
			strconv.Itoa(stats.DescTokens),
			strconv.Itoa(stats.TotalTokens),
		)
	}

	resourceTable.AddFooters(
		totalResourceStats.Name,
		strconv.Itoa(totalResourceStats.NameTokens),
		strconv.Itoa(totalResourceStats.URITokens),
		strconv.Itoa(totalResourceStats.DescTokens),
		strconv.Itoa(totalResourceStats.TotalTokens),
	)
	resourceTable.Render()

	// Summary breakdown.
	fmt.Printf("\nSummary MCP Static Token Usage\n")
	grandTotal := instrxTokens + totalToolStats.TotalTokens + totalPromptStats.TotalTokens + totalResourceStats.TotalTokens
	summaryTotalTable := table.New(os.Stdout)
	summaryTotalTable.SetHeaders("MCP component", "Tokens")
	summaryTotalTable.AddRow("Instructions", strconv.Itoa(instrxTokens))
	summaryTotalTable.AddRow("Tools", strconv.Itoa(totalToolStats.TotalTokens))
	summaryTotalTable.AddRow("Prompts", strconv.Itoa(totalPromptStats.TotalTokens))
	summaryTotalTable.AddRow("Resources and Templates", strconv.Itoa(totalResourceStats.TotalTokens))
	summaryTotalTable.AddFooters(tableLabelTotal, strconv.Itoa(grandTotal))
	summaryTotalTable.Render()

	return nil
}
