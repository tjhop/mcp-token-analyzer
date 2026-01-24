package config

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Transport is the MCP transport type (stdio or http).
type Transport string

const (
	TransportStdio Transport = "stdio"
	TransportHTTP  Transport = "http"

	// TransportStreamableHTTP is an alias for TransportHTTP used by some MCP
	// clients (Continue, Roo Code, Anthropic MCP Registry). InferDefaults
	// normalizes it to TransportHTTP so downstream code only needs to handle
	// the canonical transport values.
	TransportStreamableHTTP Transport = "streamable-http"
)

// ServerConfig holds configuration for a single MCP server.
type ServerConfig struct {
	Name    string            `json:"-"`              // From map key, populated during parsing
	Type    Transport         `json:"type,omitempty"` // Inferred from fields if empty
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	EnvFile string            `json:"envFile,omitempty"`

	// Security options (parsed but NOT IMPLEMENTED - future work).
	// When present, these generate warnings via Config.Warnings().
	Auth *OAuthConfig `json:"auth,omitempty"`
	TLS  *TLSConfig   `json:"tls,omitempty"`
}

// OAuthConfig holds OAuth 2.0 client credentials (Cursor format).
// NOT IMPLEMENTED - parsed for future compatibility.
type OAuthConfig struct {
	ClientID     string   `json:"CLIENT_ID,omitempty"`
	ClientSecret string   `json:"CLIENT_SECRET,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
}

// TLSConfig holds custom TLS settings for HTTP transport.
// NOT IMPLEMENTED - parsed for future compatibility.
type TLSConfig struct {
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty"`
	CACertFile         string `json:"caCertFile,omitempty"`
	ClientCertFile     string `json:"clientCertFile,omitempty"`
	ClientKeyFile      string `json:"clientKeyFile,omitempty"`
}

// Config represents an MCP configuration file.
// It supports both Claude/Cursor format (mcpServers) and VS Code format (servers).
//
// Config is intended to be treated as read-only after parsing. The InferDefaults
// method populates derived fields, after which the configuration should not be
// modified. This allows safe concurrent access during multi-server analysis.
type Config struct {
	MCPServers map[string]*ServerConfig `json:"mcpServers,omitempty"` // Claude/Cursor format
	Servers    map[string]*ServerConfig `json:"servers,omitempty"`    // VS Code format
}

// LoadConfig loads and parses an MCP configuration file from the given path.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return ParseConfig(data)
}

// ParseConfig parses MCP configuration from JSON bytes.
func ParseConfig(data []byte) (*Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Populate server names from map keys
	for name, srv := range cfg.MCPServers {
		if srv != nil {
			srv.Name = name
		}
	}
	for name, srv := range cfg.Servers {
		if srv != nil {
			srv.Name = name
		}
	}

	return &cfg, nil
}

// GetServers returns a unified map of all servers from both mcpServers and servers keys.
// If both keys are present, mcpServers takes precedence for duplicate names.
//
// A new map is returned on each call. The map values are shared pointers to the
// underlying ServerConfig objects, so callers should not modify them.
func (c *Config) GetServers() map[string]*ServerConfig {
	result := make(map[string]*ServerConfig)

	// First add VS Code format servers
	for name, srv := range c.Servers {
		result[name] = srv
	}

	// Then overlay Claude/Cursor format (takes precedence)
	for name, srv := range c.MCPServers {
		result[name] = srv
	}

	return result
}

// InferDefaults populates default values for server configurations where they
// can be derived from other fields. In particular, it infers the transport
// Type from the Command or URL fields when Type is not explicitly set, and
// normalizes transport aliases (e.g. "streamable-http" -> "http").
// Callers should invoke InferDefaults before Validate to ensure transport
// types are resolved prior to validation.
func (c *Config) InferDefaults() {
	for _, srv := range c.GetServers() {
		if srv == nil {
			continue
		}

		// Normalize transport aliases to canonical values.
		if srv.Type == TransportStreamableHTTP {
			srv.Type = TransportHTTP
		}

		if srv.Type == "" {
			if srv.Command != "" {
				srv.Type = TransportStdio
			} else if srv.URL != "" {
				srv.Type = TransportHTTP
			}
		}
	}
}

