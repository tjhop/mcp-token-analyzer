// Package analyzer provides token counting and analysis functionality for MCP
// server artifacts including tools, prompts, and resources. It uses tiktoken
// encoding to count tokens compatible with various LLM models.
package analyzer

import (
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkoukk/tiktoken-go"
)

// DefaultTokenEncoding is the default tiktoken encoding used when no model is specified.
const DefaultTokenEncoding = "cl100k_base"

// ToolTokens holds token count information for an MCP tool definition.
type ToolTokens struct {
	Name         string
	NameTokens   int
	DescTokens   int
	SchemaTokens int
	TotalTokens  int
}

// PromptTokens holds token count information for an MCP prompt definition.
type PromptTokens struct {
	Name        string
	NameTokens  int
	DescTokens  int
	ArgsTokens  int
	TotalTokens int
}

// ResourceTokens holds token count information for an MCP resource or resource template.
type ResourceTokens struct {
	Name        string
	NameTokens  int
	URITokens   int
	DescTokens  int
	TotalTokens int
}

// TokenCounter wraps tiktoken to provide token counting for MCP artifacts.
type TokenCounter struct {
	*tiktoken.Tiktoken
}

// NewTokenCounter creates a TokenCounter using the encoding for the specified model.
// If model is empty, it uses DefaultTokenEncoding.
func NewTokenCounter(model string) (*TokenCounter, error) {
	var (
		tkm *tiktoken.Tiktoken
		err error
	)

	if model != "" {
		tkm, err = tiktoken.EncodingForModel(model)
	} else {
		tkm, err = tiktoken.GetEncoding(DefaultTokenEncoding)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get encoding: %w", err)
	}

	return &TokenCounter{tkm}, nil
}

// AnalyzeTool counts tokens in a tool's name, description, and input schema.
func (c *TokenCounter) AnalyzeTool(tool *mcp.Tool) (ToolTokens, error) {
	schemaBytes, err := json.Marshal(tool.InputSchema)
	if err != nil {
		return ToolTokens{}, fmt.Errorf("failed to marshal input schema: %w", err)
	}
	schemaStr := string(schemaBytes)

	nameTokens := len(c.Encode(tool.Name, nil, nil))
	descTokens := len(c.Encode(tool.Description, nil, nil))
	schemaTokens := len(c.Encode(schemaStr, nil, nil))

	return ToolTokens{
		Name:         tool.Name,
		NameTokens:   nameTokens,
		DescTokens:   descTokens,
		SchemaTokens: schemaTokens,
		TotalTokens:  nameTokens + descTokens + schemaTokens,
	}, nil
}

// AnalyzePrompt counts tokens in a prompt's name, description, and arguments.
func (c *TokenCounter) AnalyzePrompt(prompt *mcp.Prompt) (PromptTokens, error) {
	argsBytes, err := json.Marshal(prompt.Arguments)
	if err != nil {
		return PromptTokens{}, fmt.Errorf("failed to marshal prompt arguments: %w", err)
	}
	argsStr := string(argsBytes)

	nameTokens := len(c.Encode(prompt.Name, nil, nil))
	descTokens := len(c.Encode(prompt.Description, nil, nil))
	argsTokens := len(c.Encode(argsStr, nil, nil))

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
	nameTokens := len(c.Encode(resource.Name, nil, nil))
	uriTokens := len(c.Encode(resource.URI, nil, nil))
	descTokens := len(c.Encode(resource.Description, nil, nil))

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
	nameTokens := len(c.Encode(template.Name, nil, nil))
	uriTokens := len(c.Encode(template.URITemplate, nil, nil))
	descTokens := len(c.Encode(template.Description, nil, nil))

	return ResourceTokens{
		Name:        template.Name,
		NameTokens:  nameTokens,
		URITokens:   uriTokens,
		DescTokens:  descTokens,
		TotalTokens: nameTokens + uriTokens + descTokens,
	}, nil
}
