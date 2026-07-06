# Safety

Polygolem is read-only by default and moves real funds only through explicit,
credential-gated commands. This page describes the guards that actually exist in
the shipped CLI. It is not a description of a future state; where a guard is
partial today, this page says so.

## Read-Only Default

Read-only is the default. Market data, discovery, streaming, orderbook reads,
data analytics, health checks, and diagnostics require no wallet credentials,
sign no payloads, and submit no mutations. Authenticated paths are opt-in only
when `SIGNER_PRIVATE_KEY` (legacy `POLYMARKET_PRIVATE_KEY`) is set.

## Paper Mode

Paper mode is local-only. Simulated buys, sells, positions, and reset operations
are stored in local state and never call authenticated trading endpoints or
on-chain paths. Paper mode may read public market data to price simulations.

## What can move funds

Only these command groups have live execution paths:

- `clob create-order`, `clob market-order`, `clob batch-orders` — sign and post
  CLOB V2 orders.
- `deposit-wallet deploy`, `approve`, `approve-adapters`, `batch`, `fund`,
  `onboard`, `redeem` — relayer-sponsored or direct on-chain transactions.

Everything else is read-only. Bridge withdrawals, RFQ submission, and CTF
split/merge are typed dry-run surfaces that return an explicit unsupported error
before any live call (see the Safety-First Mutating Surface term in
[CONTEXT.md](../CONTEXT.md)).

## Live-order cap

Every order-placing command enforces a notional cap before signing:
`POLYGOLEM_MAX_LIVE_ORDER_USD` (default **$1**). `create-order` and
`market-order` check the single order; `batch-orders` checks each order and the
summed batch notional. An order over the cap is rejected before the private key
is loaded. Raise the cap deliberately with the environment variable; there is no
flag that bypasses it.

## Typed live-money confirmation

Every deposit-wallet command that signs and submits real transactions requires a
typed confirmation token, so a live submission cannot happen from a single
mistyped flag:

| Command | Required token |
|---|---|
| `deposit-wallet approve-adapters` (with `--submit`) | `--confirm APPROVE_ADAPTERS` |
| `deposit-wallet redeem` (with `--submit`) | `--confirm REDEEM_WINNERS` |
| `deposit-wallet approve` (with `--submit`) | `--confirm APPROVE_TRADING` |
| `deposit-wallet batch` | `--confirm SUBMIT_BATCH` |
| `deposit-wallet onboard` | `--confirm ONBOARD_WALLET` |

The dry-run-capable commands (`approve`, `approve-adapters`, `redeem`) print
calldata for review when run without `--submit`. The confirmation token is
checked before the private key is loaded.

## Preflight and readiness

Preflight and live-readiness checks aggregate config validity, wallet readiness,
auth readiness, API reachability, and chain consistency into one pass/fail
result. Automation must treat any preflight failure as terminal and stop rather
than retrying a different mode or assuming a partial result is usable.

The `polygolem live status` command evaluates advisory gates
(`POLYMARKET_LIVE_PROFILE`, `--confirm-live`, preflight) and reports whether an
operator has opted into a live posture. This status is **advisory**: it helps an
operator confirm intent, but the enforced money guards are the live-order cap and
the typed confirmation tokens above, not the `live status` flags.

## Failure behavior

Live commands abort on failure with a structured error and non-zero exit code.
The CLI never silently downgrades to paper or read-only mode, because that would
hide operator intent and make automation unsafe.

## Credential handling

Read-only and paper workflows require no private key. The private key is read
from the environment only, is never persisted or logged, and is printed only by
`auth export-key`, which is double-confirmed. Diagnostics (`diag`) and config
loading redact secrets; relayer and builder credentials are stored `0600` and
never emitted in JSON output.

## Deposit Wallet Safety

Polymarket requires deposit wallet (POLY_1271 / signature type 3) for all
trading. EOA, proxy, and Gnosis Safe are blocked by CLOB V2. This
introduces safety rules beyond the direct EOA model.

