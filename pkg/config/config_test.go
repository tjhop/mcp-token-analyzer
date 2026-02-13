package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConfig_ClaudeDesktop(t *testing.T) {
	data, err := os.ReadFile("testdata/claude_desktop.json")
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	cfg, err := ParseConfig(data)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if len(cfg.MCPServers) != 2 {
		t.Errorf("expected 2 mcpServers, got %d", len(cfg.MCPServers))
	}

	// Check filesystem server
	fs := cfg.MCPServers["filesystem"]
	if fs == nil {
		t.Fatal("filesystem server not found")
	}
	if fs.Name != "filesystem" {
		t.Errorf("expected name 'filesystem', got %q", fs.Name)
	}
	if fs.Command != "npx" {
		t.Errorf("expected command 'npx', got %q", fs.Command)
	}
	if len(fs.Args) != 3 {
		t.Errorf("expected 3 args, got %d", len(fs.Args))
	}
	if fs.Env["DEBUG"] != "true" {
		t.Errorf("expected env DEBUG=true, got %q", fs.Env["DEBUG"])
	}

	// InferDefaults should set stdio transport, Validate should pass
	cfg.InferDefaults()
	if err := cfg.Validate(); err != nil {
		t.Errorf("validation failed: %v", err)
	}
	if fs.Type != TransportStdio {
		t.Errorf("expected stdio transport, got %q", fs.Type)
	}
}

func TestParseConfig_VSCode(t *testing.T) {
	data, err := os.ReadFile("testdata/vscode.json")
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	cfg, err := ParseConfig(data)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if len(cfg.Servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(cfg.Servers))
	}

	// Check postgres server (stdio)
	pg := cfg.Servers["postgres"]
	if pg == nil {
		t.Fatal("postgres server not found")
	}
	if pg.Command != "npx" {
		t.Errorf("expected command 'npx', got %q", pg.Command)
	}

	// Check remote-api server (http)
	remote := cfg.Servers["remote-api"]
	if remote == nil {
		t.Fatal("remote-api server not found")
	}
	if remote.URL != "http://localhost:3000/mcp" {
		t.Errorf("expected url 'http://localhost:3000/mcp', got %q", remote.URL)
	}
	if remote.Headers["Authorization"] != "Bearer token123" {
		t.Errorf("expected Authorization header, got %q", remote.Headers["Authorization"])
	}

	// InferDefaults should set transport types, Validate should pass
	cfg.InferDefaults()
	if err := cfg.Validate(); err != nil {
		t.Errorf("validation failed: %v", err)
	}
	if pg.Type != TransportStdio {
		t.Errorf("expected postgres to be stdio, got %q", pg.Type)
	}
	if remote.Type != TransportHTTP {
		t.Errorf("expected remote-api to be http, got %q", remote.Type)
	}
}

func TestParseConfig_Cursor(t *testing.T) {
	data, err := os.ReadFile("testdata/cursor.json")
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	cfg, err := ParseConfig(data)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	// Check prometheus server with envFile
	prom := cfg.MCPServers["prometheus"]
	if prom == nil {
		t.Fatal("prometheus server not found")
	}
	if prom.EnvFile != ".env" {
		t.Errorf("expected envFile '.env', got %q", prom.EnvFile)
	}

	// Check secure-server with OAuth and TLS (should generate warnings)
	secure := cfg.MCPServers["secure-server"]
	if secure == nil {
		t.Fatal("secure-server not found")
	}
	if secure.Auth == nil {
		t.Fatal("expected auth config")
	}
	if secure.Auth.ClientID != "my-client" {
		t.Errorf("expected ClientID 'my-client', got %q", secure.Auth.ClientID)
	}
	if secure.TLS == nil {
		t.Fatal("expected tls config")
	}
	if secure.TLS.CACertFile != "/path/to/ca.crt" {
		t.Errorf("expected CACertFile, got %q", secure.TLS.CACertFile)
	}

	cfg.InferDefaults()
	if err := cfg.Validate(); err != nil {
		t.Errorf("validation failed: %v", err)
	}

	// Should have warnings for unsupported options: one for OAuth, one for TLS.
	warnings := cfg.Warnings()
	if len(warnings) != 2 {
		t.Fatalf("expected 2 warnings, got %d: %v", len(warnings), warnings)
	}

	var foundOAuth, foundTLS bool
	for _, w := range warnings {
		if strings.Contains(w, "OAuth") {
			foundOAuth = true
		}
		if strings.Contains(w, "TLS") {
			foundTLS = true
		}
	}
	if !foundOAuth {
		t.Errorf("expected a warning mentioning 'OAuth', got: %v", warnings)
	}
	if !foundTLS {
		t.Errorf("expected a warning mentioning 'TLS', got: %v", warnings)
	}
}

