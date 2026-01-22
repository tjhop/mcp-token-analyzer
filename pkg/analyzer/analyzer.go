package analyzer

import (
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkoukk/tiktoken-go"
)

type ToolTokens struct {
	Name         string
	NameTokens   int
	DescTokens   int
	SchemaTokens int
	TotalTokens  int
}

type TokenCounter struct {
	*tiktoken.Tiktoken
}

func NewTokenCounter(model string) (*TokenCounter, error) {
	tkm, err := tiktoken.GetEncoding("cl100k_base")
	if model != "" {
		tkm, err = tiktoken.EncodingForModel(model)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get encoding: %w", err)
	}
	return &TokenCounter{tkm}, nil
}

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