### Signer vs Funder Separation

The EOA remains the cryptographic signing key for the ERC-7739 wrapper. The
deposit wallet is the CLOB account: it holds pUSD and appears as both order
`maker` and order `signer` in the V2 payload. These must never be confused:

- **EOA pUSD does NOT fund deposit-wallet orders.** CLOB reads the deposit wallet's balance.
- **Approvals must come from the deposit wallet** via relayer WALLET batch, not from the EOA.
- **Diagnostics must distinguish** `owner_eoa` from `funder/deposit_wallet` in logs and audit records.

### Builder Credential Isolation

V2 relayer keys (`RELAYER_API_KEY` / `RELAYER_API_KEY_ADDRESS`) and legacy
builder HMAC credentials (`BUILDER_API_KEY/SECRET/PASSPHRASE`) authenticate
with the relayer for WALLET-CREATE and WALLET batch operations. These are a
separate auth system and must never be:

- Reused as CLOB L2 credentials
- Stored alongside CLOB API keys in the same config section
- Added to order-signing headers (orders use `builderCode`, not builder HMAC)

### Relayer Auth vs Trading Auth

- **Relayer auth**: V2 relayer key or builder HMAC credentials → used for wallet lifecycle operations
- **Trading auth**: CLOB L1/L2 credentials → used for order placement and balance queries
- These systems are independent. A failed relayer call must not be retried as a CLOB call.

### Deposit-Wallet Balance Routing

When `signature_type = 3` (deposit), the CLOB balance endpoint returns the
**deposit wallet's pUSD balance**, not the EOA's. Live readiness must:

1. Check the CLOB balance with `signature_type = 3`
2. Verify the deposit wallet is deployed. Polygon `eth_getCode` is the source
   of truth; relayer `/deployed` is advisory and can return false while the
   contract already has bytecode.
3. Verify collateral allowances are non-zero
4. Block before order submission if any check fails

See [DEPOSIT-WALLET-MIGRATION.md](./DEPOSIT-WALLET-MIGRATION.md) for the full onboarding flow,
common pitfalls, and recovery steps.

## Deposit Wallet Safety Rules

The May 2026 deposit-wallet migration introduces a new family of commands
(`polygolem deposit-wallet *`) that perform on-chain or relayer-bound
operations. These rules apply.

1. **Builder credentials are required and redacted.** `--deploy`,
   `--batch`, `--approve --submit`, and `--onboard` require
   `POLYMARKET_BUILDER_API_KEY`, `POLYMARKET_BUILDER_SECRET`, and
   `POLYMARKET_BUILDER_PASSPHRASE`. Configuration loading redacts all three
   on every load; no command emits them in JSON output.

2. **Read-only deposit-wallet commands stay read-only.**
   `deposit-wallet derive`, `deposit-wallet status`, and
   `deposit-wallet nonce` perform no on-chain or relayer mutations.

3. **Batch signing requires explicit calldata input.**
   `deposit-wallet batch --calls-json` requires structured input. The CLI
   does not synthesize calls. The `approve` shortcut shows calldata before
   submission unless `--submit` is passed.

4. **Funding moves real money.** `deposit-wallet fund --amount X`
   transfers ERC-20 pUSD from the EOA to the deposit wallet via direct RPC.
   The amount must be specified explicitly. There is no default.

5. **Onboarding is the only multi-step composite.**
   `deposit-wallet onboard --fund-amount X` performs deploy → approve →
   fund. Each step is gated; failure of any step aborts the composite and
   leaves the wallet in a recoverable state visible to
   `deposit-wallet status`.

6. **POLY_1271 orders use the deployed wallet's signature path.**
   `clob create-order` and
   `clob market-order` sign with the deposit
   wallet's POLY_1271 path. Orders signed without the deposit signature
   type after the May 2026 cutoff will be rejected by Polymarket for
   new accounts. Readiness must verify non-empty bytecode at the deposit
   wallet address before order submission.