func TestMergedServers_Merging(t *testing.T) {
	// Test that mcpServers takes precedence over servers for duplicate names
	cfg := &Config{
		MCPServers: map[string]*ServerConfig{
			"test": {Name: "test", Command: "from-mcpServers"},
		},
		Servers: map[string]*ServerConfig{
			"test":   {Name: "test", Command: "from-servers"},
			"unique": {Name: "unique", Command: "only-in-servers"},
		},
	}

	servers := cfg.MergedServers()
	if len(servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(servers))
	}

	if servers["test"].Command != "from-mcpServers" {
		t.Errorf("expected mcpServers to take precedence, got command %q", servers["test"].Command)
	}

	if servers["unique"] == nil {
		t.Error("unique server should be present")
	}
}

func TestValidate_NoServers(t *testing.T) {
	cfg := &Config{}
	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for empty config")
	}
}

func TestValidate_MissingCommand(t *testing.T) {
	cfg := &Config{
		MCPServers: map[string]*ServerConfig{
			"broken": {Name: "broken", Type: TransportStdio},
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for stdio without command")
	}
}

func TestValidate_MissingURL(t *testing.T) {
	cfg := &Config{
		MCPServers: map[string]*ServerConfig{
			"broken": {Name: "broken", Type: TransportHTTP},
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for http without url")
	}
}

func TestValidate_CannotInferTransport(t *testing.T) {
	cfg := &Config{
		MCPServers: map[string]*ServerConfig{
			"broken": {Name: "broken"},
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("expected error when transport cannot be inferred")
	}
}

func TestLoadEnvFile(t *testing.T) {
	envVars, err := LoadEnvFile("testdata/test.env")
	if err != nil {
		t.Fatalf("failed to load env file: %v", err)
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"API_KEY", "test-api-key"},
		{"DATABASE_URL", "postgres://localhost/testdb"},
		{"SECRET", "quoted-secret"},
		{"EMPTY_VALUE", ""},
		{"MULTI_WORD", "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := envVars[tt.key]; got != tt.expected {
				t.Errorf("expected %q=%q, got %q", tt.key, tt.expected, got)
			}
		})
	}
}

func TestLoadEnvFile_NotFound(t *testing.T) {
	_, err := LoadEnvFile("testdata/nonexistent.env")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestMergeServerEnv(t *testing.T) {
	// Create a temporary .env file
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envPath, []byte("FROM_FILE=file-value\nSHARED=from-file"), 0644); err != nil {
		t.Fatalf("failed to write temp env file: %v", err)
	}

	srv := &ServerConfig{
		EnvFile: ".env",
		Env: map[string]string{
			"FROM_MAP": "map-value",
			"SHARED":   "from-map", // Should override file value
		},
	}

	env, err := MergeServerEnv(srv, tmpDir)
	if err != nil {
		t.Fatalf("failed to resolve env: %v", err)
	}

	if env["FROM_FILE"] != "file-value" {
		t.Errorf("expected FROM_FILE=file-value, got %q", env["FROM_FILE"])
	}
	if env["FROM_MAP"] != "map-value" {
		t.Errorf("expected FROM_MAP=map-value, got %q", env["FROM_MAP"])
	}
	if env["SHARED"] != "from-map" {
		t.Errorf("expected SHARED=from-map (override), got %q", env["SHARED"])
	}
}

func TestMergeServerEnv_NoEnvFile(t *testing.T) {
	srv := &ServerConfig{
		Env: map[string]string{
			"KEY": "value",
		},
	}

	env, err := MergeServerEnv(srv, "")
	if err != nil {
		t.Fatalf("failed to resolve env: %v", err)
	}

	if env["KEY"] != "value" {
		t.Errorf("expected KEY=value, got %q", env["KEY"])
	}
}

func TestLoadConfig(t *testing.T) {
	cfg, err := LoadConfig("testdata/claude_desktop.json")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.MCPServers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(cfg.MCPServers))
	}
}

