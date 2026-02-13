package analyzer

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestAnalyzeTool(t *testing.T) {
	counter, err := NewTokenCounter("gpt-4")
	if err != nil {
		t.Fatalf("failed to create counter: %v", err)
	}

	tool := &mcp.Tool{
		Name:        "test-tool",
		Description: "A test tool description",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"param1": map[string]any{
					"type": "string",
				},
			},
		},
	}

	stats, err := counter.AnalyzeTool(tool)
	if err != nil {
		t.Fatalf("failed to analyze tool: %v", err)
	}

	if stats.Name != tool.Name {
		t.Errorf("expected name %s, got %s", tool.Name, stats.Name)
	}

	if stats.NameTokens <= 0 {
		t.Errorf("expected positive name tokens, got %d", stats.NameTokens)
	}

	if stats.DescTokens <= 0 {
		t.Errorf("expected positive desc tokens, got %d", stats.DescTokens)
	}

	if stats.SchemaTokens <= 0 {
		t.Errorf("expected positive schema tokens, got %d", stats.SchemaTokens)
	}

	expectedTotal := stats.NameTokens + stats.DescTokens + stats.SchemaTokens
	if stats.TotalTokens != expectedTotal {
		t.Errorf("expected total tokens %d, got %d", expectedTotal, stats.TotalTokens)
	}
}

func TestAnalyzePrompt(t *testing.T) {
	counter, err := NewTokenCounter("gpt-4")
	if err != nil {
		t.Fatalf("failed to create counter: %v", err)
	}

	prompt := &mcp.Prompt{
		Name:        "test-prompt",
		Description: "A test prompt description",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "arg1",
				Description: "An argument",
				Required:    true,
			},
		},
	}

	stats, err := counter.AnalyzePrompt(prompt)
	if err != nil {
		t.Fatalf("failed to analyze prompt: %v", err)
	}

	if stats.Name != prompt.Name {
		t.Errorf("expected name %s, got %s", prompt.Name, stats.Name)
	}

	if stats.NameTokens <= 0 || stats.DescTokens <= 0 || stats.ArgsTokens <= 0 {
		t.Errorf("expected positive tokens, got Name:%d, Desc:%d, Args:%d", stats.NameTokens, stats.DescTokens, stats.ArgsTokens)
	}

	expectedTotal := stats.NameTokens + stats.DescTokens + stats.ArgsTokens
	if stats.TotalTokens != expectedTotal {
		t.Errorf("expected total tokens %d, got %d", expectedTotal, stats.TotalTokens)
	}
}

func TestAnalyzeResource(t *testing.T) {
	counter, err := NewTokenCounter("gpt-4")
	if err != nil {
		t.Fatalf("failed to create counter: %v", err)
	}

	resource := &mcp.Resource{
		Name:        "test-resource",
		URI:         "file:///test/path",
		Description: "A test resource description",
	}

	stats, err := counter.AnalyzeResource(resource)
	if err != nil {
		t.Fatalf("failed to analyze resource: %v", err)
	}

	if stats.Name != resource.Name {
		t.Errorf("expected name %s, got %s", resource.Name, stats.Name)
	}

	if stats.NameTokens <= 0 || stats.URITokens <= 0 || stats.DescTokens <= 0 {
		t.Errorf("expected positive tokens, got Name:%d, URI:%d, Desc:%d", stats.NameTokens, stats.URITokens, stats.DescTokens)
	}

	expectedTotal := stats.NameTokens + stats.URITokens + stats.DescTokens
	if stats.TotalTokens != expectedTotal {
		t.Errorf("expected total tokens %d, got %d", expectedTotal, stats.TotalTokens)
	}
}

func TestAnalyzeResourceTemplate(t *testing.T) {
	counter, err := NewTokenCounter("gpt-4")
	if err != nil {
		t.Fatalf("failed to create counter: %v", err)
	}

	template := &mcp.ResourceTemplate{
		Name:        "test-template",
		URITemplate: "file:///test/{path}",
		Description: "A test template description",
	}

	stats, err := counter.AnalyzeResourceTemplate(template)
	if err != nil {
		t.Fatalf("failed to analyze template: %v", err)
	}

	if stats.Name != template.Name {
		t.Errorf("expected name %s, got %s", template.Name, stats.Name)
	}

	if stats.NameTokens <= 0 || stats.URITokens <= 0 || stats.DescTokens <= 0 {
		t.Errorf("expected positive tokens, got Name:%d, URI:%d, Desc:%d", stats.NameTokens, stats.URITokens, stats.DescTokens)
	}

	expectedTotal := stats.NameTokens + stats.URITokens + stats.DescTokens
	if stats.TotalTokens != expectedTotal {
		t.Errorf("expected total tokens %d, got %d", expectedTotal, stats.TotalTokens)
	}
}

