// render.go contains table rendering utilities for displaying token analysis results.

package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/aquasecurity/table"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/tjhop/mcp-token-analyzer/pkg/analyzer"
)

// printer is used for locale-aware number formatting with thousands separators.
var printer = message.NewPrinter(language.English)

// namespaceSeparator is used in multi-server mode to create unique identifiers
// for tools, prompts, and resources across servers. For example, a tool named
// "query" from server "prometheus" becomes "prometheus__query".
const namespaceSeparator = "__"

// renderContextUsage prints context window usage as a percentage if a limit is configured.
func renderContextUsage(grandTotal int) {
	if *flagContextLimit > 0 {
		pct := float64(grandTotal) / float64(*flagContextLimit) * 100
		fmt.Printf("\nContext Usage: %s / %s (%.1f%%)\n",
			printer.Sprintf("%d", grandTotal),
			printer.Sprintf("%d", *flagContextLimit),
			pct,
		)
	}
}

// renderSingleServerOutput renders the original single-server output format.
func renderSingleServerOutput(result *ServerResult) {
	// Instructions table
	fmt.Printf("\nMCP Server Instructions Analysis\n")
	instructionsTable := table.New(os.Stdout)
	instructionsTable.SetHeaders("MCP Server Name", "Instructions tokens")
	instructionsTable.AddRow(result.Name, strconv.Itoa(result.InstructionTokens))
	instructionsTable.Render()

	// Tools table
	if len(result.ToolStats) > 0 {
		sort.Slice(result.ToolStats, func(i, j int) bool {
			return result.ToolStats[i].TotalTokens > result.ToolStats[j].TotalTokens
		})

		fmt.Printf("\nMCP Tool Analysis\n")
		toolTable := table.New(os.Stdout)
		toolTable.SetHeaders("Tool Name", "Name Tokens", "Desc Tokens", "Schema Tokens", "Total Tokens")
		for _, stats := range result.ToolStats {
			toolTable.AddRow(
				stats.Name,
				strconv.Itoa(stats.NameTokens),
				strconv.Itoa(stats.DescTokens),
				strconv.Itoa(stats.SchemaTokens),
				strconv.Itoa(stats.TotalTokens),
			)
		}
		toolTable.AddFooters(
			result.TotalToolTokens.Name,
			strconv.Itoa(result.TotalToolTokens.NameTokens),
			strconv.Itoa(result.TotalToolTokens.DescTokens),
			strconv.Itoa(result.TotalToolTokens.SchemaTokens),
			strconv.Itoa(result.TotalToolTokens.TotalTokens),
		)
		toolTable.Render()
	}

	// Prompts table
	if len(result.PromptStats) > 0 {
		sort.Slice(result.PromptStats, func(i, j int) bool {
			return result.PromptStats[i].TotalTokens > result.PromptStats[j].TotalTokens
		})

		fmt.Printf("\nMCP Prompt Analysis\n")
		promptTable := table.New(os.Stdout)
		promptTable.SetHeaders("Prompt Name", "Name Tokens", "Desc Tokens", "Args Tokens", "Total Tokens")
		for _, stats := range result.PromptStats {
			promptTable.AddRow(
				stats.Name,
				strconv.Itoa(stats.NameTokens),
				strconv.Itoa(stats.DescTokens),
				strconv.Itoa(stats.ArgsTokens),
				strconv.Itoa(stats.TotalTokens),
			)
		}
		promptTable.AddFooters(
			result.TotalPromptTokens.Name,
			strconv.Itoa(result.TotalPromptTokens.NameTokens),
			strconv.Itoa(result.TotalPromptTokens.DescTokens),
			strconv.Itoa(result.TotalPromptTokens.ArgsTokens),
			strconv.Itoa(result.TotalPromptTokens.TotalTokens),
		)
		promptTable.Render()
	}

	// Resources table
	if len(result.ResourceStats) > 0 {
		sort.Slice(result.ResourceStats, func(i, j int) bool {
			return result.ResourceStats[i].TotalTokens > result.ResourceStats[j].TotalTokens
		})

		fmt.Printf("\nMCP Resource Analysis\n")
		resTable := table.New(os.Stdout)
		resTable.SetHeaders("Resource Name", "Name Tokens", "URI Tokens", "Desc Tokens", "Total Tokens")
		for _, stats := range result.ResourceStats {
			resTable.AddRow(
				stats.Name,
				strconv.Itoa(stats.NameTokens),
				strconv.Itoa(stats.URITokens),
				strconv.Itoa(stats.DescTokens),
				strconv.Itoa(stats.TotalTokens),
			)
		}
		resTable.AddFooters(
			result.TotalResourceTokens.Name,
			strconv.Itoa(result.TotalResourceTokens.NameTokens),
			strconv.Itoa(result.TotalResourceTokens.URITokens),
			strconv.Itoa(result.TotalResourceTokens.DescTokens),
			strconv.Itoa(result.TotalResourceTokens.TotalTokens),
		)
		resTable.Render()
	}

	// Summary breakdown
	fmt.Printf("\nSummary MCP Static Token Usage\n")
	grandTotal := result.TotalTokens()
	summaryTotalTable := table.New(os.Stdout)
	summaryTotalTable.SetHeaders("MCP component", "Tokens")
	summaryTotalTable.AddRow("Instructions", strconv.Itoa(result.InstructionTokens))
	summaryTotalTable.AddRow("Tools", strconv.Itoa(result.TotalToolTokens.TotalTokens))
	summaryTotalTable.AddRow("Prompts", strconv.Itoa(result.TotalPromptTokens.TotalTokens))
	summaryTotalTable.AddRow("Resources and Templates", strconv.Itoa(result.TotalResourceTokens.TotalTokens))
	summaryTotalTable.AddFooters(tableLabelTotal, strconv.Itoa(grandTotal))
	summaryTotalTable.Render()

	renderContextUsage(grandTotal)
}

