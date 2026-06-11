# Polygolem MCP and OpenAPI Read-Only Deployment Notes

Polygolem exposes agent-friendly read-only integration surfaces without adding
Python, Node, hosted proxy, or live-trading risk.

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
per-call timeout.

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
`polygolem.marketdata_snapshot` handler.