7. **Builder attribution does not bypass safety.** Setting builder
   credentials enables deposit-wallet operations; it does not relax any
   gate or grant trading privileges.

8. **Decision-window safety.** External applications that automate trading
   decisions must bind each decision to the selected market window. A signal
   for `2026-05-09T08:20:00Z` must not buy a market that starts at
   `2026-05-09T12:20:00Z`, even when the asset and timeframe match. Polygolem
   provides the strict window resolver that returns a `window_mismatch` status
   instead of silently falling back to a future market; it does not create the
   trading signal.

## Matched, Winning, And Redeemable

Matched, winning, and redeemable are separate states:

- `matched`: the CLOB filled the order and transferred or minted position
  shares.
- `winning`: the market resolved and the held token is the winning outcome.
- `redeemable`: Polymarket's Data API reports the held position can be
  redeemed.

A matched order is not proof that the market won. Redemption automation must
read the deposit wallet's Data API `/positions` rows and use
`redeemable=true` as the readiness signal. The current position schema exposes
`redeemable`, `mergeable`, `negativeRisk`, `outcome`, `outcomeIndex`,
`oppositeOutcome`, `oppositeAsset`, and `endDate`; it does not expose a
separate `resolved` boolean.

## V2 Redeem Readiness

Polymarket V2 uses collateral adapter contracts for pUSD-native CTF actions.
For deposit-wallet positions this path is non-negotiable: the owner signs an
EIP-712 WALLET batch, the relayer submits it through the deposit-wallet
factory, and the wallet call targets:

- `CtfCollateralAdapter` for standard binary markets:
  `0xAdA100Db00Ca00073811820692005400218FcE1f`
- `NegRiskCtfCollateralAdapter` for negative-risk markets:
  `0xadA2005600Dec949baf300f4C6120000bDB6eAab`

Do not call `ConditionalTokens.redeemPositions` directly from a V2
deposit-wallet flow. The adapter reads `conditionId`, detects the wallet's
current CTF balances, redeems through the underlying CTF path, wraps proceeds
back into pUSD, and returns pUSD to the deposit wallet.

SAFE and PROXY examples in upstream relayer clients are not deposit-wallet
precedent. Deposit wallets use `executeDepositWalletBatch(...)` / relayer
`WALLET` transactions, not the SAFE/PROXY `execute(...)` shortcut.

Adapter readiness is distinct from trading readiness. The existing trading
approval batch covers CLOB exchange spenders. V2 redeem requires the deposit
wallet to approve the collateral adapters with CTF `setApprovalForAll`; the
one-time adapter approval batch should also include pUSD `approve` for future
split support. Existing live wallets that only ran the trading approval batch
need a one-shot adapter-approval migration before their first V2 redeem.

The first-class `polygolem deposit-wallet approve-adapters`, `redeemable`,
and `redeem` commands build the V2 adapter path (commits `c77e735` and
`0593991`). Every signing path defaults to dry-run; submission requires both
`--submit` and a typed `--confirm` token (`APPROVE_ADAPTERS` for adapter
approvals, `REDEEM_WINNERS` for redeem). The redeem command runs an
`isApprovedForAll(wallet, adapter)` pre-check via `eth_call` and refuses to
sign if any approval is missing — the relayer never sees `/submit` when the
pre-check fails.

If the relayer rejects adapter approval or redeem calls as "not in the allowed
list", first verify the adapter constants against Polymarket's current
contracts reference. The 2026-05-09 live recovery proved that stale adapter
addresses can produce this exact symptom. If the addresses match the official
reference and the relayer still rejects the batch, stop and treat it as an
upstream allowlist blocker. The production `DepositWalletFactory.deploy()` and
`proxy()` entrypoints are `onlyOperator`, so the owner EOA cannot bypass the
relayer, and raw `ConditionalTokens.redeemPositions` is not a V2
deposit-wallet fallback.
