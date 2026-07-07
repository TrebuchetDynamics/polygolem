# Polygolem Wiki

## What this is

This wiki is the compact, source-backed map for `polygolem`: a Go module at
`github.com/TrebuchetDynamics/polygolem` (`go.mod:1`) targeting Go 1.25
(`go.mod:3`). It is meant for agents, contributors, operators, and SDK users who
need the current command model and code boundaries without reading every guide.

## Start here

- New to the repo: read [CLI command model](./cli-command-model.md), then
  [SDK package boundary](./sdk-package-boundary.md).
- Near live funds: read [Safety and wallet model](./safety-and-wallet-model.md)
  before running any `wallet` or `exchange` mutation.
- Updating docs or tests: read [Docs and validation](./docs-and-validation.md).

## Page graph

| Page | Use it for | Primary source evidence |
|---|---|---|
| [CLI command model](./cli-command-model.md) | Current top-level commands, JSON envelope, generated command docs | `internal/cli/root.go:71`, `internal/cli/root.go:117`, `internal/cli/docs_generation.go:23` |
| [Safety and wallet model](./safety-and-wallet-model.md) | Deposit-wallet-only live posture, caps, confirmations, simulator boundary | `internal/livegate/livegate.go:1`, `internal/cli/deposit_wallet.go:59`, `internal/cli/cmd_clob.go:254` |
| [SDK package boundary](./sdk-package-boundary.md) | Which `pkg/` packages are public and how docs enforce that inventory | `go.mod:1`, `tests/repository_hygiene_test.go:90`, `README.md` SDK table |
| [Docs and validation](./docs-and-validation.md) | What docs are generated, what tests guard docs, what commands validate changes | `.github/workflows/ci.yml:31`, `internal/cli/docs_generation.go:23`, `tests/cli_rename_e2e_test.go:10` |

## Current command names

The renamed command set is intentionally breaking: `ping`, `markets`, `book`,
`exchange`, `analytics`, `wallets`, `prices`, `stream`, `sim`, `wallet`,
`credentials`, `builder-keys`, `bridge`, `tx`, `risk`, `doctor`, `debug`,
`check-upstream`, `events`, and `version`. The root command groups read-only,
paper, live, and diagnostic surfaces separately (`internal/cli/root.go:117` to
`internal/cli/root.go:184`).

## Update triggers

Refresh this wiki when:

- `internal/cli/root.go` adds/removes/renames a top-level command;
- `internal/cli/docs_generation.go` changes generated command docs;
- `pkg/` gains, removes, or promotes a public package;
- live-money guard logic changes in `internal/livegate` or wallet/exchange CLI
  commands;
- CI or docs-site validation changes under `.github/workflows/ci.yml` or
  `docs/docs-site/package.json`.
