# Public SDK boundary

Stable downstream integrations should use `pkg/` packages. `internal/` packages remain implementation details for the CLI and may change without SDK compatibility guarantees.

Stable user-directed order transactions live in `pkg/clob`; wallet identity in `pkg/wallet`; contract addresses and approval capability sets in `pkg/contracts`; relayer onboarding in `pkg/relayer`; settlement readiness in `pkg/settlement`; read-only market data in `pkg/gamma`, `pkg/data`, `pkg/orderbook`, `pkg/stream`, and related DTO packages.

`pkg/experimental/` is importable but not SDK-stable. Experimental helpers may validate or shape DTOs, but they must not replace stable Polymarket API interfaces or introduce strategy decisions inside Polygolem.
