// Package analyzer provides token counting and analysis functionality for MCP
// server artifacts including tools, prompts, and resources. It uses tiktoken
// encoding to count tokens compatible with various LLM models.
package analyzer

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkoukk/tiktoken-go"
)

// defaultTokenEncoding is the default tiktoken encoding used when no model is specified.
const defaultTokenEncoding = "cl100k_base"

// ToolTokens holds token count information for an MCP tool definition.
type ToolTokens struct {
	Name               string
	NameTokens         int
	DescTokens         int
	SchemaTokens       int
	OutputSchemaTokens int
	AnnotationsTokens  int
	TotalTokens        int
}

// Add accumulates all numeric fields from other into the receiver.
func (t *ToolTokens) Add(other ToolTokens) {
	t.NameTokens += other.NameTokens
	t.DescTokens += other.DescTokens
	t.SchemaTokens += other.SchemaTokens
	t.OutputSchemaTokens += other.OutputSchemaTokens
	t.AnnotationsTokens += other.AnnotationsTokens
	t.TotalTokens += other.TotalTokens
}

// PromptTokens holds token count information for an MCP prompt definition.
type PromptTokens struct {
	Name        string
	NameTokens  int
	DescTokens  int
	ArgsTokens  int
	TotalTokens int
}

// Add accumulates all numeric fields from other into the receiver.
func (p *PromptTokens) Add(other PromptTokens) {
	p.NameTokens += other.NameTokens
	p.DescTokens += other.DescTokens
	p.ArgsTokens += other.ArgsTokens
	p.TotalTokens += other.TotalTokens
}

// ResourceTokens holds token count information for an MCP resource or resource template.
type ResourceTokens struct {
	Name        string
	NameTokens  int
	URITokens   int
	DescTokens  int
	TotalTokens int
}

// Add accumulates all numeric fields from other into the receiver.
func (r *ResourceTokens) Add(other ResourceTokens) {
	r.NameTokens += other.NameTokens
	r.URITokens += other.URITokens
	r.DescTokens += other.DescTokens
	r.TotalTokens += other.TotalTokens
}

// TokenCounter wraps tiktoken to provide thread-safe token counting for MCP artifacts.
//
// Thread safety: The underlying tiktoken encoder is not documented as thread-safe,
// so TokenCounter uses a mutex to serialize access. When analyzing multiple servers
// concurrently, all goroutines share the same TokenCounter instance, creating a
// serialization point. This is acceptable for the current use case since token
// counting is fast relative to network I/O, but could be optimized by using a
// pool of encoders if profiling shows contention.
type TokenCounter struct {
	mu  sync.Mutex
	enc *tiktoken.Tiktoken
}

// NewTokenCounter creates a TokenCounter using the encoding for the specified model.
// If model is empty, it uses defaultTokenEncoding.
func NewTokenCounter(model string) (*TokenCounter, error) {
	var (
		encoder *tiktoken.Tiktoken
		err     error
	)

	if model != "" {
		encoder, err = tiktoken.EncodingForModel(model)
	} else {
		encoder, err = tiktoken.GetEncoding(defaultTokenEncoding)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get encoding: %w", err)
	}

	return &TokenCounter{enc: encoder}, nil
}

// CountTokens returns the number of tokens in the given text.
// It is safe for concurrent use.
func (c *TokenCounter) CountTokens(text string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.enc.Encode(text, nil, nil))
}

// AnalyzeTool counts tokens in a tool's name, description, input schema,
// output schema, and annotations.
func (c *TokenCounter) AnalyzeTool(tool *mcp.Tool) (ToolTokens, error) {
	schemaBytes, err := json.Marshal(tool.InputSchema)
	if err != nil {
		return ToolTokens{}, fmt.Errorf("failed to marshal input schema: %w", err)
	}

	nameTokens := c.CountTokens(tool.Name)
	descTokens := c.CountTokens(tool.Description)
	schemaTokens := c.CountTokens(string(schemaBytes))

	var outputSchemaTokens int
	if tool.OutputSchema != nil {
		outputBytes, err := json.Marshal(tool.OutputSchema)
		if err != nil {
			return ToolTokens{}, fmt.Errorf("failed to marshal output schema: %w", err)
		}
		outputSchemaTokens = c.CountTokens(string(outputBytes))
	}

	var annotationsTokens int
	if tool.Annotations != nil {
		annotBytes, err := json.Marshal(tool.Annotations)
		if err != nil {
			return ToolTokens{}, fmt.Errorf("failed to marshal annotations: %w", err)
		}
		annotationsTokens = c.CountTokens(string(annotBytes))
	}

	return ToolTokens{
		Name:               tool.Name,
		NameTokens:         nameTokens,
		DescTokens:         descTokens,
		SchemaTokens:       schemaTokens,
		OutputSchemaTokens: outputSchemaTokens,
		AnnotationsTokens:  annotationsTokens,
		TotalTokens:        nameTokens + descTokens + schemaTokens + outputSchemaTokens + annotationsTokens,
	}, nil
}

// AnalyzePrompt counts tokens in a prompt's name, description, and arguments.
func (c *TokenCounter) AnalyzePrompt(prompt *mcp.Prompt) (PromptTokens, error) {
	argsBytes, err := json.Marshal(prompt.Arguments)
	if err != nil {
		return PromptTokens{}, fmt.Errorf("failed to marshal prompt arguments: %w", err)
	}
	argsStr := string(argsBytes)

	nameTokens := c.CountTokens(prompt.Name)
	descTokens := c.CountTokens(prompt.Description)
	argsTokens := c.CountTokens(argsStr)

	return PromptTokens{
		Name:        prompt.Name,
		NameTokens:  nameTokens,
		DescTokens:  descTokens,
		ArgsTokens:  argsTokens,
		TotalTokens: nameTokens + descTokens + argsTokens,
	}, nil
}

// AnalyzeResource counts tokens in a resource's name, URI, and description.
func (c *TokenCounter) AnalyzeResource(resource *mcp.Resource) (ResourceTokens, error) {
	nameTokens := c.CountTokens(resource.Name)
	uriTokens := c.CountTokens(resource.URI)
	descTokens := c.CountTokens(resource.Description)

	return ResourceTokens{
		Name:        resource.Name,
		NameTokens:  nameTokens,
		URITokens:   uriTokens,
		DescTokens:  descTokens,
		TotalTokens: nameTokens + uriTokens + descTokens,
	}, nil
}

// AnalyzeResourceTemplate counts tokens in a resource template's name, URI template, and description.
func (c *TokenCounter) AnalyzeResourceTemplate(template *mcp.ResourceTemplate) (ResourceTokens, error) {
	nameTokens := c.CountTokens(template.Name)
	uriTokens := c.CountTokens(template.URITemplate)
	descTokens := c.CountTokens(template.Description)

	return ResourceTokens{
		Name:        template.Name,
		NameTokens:  nameTokens,
		URITokens:   uriTokens,
		DescTokens:  descTokens,
		TotalTokens: nameTokens + uriTokens + descTokens,
	}, nil
}
