# Architecture Decision Records

ADRs record project-level decisions that should guide future code and documentation changes.

| ADR | Decision |
|---|---|
| [0001-wallet-intelligence-v1-boundary.md](0001-wallet-intelligence-v1-boundary.md) | Wallet intelligence V1 uses explicit formula versions and defers global stream alerts/funding clusters. |
| [0002-polymarket-api-interface-boundary.md](0002-polymarket-api-interface-boundary.md) | Polygolem is a Polymarket API/contract interface, not a bot or strategy engine. |
| [0003-deposit-wallet-only-trading.md](0003-deposit-wallet-only-trading.md) | Production trading uses deposit wallets / POLY_1271 only. |
| [0004-public-sdk-boundary.md](0004-public-sdk-boundary.md) | Stable downstream integrations use `pkg/`; `internal/` and `pkg/experimental/` have narrower guarantees. |
