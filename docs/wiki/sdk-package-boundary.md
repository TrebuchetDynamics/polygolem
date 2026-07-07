# SDK Package Boundary

## What this is

A source-backed map of the public Go SDK boundary. Use this when adding,
renaming, documenting, or importing packages under `pkg/`.

## Start here

The module path is `github.com/TrebuchetDynamics/polygolem` (`go.mod:1`) and the
repo targets Go 1.25 (`go.mod:3`). Public SDK packages live under `pkg/` and are
expected to be importable by external consumers.

## Public package inventory

Current top-level public packages under `pkg/`:

`bridge`, `builder`, `capabilities`, `clob`, `compat`, `contracts`,
`cryptoprice`, `ctf`, `data`, `enabletrading`, `experimental`, `funding`,
`gamma`, `geoblock`, `intel`, `marketdata`, `marketresolver`, `mcp`, `openapi`,
`orderbook`, `orderfills`, `orderresults`, `pagination`, `plugins`,
`polyerrors`, `reconciliation`, `relayer`, `rfq`, `rtds`, `settlement`,
`signers`, `stream`, `types`, `universal`, `upstreamdrift`, and `wallet`.

## Documentation contract

The repo has an explicit package-inventory test. For every public package,
`TestPublicPackageInventoryIsDocumented` requires:

- a README link in the form ``[`pkg/name`](pkg/name)``;
- an architecture row in `docs/ARCHITECTURE.md`;
- a docs-site SDK heading in `docs/docs-site/src/content/docs/docs/reference/sdk.mdx`.

Source: `tests/repository_hygiene_test.go:90` to
`tests/repository_hygiene_test.go:110`.

## Import fixture contract

The same test file checks a public SDK compile fixture imports every public
package (`tests/repository_hygiene_test.go:112` to
`tests/repository_hygiene_test.go:115`). This prevents an undocumented package
from silently entering the public boundary.

## Practical package routes

| Need | Start with |
|---|---|
| One client across Gamma/CLOB/Data/Stream | `pkg/universal` |
| CLOB orders/books/account reads | `pkg/clob`, `pkg/orderbook`, `pkg/orderfills` |
| Market discovery and crypto windows | `pkg/gamma`, `pkg/marketresolver`, `pkg/cryptoprice` |
| Public analytics and wallet intelligence | `pkg/data`, `pkg/intel`, `pkg/orderresults`, `pkg/reconciliation` |
| Deposit-wallet and relayer flows | `pkg/wallet`, `pkg/relayer`, `pkg/contracts`, `pkg/funding`, `pkg/settlement`, `pkg/enabletrading` |
| Tooling/agent surfaces | `pkg/mcp`, `pkg/openapi`, `pkg/capabilities`, `pkg/compat`, `pkg/upstreamdrift` |

## Update triggers

Refresh this page when:

- a top-level `pkg/` directory is added, removed, or promoted;
- README SDK table changes;
- `docs/ARCHITECTURE.md` package inventory changes;
- `docs/docs-site/src/content/docs/docs/reference/sdk.mdx` changes;
- `tests/repository_hygiene_test.go` changes its public boundary checks.