func TestLoadConfig_NotFound(t *testing.T) {
	_, err := LoadConfig("testdata/nonexistent.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParseConfig_InvalidJSON(t *testing.T) {
	_, err := ParseConfig([]byte("not valid json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestValidate_InvalidURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid_http", "http://localhost:3000/mcp", false},
		{"valid_https", "https://api.example.com/mcp", false},
		{"valid_with_path", "http://localhost:8080/api/v1/mcp", false},
		{"missing_scheme", "localhost:3000/mcp", true},
		{"invalid_scheme_ftp", "ftp://files.example.com/mcp", true},
		{"invalid_scheme_ws", "ws://localhost:3000/mcp", true},
		{"missing_host", "http:///mcp", true},
		// Note: empty_url tests the "missing URL" check (srv.URL == "") in Validate,
		// NOT the validateURL format check. It is included here for completeness since
		// it exercises the same http-transport validation branch.
		{"empty_url", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				MCPServers: map[string]*ServerConfig{
					"test": {Name: "test", Type: TransportHTTP, URL: tt.url},
				},
			}
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("expected error for URL %q", tt.url)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for URL %q: %v", tt.url, err)
			}
		})
	}
}

func TestParseConfig_StreamableHTTP(t *testing.T) {
	data, err := os.ReadFile("testdata/continue.json")
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	cfg, err := ParseConfig(data)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	remote := cfg.MCPServers["remote-api"]
	if remote == nil {
		t.Fatal("remote-api server not found")
	}
	if remote.Type != TransportStreamableHTTP {
		t.Errorf("expected type %q before InferDefaults, got %q", TransportStreamableHTTP, remote.Type)
	}

	// InferDefaults should normalize streamable-http to http
	cfg.InferDefaults()
	if remote.Type != TransportHTTP {
		t.Errorf("expected type %q after InferDefaults, got %q", TransportHTTP, remote.Type)
	}

	// Validation should pass
	if err := cfg.Validate(); err != nil {
		t.Errorf("validation failed: %v", err)
	}
}

func TestInferDefaults_StreamableHTTPNormalization(t *testing.T) {
	cfg := &Config{
		MCPServers: map[string]*ServerConfig{
			"explicit-streamable": {
				Name: "explicit-streamable",
				Type: TransportStreamableHTTP,
				URL:  "http://localhost:3000/mcp",
			},
			"inferred-http": {
				Name: "inferred-http",
				URL:  "http://localhost:3001/mcp",
			},
		},
	}

	cfg.InferDefaults()

	// Explicit streamable-http should be normalized to http
	if cfg.MCPServers["explicit-streamable"].Type != TransportHTTP {
		t.Errorf("expected streamable-http normalized to %q, got %q",
			TransportHTTP, cfg.MCPServers["explicit-streamable"].Type)
	}

	// Inferred type should still be http (not streamable-http)
	if cfg.MCPServers["inferred-http"].Type != TransportHTTP {
		t.Errorf("expected inferred type %q, got %q",
			TransportHTTP, cfg.MCPServers["inferred-http"].Type)
	}
}

