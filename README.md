# MCP Token Analyzer

[![license](https://img.shields.io/github/license/tjhop/mcp-token-analyzer)](https://github.com/tjhop/mcp-token-analyzer/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/tjhop/mcp-token-analyzer)](https://goreportcard.com/report/github.com/tjhop/mcp-token-analyzer)
[![golangci-lint](https://github.com/tjhop/mcp-token-analyzer/actions/workflows/golangci-lint.yaml/badge.svg)](https://github.com/tjhop/mcp-token-analyzer/actions/workflows/golangci-lint.yaml)
[![Latest Release](https://img.shields.io/github/v/release/tjhop/mcp-token-analyzer)](https://github.com/tjhop/mcp-token-analyzer/releases/latest)
[![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/tjhop/mcp-token-analyzer/total)](https://github.com/tjhop/mcp-token-analyzer/releases/latest)

## About
`mcp-token-analyzer` is a CLI tool designed to connect to a Model Context Protocol (MCP) server, retrieve its tool definitions and schema, and analyze the token usage of each tool.
This tool assists developers in optimizing their tool descriptions and schemas for efficient context usage with LLMs.
It provides similar data to the `/context` slash command in claude code, except it's claude agnostic and specific to analyzing MCP servers.

Example analyzing [prometheus-mcp-server](https://github.com/tjhop/prometheus-mcp-server):

```bash
~/go/src/github.com/tjhop/mcp-token-analyzer (main [ ]) -> ./mcp-token-analyzer --mcp.transport http --mcp.url "http://localhost:8080/mcp"
Token Analysis Summary
┌───────────────────────┬──────────────┬───────┬─────────┬───────────┬──────────────┐
│      MCP Server       │ Instructions │ Tools │ Prompts │ Resources │ Total Tokens │
├───────────────────────┼──────────────┼───────┼─────────┼───────────┼──────────────┤
│ prometheus-mcp-server │ 4,527        │ 1,666 │ 0       │ 93        │ 6,286        │
├───────────────────────┼──────────────┼───────┼─────────┼───────────┼──────────────┤
│         TOTAL         │    4,527     │ 1,666 │    0    │    93     │    6,286     │
└───────────────────────┴──────────────┴───────┴─────────┴───────────┴──────────────┘

Tool Analysis (sorted by total tokens)
┌───────────────────────┬───────────────────┬──────┬──────┬────────┬───────┐
│        Server         │       Tool        │ Name │ Desc │ Schema │ Total │
├───────────────────────┼───────────────────┼──────┼──────┼────────┼───────┤
│ prometheus-mcp-server │ label_values      │ 2    │ 16   │ 156    │ 174   │
├───────────────────────┼───────────────────┼──────┼──────┼────────┼───────┤
│ prometheus-mcp-server │ range_query       │ 2    │ 8    │ 156    │ 166   │
├───────────────────────┼───────────────────┼──────┼──────┼────────┼───────┤
│ prometheus-mcp-server │ label_names       │ 2    │ 18   │ 135    │ 155   │
├───────────────────────┼───────────────────┼──────┼──────┼────────┼───────┤
│            ...        │       ...         │ ...  │ ...  │  ...   │  ...  │
├───────────────────────┼───────────────────┼──────┼──────┼────────┼───────┤
│ prometheus-mcp-server │ flags             │ 1    │ 3    │ 10     │ 14    │
├───────────────────────┼───────────────────┼──────┼──────┼────────┼───────┤
│ prometheus-mcp-server │ config            │ 1    │ 3    │ 10     │ 14    │
└───────────────────────┴───────────────────┴──────┴──────┴────────┴───────┘

Resource Analysis (sorted by total tokens)
┌───────────────────────┬─────────────────────────────────────────────────┬──────┬─────┬──────┬───────┐
│        Server         │                    Resource                     │ Name │ URI │ Desc │ Total │
├───────────────────────┼─────────────────────────────────────────────────┼──────┼─────┼──────┼───────┤
│ prometheus-mcp-server │ List of Official Prometheus Documentation Files │ 6    │ 4   │ 15   │ 25    │
├───────────────────────┼─────────────────────────────────────────────────┼──────┼─────┼──────┼───────┤
│ prometheus-mcp-server │ Official Prometheus Documentation               │ 3    │ 7   │ 15   │ 25    │
├───────────────────────┼─────────────────────────────────────────────────┼──────┼─────┼──────┼───────┤
│ prometheus-mcp-server │ TSDB Stats                                      │ 3    │ 6   │ 9    │ 18    │
├───────────────────────┼─────────────────────────────────────────────────┼──────┼─────┼──────┼───────┤
│ prometheus-mcp-server │ Targets                                         │ 1    │ 4   │ 10   │ 15    │
├───────────────────────┼─────────────────────────────────────────────────┼──────┼─────┼──────┼───────┤
│ prometheus-mcp-server │ List metrics                                    │ 2    │ 5   │ 3    │ 10    │
└───────────────────────┴─────────────────────────────────────────────────┴──────┴─────┴──────┴───────┘
```

## Features
- Transport Support
  - `stdio`: Execute a local binary
  - `http`: Connect to a streaming HTTP endpoint
  - `streamable-http`: Alias for `http`, used by some MCP clients (Continue, Roo Code, Anthropic MCP Registry)
- Configuration File Support
  - Load server configurations from `mcp.json` files
  - Compatible with Claude Desktop, Cursor, VS Code, and Continue formats
  - Analyze multiple MCP servers in parallel
  - Filter to a single server with `--server`
- Comprehensive MCP Analysis
  - Server Instructions: Token count for server-level instruction text
  - Tools: Token breakdown for names, descriptions, and input schemas
  - Prompts: Token breakdown for names, descriptions, and arguments
  - Resources & Templates: Token breakdown for names, URIs, and descriptions
- Token Counting
  - Uses `tiktoken` via [tiktoken-go](https://github.com/pkoukk/tiktoken-go) (defaults to `cl100k_base` / GPT-4)
  - Configurable tokenizer model via `--tokenizer.model`
  - See [Supported Tokenizer Models](#supported-tokenizer-models) for available models and encodings
- Reporting
  - Summary table showing token usage per server
  - Detail tables showing per-component token breakdowns (always shown for single-server, opt-in via `--detail` for multi-server)
  - Context window percentage calculation via `--limit`

## Installation and Usage

### Go

With a working go environment, the tool can be installed like so:

```bash
go install github.com/tjhop/mcp-token-analyzer@latest
/path/to/mcp-token-analyzer <flags>
```

### System Packages
Download a release appropriate for your system from the [Releases](https://github.com/tjhop/mcp-token-analyzer/releases) page.

```shell
# install system package (example assuming Debian based)
apt install /path/to/package
```

_Note_: While packages are built for several systems, there are currently no plans to attempt to submit packages to upstream package repositories.

### Building from Source

```bash
make build
./mcp-token-analyzer --help
```

## Security and Authentication

OAuth 2.0 and custom TLS configuration fields are parsed from config files but not yet implemented. When present, a warning is emitted. See the [ROADMAP](ROADMAP.md) for planned authentication support.

## Development
### Development Environment with Devbox + Direnv
If you use [Devbox](https://www.jetify.com/devbox) and
[Direnv](https://direnv.net/), then simply entering the directory for the repo
should set up the needed software.

### Building

The included Makefile has several targets to aid in development:

```bash
~/go/src/github.com/tjhop/mcp-token-analyzer (main [ ]) -> make

Usage:
  make <target>

Targets:
  help                           print this help message
  tidy                           tidy modules
  fmt                            apply go code style formatter
  lint                           run linters
  binary                         build a binary
  build                          alias for `binary`
  build-all                      test release process with goreleaser, does not publish/upload
  test                           run tests
```

## Configuration Files

The tool can load MCP server configurations from JSON files, supporting formats used by Claude Desktop, Cursor, VS Code, and Continue.

### Supported Formats

**Claude Desktop / Cursor format** (`mcpServers` key):
```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@anthropic/mcp-server-filesystem", "/tmp"],
      "env": { "DEBUG": "true" }
    },
    "remote-api": {
      "url": "http://localhost:3000/mcp",
      "headers": { "Authorization": "Bearer token123" }
    }
  }
}
```

**VS Code format** (`servers` key):
```json
{
  "servers": {
    "postgres": {
      "command": "npx",
      "args": ["-y", "@anthropic/mcp-server-postgres"]
    }
  }
}
```

**Continue format** (`mcpServers` with explicit `streamable-http` type):
```json
{
  "mcpServers": {
    "remote-api": {
      "type": "streamable-http",
      "url": "http://localhost:3000/mcp"
    }
  }
}
```

Both `mcpServers` and `servers` keys can be present in the same file; `mcpServers` takes precedence for duplicate server names.

### Server Configuration Options

| Field | Description |
|-------|-------------|
| `type` | Transport type (optional, inferred from `command` or `url` if omitted) |
| `command` | Command to execute (stdio transport) |
| `args` | Command arguments (stdio transport) |
| `url` | Server URL (http transport, must be http:// or https://) |
| `env` | Environment variables for the process |
| `envFile` | Path to .env file (relative to config file location) |
| `headers` | HTTP headers for requests (http transport) |

The transport type (`stdio` or `http`) is automatically inferred from the presence of `command` or `url` fields. The `streamable-http` type is accepted and normalized to `http`.

### Multi-Server Output

When analyzing multiple servers, the tool displays a summary table showing token usage across all servers. Use `--detail` to additionally display per-component detail tables (tools, prompts, resources) with entries from all servers sorted by total tokens. Each row includes the server name so components can be traced back to their origin.

When analyzing a single server (either ad-hoc via CLI flags or via `--server`), detail tables are always shown automatically.

## Supported Tokenizer Models

The `--tokenizer.model` flag accepts any model name recognized by [tiktoken-go](https://github.com/pkoukk/tiktoken-go). The model name determines which encoding (tokenization scheme) is used for counting. The default is `gpt-4` (`cl100k_base`).

### Encodings and Models

| Encoding | Models |
|----------|--------|
| `o200k_base` | `gpt-4.5`, `gpt-4.1`, `gpt-4o` (and dated variants like `gpt-4o-2024-05-13`) |
| `cl100k_base` | `gpt-4`, `gpt-3.5-turbo` (and dated variants), `text-embedding-ada-002`, `text-embedding-3-small`, `text-embedding-3-large` |
| `p50k_base` | `text-davinci-003`, `text-davinci-002`, `code-davinci-002`, `code-cushman-002` |
| `r50k_base` | `text-davinci-001`, `text-curie-001`, `text-babbage-001`, `text-ada-001`, `davinci`, `curie`, `babbage`, `ada` |

You can also pass an encoding name directly (e.g., `--tokenizer.model o200k_base`) if you prefer to specify the encoding rather than a model name.

### Limitations

- **Anthropic Claude models are not supported** by tiktoken-go. There is no official tokenizer for Claude models. When analyzing MCP servers used with Claude, the token counts are approximate. Using `o200k_base` or `cl100k_base` provides a reasonable estimate but will not match Claude's actual tokenization.
- tiktoken-go's model list reflects OpenAI's public models. Newer or experimental models may not be recognized until the library is updated.

## Command Line Flags

```
usage: mcp-token-analyzer [<flags>]

Flags:
  -h, --[no-]help                Show context-sensitive help (also try
                                 --help-long and --help-man).
  -t, --mcp.transport=stdio      Transport to use (stdio, http, streamable-http)
  -c, --mcp.command=MCP.COMMAND  Command to run (for stdio transport)
  -u, --mcp.url=MCP.URL          URL to connect to (for http transport)
  -m, --tokenizer.model="gpt-4"  Tokenizer model to use (e.g. gpt-4,
                                 gpt-3.5-turbo)
  -f, --config=CONFIG            Path to mcp.json config file
  -s, --server=SERVER            Analyze only this named server from config
      --[no-]detail              Show detailed per-server tables
      --limit=LIMIT              Context window limit for percentage calculation
```