// renderGroupSummary renders the group summary table for multi-server mode.
func renderGroupSummary(results []*ServerResult) {
	fmt.Println("Group Token Analysis Summary")
	groupTable := table.New(os.Stdout)
	groupTable.SetHeaders("MCP Server", "Instructions", "Tools", "Prompts", "Resources", "Total Tokens")

	var totalInstructionTokens, totalTools, totalPrompts, totalResources, grandTotal int

	for _, r := range results {
		if r.Error != nil {
			groupTable.AddRow(
				r.Name,
				"ERROR",
				"",
				"",
				"",
				r.Error.Error(),
			)
			continue
		}

		total := r.TotalTokens()
		groupTable.AddRow(
			r.Name,
			printer.Sprintf("%d", r.InstructionTokens),
			printer.Sprintf("%d", r.TotalToolTokens.TotalTokens),
			printer.Sprintf("%d", r.TotalPromptTokens.TotalTokens),
			printer.Sprintf("%d", r.TotalResourceTokens.TotalTokens),
			printer.Sprintf("%d", total),
		)

		totalInstructionTokens += r.InstructionTokens
		totalTools += r.TotalToolTokens.TotalTokens
		totalPrompts += r.TotalPromptTokens.TotalTokens
		totalResources += r.TotalResourceTokens.TotalTokens
		grandTotal += total
	}

	groupTable.AddFooters(
		"GROUP TOTAL",
		printer.Sprintf("%d", totalInstructionTokens),
		printer.Sprintf("%d", totalTools),
		printer.Sprintf("%d", totalPrompts),
		printer.Sprintf("%d", totalResources),
		printer.Sprintf("%d", grandTotal),
	)
	groupTable.Render()

	renderContextUsage(grandTotal)
}

// namespacedItem holds a stats value associated with a server for cross-server detail tables.
type namespacedItem[T any] struct {
	Server string
	Name   string
	Stats  T
}

// renderNamespacedTable collects items from results, sorts by total tokens, and renders a table.
func renderNamespacedTable[T any](
	results []*ServerResult,
	title string,
	headers []string,
	extract func(r *ServerResult) []T,
	itemName func(T) string,
	totalTokens func(T) int,
	rowValues func(T) []string,
) {
	var items []namespacedItem[T]
	for _, r := range results {
		if r.Error != nil {
			continue
		}
		for _, item := range extract(r) {
			items = append(items, namespacedItem[T]{
				Server: r.Name,
				Name:   r.Name + namespaceSeparator + itemName(item),
				Stats:  item,
			})
		}
	}

	if len(items) == 0 {
		return
	}

	sort.Slice(items, func(i, j int) bool {
		return totalTokens(items[i].Stats) > totalTokens(items[j].Stats)
	})

	fmt.Println("\n" + title)
	t := table.New(os.Stdout)
	t.SetHeaders(headers...)
	for _, item := range items {
		row := append([]string{item.Server, item.Name}, rowValues(item.Stats)...)
		t.AddRow(row...)
	}
	t.Render()
}

// renderDetailedTables renders the unified component tables for --detail mode.
func renderDetailedTables(results []*ServerResult) {
	renderNamespacedTable(
		results,
		"Unified Tool Analysis (sorted by total tokens)",
		[]string{"Server", "Tool (Namespaced)", "Name", "Desc", "Schema", "Total"},
		func(r *ServerResult) []analyzer.ToolTokens { return r.ToolStats },
		func(t analyzer.ToolTokens) string { return t.Name },
		func(t analyzer.ToolTokens) int { return t.TotalTokens },
		func(t analyzer.ToolTokens) []string {
			return []string{
				strconv.Itoa(t.NameTokens),
				strconv.Itoa(t.DescTokens),
				strconv.Itoa(t.SchemaTokens),
				strconv.Itoa(t.TotalTokens),
			}
		},
	)

	renderNamespacedTable(
		results,
		"Unified Prompt Analysis (sorted by total tokens)",
		[]string{"Server", "Prompt (Namespaced)", "Name", "Desc", "Args", "Total"},
		func(r *ServerResult) []analyzer.PromptTokens { return r.PromptStats },
		func(p analyzer.PromptTokens) string { return p.Name },
		func(p analyzer.PromptTokens) int { return p.TotalTokens },
		func(p analyzer.PromptTokens) []string {
			return []string{
				strconv.Itoa(p.NameTokens),
				strconv.Itoa(p.DescTokens),
				strconv.Itoa(p.ArgsTokens),
				strconv.Itoa(p.TotalTokens),
			}
		},
	)

	renderNamespacedTable(
		results,
		"Unified Resource Analysis (sorted by total tokens)",
		[]string{"Server", "Resource (Namespaced)", "Name", "URI", "Desc", "Total"},
		func(r *ServerResult) []analyzer.ResourceTokens { return r.ResourceStats },
		func(res analyzer.ResourceTokens) string { return res.Name },
		func(res analyzer.ResourceTokens) int { return res.TotalTokens },
		func(res analyzer.ResourceTokens) []string {
			return []string{
				strconv.Itoa(res.NameTokens),
				strconv.Itoa(res.URITokens),
				strconv.Itoa(res.DescTokens),
				strconv.Itoa(res.TotalTokens),
			}
		},
	)
}
