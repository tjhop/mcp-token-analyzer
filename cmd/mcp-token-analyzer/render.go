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

// renderSummary renders the summary table for all server results.
func renderSummary(results []*ServerResult) {
	fmt.Println("\nToken Analysis Summary")
	summaryTable := table.New(os.Stdout)
	summaryTable.SetHeaders("MCP Server", "Instructions", "Tools", "Prompts", "Resources", "Total Tokens")

	var totalInstructionTokens, totalTools, totalPrompts, totalResources, grandTotal int

	for _, r := range results {
		if r.Error != nil {
			summaryTable.AddRow(
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
		summaryTable.AddRow(
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

	summaryTable.AddFooters(
		tableLabelTotal,
		printer.Sprintf("%d", totalInstructionTokens),
		printer.Sprintf("%d", totalTools),
		printer.Sprintf("%d", totalPrompts),
		printer.Sprintf("%d", totalResources),
		printer.Sprintf("%d", grandTotal),
	)
	summaryTable.Render()

	renderContextUsage(grandTotal)
}

// detailItem holds a stats value associated with a server for detail tables.
type detailItem[T any] struct {
	Server string
	Stats  T
}

// renderDetailTable collects items from all server results, sorts by total tokens,
// and renders a per-component detail table.
func renderDetailTable[T any](
	results []*ServerResult,
	title string,
	headers []string,
	extract func(r *ServerResult) []T,
	itemName func(T) string,
	totalTokens func(T) int,
	rowValues func(T) []string,
) {
	var items []detailItem[T]
	for _, r := range results {
		if r.Error != nil {
			continue
		}
		for _, item := range extract(r) {
			items = append(items, detailItem[T]{
				Server: r.Name,
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
		row := append([]string{item.Server, itemName(item.Stats)}, rowValues(item.Stats)...)
		t.AddRow(row...)
	}
	t.Render()
}

// renderDetailTables renders per-component detail tables across all servers.
func renderDetailTables(results []*ServerResult) {
	renderDetailTable(
		results,
		"Tool Analysis (sorted by total tokens)",
		[]string{"Server", "Tool", "Name", "Desc", "Schema", "Total"},
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

	renderDetailTable(
		results,
		"Prompt Analysis (sorted by total tokens)",
		[]string{"Server", "Prompt", "Name", "Desc", "Args", "Total"},
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

	renderDetailTable(
		results,
		"Resource Analysis (sorted by total tokens)",
		[]string{"Server", "Resource", "Name", "URI", "Desc", "Total"},
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