func TestMergeServerEnv_PathTraversal(t *testing.T) {
	t.Run("relative_path_escapes_config_dir", func(t *testing.T) {
		// A relative envFile that uses ../ to escape the config directory
		// should be rejected before we ever attempt to open the file.
		configDir := t.TempDir()

		srv := &ServerConfig{
			EnvFile: "../../etc/shadow",
		}

		_, err := MergeServerEnv(srv, configDir)
		if err == nil {
			t.Fatal("expected error for envFile that escapes config directory via ../")
		}
		if !strings.Contains(err.Error(), "outside the config directory") {
			t.Errorf("expected 'outside the config directory' in error, got: %v", err)
		}
	})

	t.Run("relative_path_within_config_dir", func(t *testing.T) {
		// A relative envFile that stays within the config directory
		// should resolve and load successfully.
		configDir := t.TempDir()

		subDir := filepath.Join(configDir, "subdir")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}
		envPath := filepath.Join(subDir, "app.env")
		if err := os.WriteFile(envPath, []byte("NESTED_KEY=nested-value"), 0644); err != nil {
			t.Fatalf("failed to write env file: %v", err)
		}

		srv := &ServerConfig{
			EnvFile: filepath.Join("subdir", "app.env"),
		}

		env, err := MergeServerEnv(srv, configDir)
		if err != nil {
			t.Fatalf("expected success for envFile within config directory, got: %v", err)
		}
		if env["NESTED_KEY"] != "nested-value" {
			t.Errorf("expected NESTED_KEY=nested-value, got %q", env["NESTED_KEY"])
		}
	})

	t.Run("absolute_path_bypasses_confinement", func(t *testing.T) {
		// An absolute envFile path is treated as explicit and should
		// bypass the config directory confinement check entirely.
		envDir := t.TempDir()
		configDir := t.TempDir() // deliberately a different directory

		envPath := filepath.Join(envDir, "external.env")
		if err := os.WriteFile(envPath, []byte("ABS_KEY=abs-value"), 0644); err != nil {
			t.Fatalf("failed to write env file: %v", err)
		}

		srv := &ServerConfig{
			EnvFile: envPath, // absolute path outside configDir
		}

		env, err := MergeServerEnv(srv, configDir)
		if err != nil {
			t.Fatalf("expected success for absolute envFile path, got: %v", err)
		}
		if env["ABS_KEY"] != "abs-value" {
			t.Errorf("expected ABS_KEY=abs-value, got %q", env["ABS_KEY"])
		}
	})
}

func TestInferDefaults_ServersOnly(t *testing.T) {
	// Verify that InferDefaults correctly infers transport types for servers
	// that only appear in the Servers map (VS Code format), ensuring it
	// processes both map sources via MergedServers.
	cfg := &Config{
		Servers: map[string]*ServerConfig{
			"stdio-server": {
				Name:    "stdio-server",
				Command: "my-mcp-server",
				Args:    []string{"--verbose"},
			},
			"http-server": {
				Name: "http-server",
				URL:  "http://localhost:3000/mcp",
			},
		},
	}

	cfg.InferDefaults()

	stdio := cfg.Servers["stdio-server"]
	if stdio.Type != TransportStdio {
		t.Errorf("expected stdio-server type %q, got %q", TransportStdio, stdio.Type)
	}

	http := cfg.Servers["http-server"]
	if http.Type != TransportHTTP {
		t.Errorf("expected http-server type %q, got %q", TransportHTTP, http.Type)
	}

	// Validate should also pass after InferDefaults.
	if err := cfg.Validate(); err != nil {
		t.Errorf("validation failed after InferDefaults: %v", err)
	}
}

func TestLoadEnvFile_MalformedLines(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantSub string // expected substring in error message
	}{
		{
			name:    "missing_equals",
			content: "NO_EQUALS_HERE",
			wantSub: "missing '='",
		},
		{
			name:    "empty_key",
			content: "=value",
			wantSub: "empty key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			envPath := filepath.Join(tmpDir, ".env")
			if err := os.WriteFile(envPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write temp env file: %v", err)
			}

			_, err := LoadEnvFile(envPath)
			if err == nil {
				t.Fatalf("expected error for content %q, got nil", tt.content)
			}
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("expected error containing %q, got: %v", tt.wantSub, err)
			}
		})
	}
}

func TestMergedServers_CachedResult(t *testing.T) {
	srv1 := &ServerConfig{Name: "server1", Command: "cmd1"}
	srv2 := &ServerConfig{Name: "server2", Command: "cmd2"}
	cfg := &Config{
		MCPServers: map[string]*ServerConfig{"server1": srv1},
		Servers:    map[string]*ServerConfig{"server2": srv2},
	}

	servers1 := cfg.MergedServers()
	if len(servers1) != 2 {
		t.Errorf("expected 2 servers, got %d", len(servers1))
	}

	// Returned map values should point to the same ServerConfig objects.
	if servers1["server1"] != srv1 {
		t.Error("expected server1 to be the same pointer as the original")
	}
	if servers1["server2"] != srv2 {
		t.Error("expected server2 to be the same pointer as the original")
	}

	// Subsequent calls should return the same cached map.
	servers2 := cfg.MergedServers()
	if len(servers2) != len(servers1) {
		t.Errorf("expected cached result with %d servers, got %d", len(servers1), len(servers2))
	}
	if servers2["server1"] != srv1 || servers2["server2"] != srv2 {
		t.Error("cached result should contain the same pointers")
	}
}
