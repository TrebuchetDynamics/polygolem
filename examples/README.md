# Polygolem Examples

These are copyable starting points, not production strategies.

| Example | Purpose | Risk |
|---|---|---|
| `read-only-market-monitor.sh` | Poll public crypto market discovery with no credentials. | Read-only |
| `paper-strategy.sh` | Reset a paper account, trade one BTC 5m side, and print positions. | Local-only paper state |
| `bot-basic/main.go` | Minimal Go SDK bot skeleton that lists active markets. | Read-only |
| `tradegate-bot/main.go` | Shows `TradeGate` blocking an order before signing/submission. | No live submission |

Run shell examples from the repository root. Set `POLYGOLEM_BIN=/path/to/polygolem` to use an installed binary; otherwise they use `go run ./cmd/polygolem --`.
