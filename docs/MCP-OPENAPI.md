# Polygolem MCP and OpenAPI Read-Only Deployment Notes

## What this is

Polygolem exposes agent-friendly read-only integration surfaces without adding
Python, Node, hosted proxy, or live-trading risk.

## Start here

- Use MCP when an agent speaks JSON-RPC over stdio (`cmd/polygolem_mcp/main.go:13`).
- Use OpenAPI when local proxy/tooling wants a static read-only route map (`cmd/polygolem_openapi/main.go:11`).
- Use the regular CLI for anything mutating; MCP/OpenAPI v1 intentionally excludes trading, signing, approvals, withdrawals, and relayer submission (`pkg/openapi/openapi.go:6`).

## Source anchors

| Claim | Source |
|---|---|
| The stdio MCP binary creates `mcp.NewServer()` and handles one JSON-RPC request per input line. | `cmd/polygolem_mcp/main.go:14`, `cmd/polygolem_mcp/main.go:17`, `cmd/polygolem_mcp/main.go:27` |
| The default MCP server exposes the manifest but does not execute tools until handlers are supplied. | `pkg/mcp/mcp.go:68`, `pkg/mcp/mcp.go:70`, `pkg/mcp/mcp.go:141` |
| Safe MCP tools are declared in code, not hand-maintained in this page. | `pkg/mcp/mcp.go:86`, `pkg/mcp/mcp.go:89`, `pkg/mcp/mcp.go:94` |
| Handler execution has a default timeout and requires configured read-only adapters. | `pkg/mcp/handlers.go:9`, `pkg/mcp/handlers.go:31`, `pkg/mcp/handlers.go:52` |
| The OpenAPI binary pretty-prints `openapi.Spec()`. | `cmd/polygolem_openapi/main.go:13`, `cmd/polygolem_openapi/main.go:14` |
| OpenAPI paths live in `pkg/openapi`. | `pkg/openapi/openapi.go:14`, `pkg/openapi/openapi.go:23`, `pkg/openapi/openapi.go:36` |

## Safety boundary

MCP and OpenAPI v1 are read-only only. They must not expose:

- live order placement or cancellation;
- private-key signing;
- token approvals;
- bridge withdrawals;
- deposit-wallet relayer submission;
- authenticated trading mutations.

Use the regular CLI safety gates for any future mutating flow.

## MCP stdio server

Run the minimal MCP stdio server:

```bash
go run ./cmd/polygolem_mcp
```

List safe tools with one JSON-RPC line:

```bash
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' \
  | go run ./cmd/polygolem_mcp
```

Current tool manifest:

- `polygolem.health`
- `polygolem.discover_search`
- `polygolem.data_positions`
- `polygolem.orderbook_book`
- `polygolem.marketdata_snapshot`

By default, `cmd/polygolem_mcp` exposes the manifest and rejects unconfigured
execution. Embedders can use `pkg/mcp.NewServerWithHandlers` or
`pkg/mcp.NewSDKReadOnlyHandlers` to wire concrete read-only SDK clients under a
per-call timeout (`pkg/mcp/mcp.go:75`, `pkg/mcp/sdk_handlers.go:18`).

## OpenAPI spec

Print the read-only OpenAPI 3.1 document:

```bash
go run ./cmd/polygolem_openapi
```

Current paths:

- `GET /health`
- `GET /diag`
- `GET /discover/search?q=...`
- `GET /data/positions?user=...`
- `GET /orderbook/{token_id}`
- `GET /marketdata/snapshot?token_id=...`

This is a spec artifact for local proxy/tooling experiments. Polygolem does not
ship a hosted proxy service.

## Go embedding sketch

```go
handlers := mcp.NewSDKReadOnlyHandlers(
    mcp.HandlerConfig{Timeout: 10 * time.Second},
    gamma.NewClient(""),
    data.NewClient(data.Config{}),
    clob.NewClient(clob.Config{}),
)
server := mcp.NewServerWithHandlers(handlers)
_ = server
```

For market-data snapshots, feed a `marketdata.Tracker` from your websocket
reader and register `mcp.NewMarketDataSnapshotHandler(tracker)` as the
`polygolem.marketdata_snapshot` handler (`pkg/mcp/sdk_handlers.go:67`).

## Update triggers

Refresh this page when:

- `pkg/mcp.SafeTools()` adds, removes, or renames a tool;
- `pkg/openapi.Spec()` adds, removes, or renames a path;
- `cmd/polygolem_mcp` changes transport behavior beyond stdio line JSON-RPC;
- a mutating MCP/OpenAPI surface is proposed — update [SAFETY.md](SAFETY.md) and ADRs first.