// Validate checks the configuration for errors. It returns an error if any
// server configuration is invalid. Validate expects InferDefaults to have
// been called first so that transport types are already resolved; it does not
// perform any inference itself.
func (c *Config) Validate() error {
	servers := c.GetServers()

	if len(servers) == 0 {
		return errors.New("no servers defined in configuration")
	}

	var errs []string

	for name, srv := range servers {
		if srv == nil {
			errs = append(errs, fmt.Sprintf("server %q: nil configuration", name))
			continue
		}

		// Validate based on transport type
		switch srv.Type {
		case TransportStdio:
			if srv.Command == "" {
				errs = append(errs, fmt.Sprintf("server %q: stdio transport requires 'command' field", name))
			}
		case TransportHTTP:
			if srv.URL == "" {
				errs = append(errs, fmt.Sprintf("server %q: http transport requires 'url' field", name))
			} else if err := validateURL(srv.URL); err != nil {
				errs = append(errs, fmt.Sprintf("server %q: invalid url: %v", name, err))
			}
		case "":
			errs = append(errs, fmt.Sprintf("server %q: cannot infer transport type (need 'command' or 'url')", name))
		default:
			errs = append(errs, fmt.Sprintf("server %q: unknown transport type %q", name, srv.Type))
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

// validateURL checks that a URL is well-formed and uses http or https scheme.
func validateURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("malformed URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported scheme %q (must be http or https)", parsed.Scheme)
	}

	if parsed.Host == "" {
		return errors.New("missing host")
	}

	return nil
}

// Warnings returns a list of warning messages for unsupported configuration options.
// These are options that are parsed but not yet implemented.
func (c *Config) Warnings() []string {
	var warnings []string

	servers := c.GetServers()
	for name, srv := range servers {
		if srv == nil {
			continue
		}

		if srv.Auth != nil && (srv.Auth.ClientID != "" || srv.Auth.ClientSecret != "") {
			warnings = append(warnings, fmt.Sprintf("server %q: OAuth auth config present but not yet implemented (ignored)", name))
		}

		if srv.TLS != nil && (srv.TLS.InsecureSkipVerify || srv.TLS.CACertFile != "" || srv.TLS.ClientCertFile != "" || srv.TLS.ClientKeyFile != "") {
			warnings = append(warnings, fmt.Sprintf("server %q: TLS config present but not yet implemented (ignored)", name))
		}
	}

	return warnings
}

// LoadEnvFile parses a .env file and returns the key-value pairs.
// Lines starting with # are treated as comments. Empty lines are skipped.
// Format: KEY=value or KEY="value" (quotes are stripped).
func LoadEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open env file: %w", err)
	}
	defer file.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Find the first = sign
		idx := strings.Index(line, "=")
		if idx == -1 {
			return nil, fmt.Errorf("line %d: invalid format (missing '=')", lineNum)
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Strip surrounding quotes if present
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		if key == "" {
			return nil, fmt.Errorf("line %d: empty key", lineNum)
		}

		result[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading env file: %w", err)
	}

	return result, nil
}

// ResolveServerEnv resolves the environment variables for a server configuration.
// It merges the env map with any envFile contents (envFile values are loaded first,
// then env map values override). The configDir is used to resolve relative envFile paths.
func ResolveServerEnv(srv *ServerConfig, configDir string) (map[string]string, error) {
	result := make(map[string]string)

	// First, load envFile if specified
	if srv.EnvFile != "" {
		envPath := srv.EnvFile
		if !filepath.IsAbs(envPath) {
			envPath = filepath.Join(configDir, envPath)
		}

		envVars, err := LoadEnvFile(envPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load envFile %q: %w", srv.EnvFile, err)
		}

		for k, v := range envVars {
			result[k] = v
		}
	}

	// Then overlay the env map (takes precedence)
	for k, v := range srv.Env {
		result[k] = v
	}

	return result, nil
}