func TestCountTokens_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		model   string
		input   string
		wantErr bool // error expected from NewTokenCounter
	}{
		// Empty and whitespace inputs
		{"empty string", "gpt-4", "", false},
		{"whitespace only spaces", "gpt-4", "     ", false},
		{"whitespace only tabs", "gpt-4", "\t\t\t", false},
		{"whitespace only newlines", "gpt-4", "\n\n\n", false},
		{"whitespace mixed", "gpt-4", "   \t\n  \r\n  ", false},

		// Unicode: various languages and scripts
		{"unicode japanese", "gpt-4", "こんにちは", false},
		{"unicode chinese", "gpt-4", "你好世界", false},
		{"unicode korean", "gpt-4", "안녕하세요", false},
		{"unicode arabic", "gpt-4", "مرحبا بالعالم", false},
		{"unicode cyrillic", "gpt-4", "Привет мир", false},
		{"unicode hebrew", "gpt-4", "שלום עולם", false},
		{"unicode thai", "gpt-4", "สวัสดีโลก", false},
		{"unicode mixed scripts", "gpt-4", "Hello 世界 Мир مرحبا", false},

		// Emojis
		{"unicode emoji single", "gpt-4", "👋", false},
		{"unicode emoji multiple", "gpt-4", "Hello 👋 World 🌍", false},
		{"unicode emoji sequence", "gpt-4", "👨‍👩‍👧‍👦", false}, // family emoji (ZWJ sequence)
		{"unicode emoji flags", "gpt-4", "🇺🇸🇬🇧🇯🇵", false},
		{"unicode emoji skin tones", "gpt-4", "👋🏻👋🏼👋🏽👋🏾👋🏿", false},

		// Special characters
		{"special chars punctuation", "gpt-4", "!@#$%^&*()_+-=[]{}|;':\",./<>?", false},
		{"special chars control", "gpt-4", "\x00\x01\x02\x03", false},
		{"special chars unicode symbols", "gpt-4", "© ® ™ € £ ¥ ¢", false},
		{"special chars math", "gpt-4", "∑ ∏ ∫ ∂ √ ∞ ≈ ≠ ≤ ≥", false},
		{"special chars box drawing", "gpt-4", "┌─┬─┐│ │ │└─┴─┘", false},

		// Long inputs
		{"very long input", "gpt-4", strings.Repeat("test ", 1000), false},
		{"very long single word", "gpt-4", strings.Repeat("a", 10000), false},
		{"very long unicode", "gpt-4", strings.Repeat("日本語", 1000), false},

		// Different models
		{"different model gpt-3.5-turbo", "gpt-3.5-turbo", "Hello world", false},
		{"different model gpt-4o", "gpt-4o", "Hello world", false},
		{"different model text-embedding-ada-002", "text-embedding-ada-002", "Hello world", false},

		// Edge case: empty model uses default encoding
		{"empty model default encoding", "", "Hello world", false},

		// Mixed content
		{"mixed ascii and unicode", "gpt-4", "Hello世界!こんにちは👋", false},
		{"code snippet", "gpt-4", "func main() {\n\tfmt.Println(\"Hello\")\n}", false},
		{"json content", "gpt-4", `{"key": "value", "number": 42, "array": [1, 2, 3]}`, false},
		{"markdown content", "gpt-4", "# Header\n\n**bold** _italic_ `code`\n\n- list item", false},

		// Boundary cases
		{"single character", "gpt-4", "a", false},
		{"single space", "gpt-4", " ", false},
		{"single newline", "gpt-4", "\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter, err := NewTokenCounter(tt.model)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewTokenCounter() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}

			// CountTokens doesn't return an error, just verify it doesn't panic
			// and returns a sensible value.
			tokens := counter.CountTokens(tt.input)

			if tt.input == "" {
				if tokens != 0 {
					t.Errorf("CountTokens() for empty string = %d, want 0", tokens)
				}
			} else {
				// All non-empty inputs should produce at least one token.
				if tokens <= 0 {
					t.Errorf("CountTokens() for non-empty input %q = %d, want > 0", tt.input, tokens)
				}
			}
		})
	}
}

