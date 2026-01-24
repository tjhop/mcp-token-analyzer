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
MCP Server Instructions Analysis
┌───────────────────────┬─────────────────────┐
│    MCP Server Name    │ Instructions tokens │
├───────────────────────┼─────────────────────┤
│ prometheus-mcp-server │ 4527                │
└───────────────────────┴─────────────────────┘

MCP Tool Analysis
┌───────────────────┬─────────────┬─────────────┬───────────────┬──────────────┐
│     Tool Name     │ Name Tokens │ Desc Tokens │ Schema Tokens │ Total Tokens │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ label_values      │ 2           │ 16          │ 156           │ 174          │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ range_query       │ 2           │ 8           │ 156           │ 166          │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ label_names       │ 2           │ 18          │ 135           │ 155          │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ series            │ 1           │ 6           │ 143           │ 150          │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ exemplar_query    │ 4           │ 9           │ 132           │ 145          │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ delete_series     │ 2           │ 11          │ 107           │ 120          │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ query             │ 1           │ 8           │ 91            │ 100          │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ targets_metadata  │ 2           │ 10          │ 68            │ 80           │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ docs_search       │ 2           │ 14          │ 49            │ 65           │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ metric_metadata   │ 2           │ 11          │ 48            │ 61           │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ snapshot          │ 1           │ 29          │ 30            │ 60           │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ docs_read         │ 2           │ 15          │ 34            │ 51           │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ ready             │ 1           │ 23          │ 10            │ 34           │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ reload            │ 1           │ 19          │ 10            │ 30           │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ clean_tombstones  │ 4           │ 14          │ 10            │ 28           │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ quit              │ 1           │ 15          │ 10            │ 26           │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ tsdb_stats        │ 3           │ 10          │ 10            │ 23           │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ healthy           │ 1           │ 12          │ 10            │ 23           │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ list_rules        │ 2           │ 10          │ 10            │ 22           │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ alertmanagers     │ 3           │ 7           │ 10            │ 20           │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ docs_list         │ 2           │ 7           │ 10            │ 19           │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ wal_replay_status │ 4           │ 5           │ 10            │ 19           │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ list_targets      │ 2           │ 6           │ 10            │ 18           │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ list_alerts       │ 3           │ 4           │ 10            │ 17           │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ runtime_info      │ 2           │ 4           │ 10            │ 16           │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ build_info        │ 2           │ 4           │ 10            │ 16           │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ flags             │ 1           │ 3           │ 10            │ 14           │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│ config            │ 1           │ 3           │ 10            │ 14           │
├───────────────────┼─────────────┼─────────────┼───────────────┼──────────────┤
│       TOTAL       │     56      │     301     │     1309      │     1666     │
└───────────────────┴─────────────┴─────────────┴───────────────┴──────────────┘

MCP Prompt Analysis
┌─────────────┬─────────────┬─────────────┬─────────────┬──────────────┐
│ Prompt Name │ Name Tokens │ Desc Tokens │ Args Tokens │ Total Tokens │
├─────────────┼─────────────┼─────────────┼─────────────┼──────────────┤
│    TOTAL    │      0      │      0      │      0      │      0       │
└─────────────┴─────────────┴─────────────┴─────────────┴──────────────┘

