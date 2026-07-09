# CLI Command Model

## What this is

A source-backed summary of how the `polygolem` CLI is wired today. Use this when
renaming commands, adding commands, or debugging generated CLI docs.

## Start here

The root command is built in `NewRootCommand` (`internal/cli/root.go:58`). Its
long help states the default posture: read-only market data needs no
credentials, authenticated paths are opt-in through `SIGNER_PRIVATE_KEY`, and
fund-moving commands use caps plus typed confirmations (`internal/cli/root.go:74`
to `internal/cli/root.go:90`).

## Top-level command groups

`NewRootCommand` adds four visible groups (`internal/cli/root.go:117` to
`internal/cli/root.go:121`):

| Group | Commands wired from source | Purpose |
|---|---|---|
| Getting started & diagnostics | `version`, `doctor`, `ping`, `debug`, `check-upstream`, `risk` (`internal/cli/root.go:130` to `internal/cli/root.go:153`) | Local readiness, API reachability, version, upstream drift, live posture |
| Market data & research | `markets`, `events`, `book`, `prices`, `stream`, `analytics`, `wallets` (`internal/cli/root.go:154` to `internal/cli/root.go:162`) | Credential-free reads and research surfaces |
| Paper trading | `sim` (`internal/cli/root.go:163`) | Local no-risk trade simulation |
| Trading & wallet | `credentials`, `wallet`, `exchange`, `bridge`, `tx`, `builder-keys` (`internal/cli/root.go:165` to `internal/cli/root.go:184`) | Credentialed account reads, wallet lifecycle, trading, relayer and key helpers |

## JSON output contract

All commands receive the root `--json` flag (`internal/cli/root.go:107`). The
central envelope type is `output.Envelope` with `ok`, `version`, `data`,
`error`, and `meta` fields (`internal/output/output.go:31` to
`internal/output/output.go:39`). Success and error envelopes are emitted through
`WriteSuccess` and `WriteErrorEnvelope` (`internal/output/output.go:54` to
`internal/output/output.go:69`).

## Generated command docs

`docs/COMMANDS.md` and the docs-site CLI page are generated from Cobra metadata.
The generator warns not to edit by hand (`internal/cli/docs_generation.go:23` to
`internal/cli/docs_generation.go:31`) and writes command conventions,
environment variables, command tree, and every command section
(`internal/cli/docs_generation.go:32` to `internal/cli/docs_generation.go:47`).

## Important command seams

- `markets categories` is read-only. It lists the curated polymarket.com
  category mapping and, with `--events`, fetches category feeds through Gamma
  `events/keyset` (`internal/cli/cmd_discover.go:132` to
  `internal/cli/cmd_discover.go:163`, `pkg/gamma/categories.go:17` to
  `pkg/gamma/categories.go:35`, `internal/gamma/categories.go:36` to
  `internal/gamma/categories.go:68`).
- `exchange market-order --dry-run` builds a local POLY_1271 preview instead of
  posting an order. The CLI routes dry-run to `PreviewMarketOrder` and the
  preview reports `signature_type`, deposit-wallet maker/signer context, and
  `signature_included=false` (`internal/cli/cmd_clob.go:594` to
  `internal/cli/cmd_clob.go:619`, `internal/clob/orders.go:1021` to
  `internal/clob/orders.go:1059`).
- `exchange simulate` is read-only. It calls a simulation runner and only writes
  JSON; its help explicitly says it does not load a key, sign, or submit an
  order (`internal/cli/cmd_clob_simulate.go:16` to
  `internal/cli/cmd_clob_simulate.go:45`).
- `exchange` owns both read-only CLOB market data and credentialed trading/account
  commands. The help separates read-only, authenticated, and live-money commands
  (`internal/cli/cmd_clob.go:254` to `internal/cli/cmd_clob.go:267`).
- `wallet` owns deposit-wallet lifecycle commands and documents the derive →
  deploy → approve → fund → trade → redeem flow (`internal/cli/deposit_wallet.go:59`
  to `internal/cli/deposit_wallet.go:74`).

## Update triggers

Refresh this page when `internal/cli/root.go`, `internal/cli/cmd_discover.go`,
`internal/cli/cmd_clob*.go`, `internal/cli/deposit_wallet.go`,
`internal/output/output.go`, or `internal/cli/docs_generation.go` changes.