func TestCountTokens_KnownValues(t *testing.T) {
	// Pin exact token counts for representative inputs using the gpt-4
	// model (cl100k_base encoding). These serve as regression anchors --
	// if the underlying tokenizer changes behavior, these tests will catch it.
	counter, err := NewTokenCounter("gpt-4")
	if err != nil {
		t.Fatalf("failed to create counter: %v", err)
	}

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"single_word", "Hello", 1},
		{"two_words", "Hello world", 2},
		{"short_sentence", "The quick brown fox", 4},
		{"json_object", `{"key":"value"}`, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := counter.CountTokens(tt.input)
			if got != tt.want {
				t.Errorf("CountTokens(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestToolTokens_Add(t *testing.T) {
	a := ToolTokens{
		Name:               "a",
		NameTokens:         1,
		DescTokens:         2,
		SchemaTokens:       3,
		OutputSchemaTokens: 4,
		AnnotationsTokens:  5,
		TotalTokens:        15,
	}
	b := ToolTokens{
		Name:               "b",
		NameTokens:         10,
		DescTokens:         20,
		SchemaTokens:       30,
		OutputSchemaTokens: 40,
		AnnotationsTokens:  50,
		TotalTokens:        150,
	}

	a.Add(b)

	if a.NameTokens != 11 {
		t.Errorf("NameTokens = %d, want 11", a.NameTokens)
	}
	if a.DescTokens != 22 {
		t.Errorf("DescTokens = %d, want 22", a.DescTokens)
	}
	if a.SchemaTokens != 33 {
		t.Errorf("SchemaTokens = %d, want 33", a.SchemaTokens)
	}
	if a.OutputSchemaTokens != 44 {
		t.Errorf("OutputSchemaTokens = %d, want 44", a.OutputSchemaTokens)
	}
	if a.AnnotationsTokens != 55 {
		t.Errorf("AnnotationsTokens = %d, want 55", a.AnnotationsTokens)
	}
	if a.TotalTokens != 165 {
		t.Errorf("TotalTokens = %d, want 165", a.TotalTokens)
	}
	// Add should not modify the Name field.
	if a.Name != "a" {
		t.Errorf("Name = %q, want %q (should be unchanged)", a.Name, "a")
	}
}

func TestPromptTokens_Add(t *testing.T) {
	a := PromptTokens{
		Name:        "a",
		NameTokens:  1,
		DescTokens:  2,
		ArgsTokens:  3,
		TotalTokens: 6,
	}
	b := PromptTokens{
		Name:        "b",
		NameTokens:  10,
		DescTokens:  20,
		ArgsTokens:  30,
		TotalTokens: 60,
	}

	a.Add(b)

	if a.NameTokens != 11 {
		t.Errorf("NameTokens = %d, want 11", a.NameTokens)
	}
	if a.DescTokens != 22 {
		t.Errorf("DescTokens = %d, want 22", a.DescTokens)
	}
	if a.ArgsTokens != 33 {
		t.Errorf("ArgsTokens = %d, want 33", a.ArgsTokens)
	}
	if a.TotalTokens != 66 {
		t.Errorf("TotalTokens = %d, want 66", a.TotalTokens)
	}
	if a.Name != "a" {
		t.Errorf("Name = %q, want %q (should be unchanged)", a.Name, "a")
	}
}

func TestResourceTokens_Add(t *testing.T) {
	a := ResourceTokens{
		Name:        "a",
		NameTokens:  1,
		URITokens:   2,
		DescTokens:  3,
		TotalTokens: 6,
	}
	b := ResourceTokens{
		Name:        "b",
		NameTokens:  10,
		URITokens:   20,
		DescTokens:  30,
		TotalTokens: 60,
	}

	a.Add(b)

	if a.NameTokens != 11 {
		t.Errorf("NameTokens = %d, want 11", a.NameTokens)
	}
	if a.URITokens != 22 {
		t.Errorf("URITokens = %d, want 22", a.URITokens)
	}
	if a.DescTokens != 33 {
		t.Errorf("DescTokens = %d, want 33", a.DescTokens)
	}
	if a.TotalTokens != 66 {
		t.Errorf("TotalTokens = %d, want 66", a.TotalTokens)
	}
	if a.Name != "a" {
		t.Errorf("Name = %q, want %q (should be unchanged)", a.Name, "a")
	}
}

func TestNewTokenCounter_InvalidModel(t *testing.T) {
	// Test that an invalid model name returns an error
	_, err := NewTokenCounter("invalid-model-name-that-does-not-exist")
	if err == nil {
		t.Error("NewTokenCounter() with invalid model should return error")
	}
}