MCP Resource Analysis
┌─────────────────────────────────────────────────┬─────────────┬────────────┬─────────────┬──────────────┐
│                  Resource Name                  │ Name Tokens │ URI Tokens │ Desc Tokens │ Total Tokens │
├─────────────────────────────────────────────────┼─────────────┼────────────┼─────────────┼──────────────┤
│ List of Official Prometheus Documentation Files │ 6           │ 4          │ 15          │ 25           │
├─────────────────────────────────────────────────┼─────────────┼────────────┼─────────────┼──────────────┤
│ Official Prometheus Documentation               │ 3           │ 7          │ 15          │ 25           │
├─────────────────────────────────────────────────┼─────────────┼────────────┼─────────────┼──────────────┤
│ TSDB Stats                                      │ 3           │ 6          │ 9           │ 18           │
├─────────────────────────────────────────────────┼─────────────┼────────────┼─────────────┼──────────────┤
│ Targets                                         │ 1           │ 4          │ 10          │ 15           │
├─────────────────────────────────────────────────┼─────────────┼────────────┼─────────────┼──────────────┤
│ List metrics                                    │ 2           │ 5          │ 3           │ 10           │
├─────────────────────────────────────────────────┼─────────────┼────────────┼─────────────┼──────────────┤
│                      TOTAL                      │     15      │     26     │     52      │      93      │
└─────────────────────────────────────────────────┴─────────────┴────────────┴─────────────┴──────────────┘

Summary MCP Static Token Usage
┌─────────────────────────┬────────┐
│      MCP component      │ Tokens │
├─────────────────────────┼────────┤
│ Instructions            │ 4527   │
├─────────────────────────┼────────┤
│ Tools                   │ 1666   │
├─────────────────────────┼────────┤
│ Prompts                 │ 0      │
├─────────────────────────┼────────┤
│ Resources and Templates │ 93     │
├─────────────────────────┼────────┤
│          TOTAL          │  6286  │
└─────────────────────────┴────────┘
```

## Features
- Transport Support
  - `stdio`: Execute a local binary
  - `http`: Connect to a streaming HTTP endpoint
- Configuration File Support
  - Load server configurations from `mcp.json` files
  - Compatible with Claude Desktop, Cursor, and VS Code formats
  - Analyze multiple MCP servers in parallel
- Comprehensive MCP Analysis
  - Server Instructions: Token count for server-level instruction text
  - Tools: Token breakdown for names, descriptions, and input schemas
  - Prompts: Token breakdown for names, descriptions, and arguments
  - Resources & Templates: Token breakdown for names, URIs, and descriptions
- Token Counting
  - Uses `tiktoken` (defaults to `cl100k_base` / GPT-4)
- Detailed Reporting
  - Formatted tables showing token breakdowns per component
  - Summary of total static token usage across all MCP components
  - Group summary for multi-server analysis

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
./dist/mcp-token-analyzer_linux_amd64_v1/mcp-token-analyzer --help
```

## Security and Authentication
TODO

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

The tool can load MCP server configurations from JSON files, supporting formats used by Claude Desktop, Cursor, and VS Code.

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

### Server Configuration Options

| Field | Description |
|-------|-------------|
| `command` | Command to execute (stdio transport) |
| `args` | Command arguments (stdio transport) |
| `url` | Server URL (http transport, must be http:// or https://) |
| `env` | Environment variables for the process |
| `envFile` | Path to .env file (relative to config file) |
| `headers` | HTTP headers for requests (http transport) |

The transport type (`stdio` or `http`) is automatically inferred from the presence of `command` or `url` fields.

### Multi-Server Output

When analyzing multiple servers, the tool displays a group summary showing token usage across all servers. In `--detail` mode, tools, prompts, and resources are shown with namespaced identifiers using the format `server__component` (e.g., `prometheus__query`) to ensure uniqueness across servers.

## Command Line Flags

```bash
usage: mcp-token-analyzer [<flags>]

Flags:
  -h, --[no-]help                Show context-sensitive help (also try --help-long and --help-man).
  -t, --mcp.transport="stdio"    Transport to use (stdio, http)
  -c, --mcp.command=MCP.COMMAND  Command to run (for stdio transport)
  -u, --mcp.url=MCP.URL          URL to connect to (for http transport)
  -m, --tokenizer.model="gpt-4"  Tokenizer model to use (e.g. gpt-4, gpt-3.5-turbo)
  -f, --config=CONFIG            Path to mcp.json config file
  -s, --server=SERVER            Analyze only this named server from config
      --detail                   Show detailed per-server tables
      --limit=LIMIT              Context window limit for percentage calculation
```
