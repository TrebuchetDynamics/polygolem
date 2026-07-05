# Polygolem Context

Polygolem is a Go SDK and CLI for Polymarket protocol access, read-only research, paper workflows, and gated deposit-wallet operations. This context defines project-specific language that keeps protocol safety, wallet identity, and research intelligence unambiguous.

## Language

### Wallet Intelligence

**Wallet Intelligence Signal**:
A reproducible, read-only statistical candidate about public wallet or market activity. It is not trading advice and not a finding of misconduct.
_Avoid_: Insider proof, fraud finding, manipulation finding, trade recommendation

**Formula Version**:
A named scoring formula contract such as `wallet_score_v1` or a future independent alert formula. Dossier Alerts reuse `wallet_score_v1` because they wrap Wallet Dossier scores rather than introducing separate alert scoring semantics.
_Avoid_: Magic score, current algorithm, hidden model

**Wallet Dossier Source Authority**:
The Polygolem-owned source adapter that a wallet dossier treats as authoritative for a derived field. In V1, Data API closed-position rows are authoritative for realized PnL and win/loss counts; disagreements from trade/activity reconstruction make the dossier conflicted or partial instead of silently replacing the value.
_Avoid_: Scraped competitor score, blended truth, silent correction

**Conflicted Dossier**:
A wallet dossier whose Polygolem source adapters disagree about a derived wallet metric. A conflicted dossier must expose the conflict instead of producing a single unqualified score.
_Avoid_: Clean score, fixed result, ignored mismatch

**Batch Intelligence Alert**:
A Wallet Intelligence Signal produced from bounded Polygolem read-adapter queries, such as Data API trades, activity, holders, or closed positions. Batch intelligence alerts are reproducible, fixture-testable, and CLI-friendly.
_Avoid_: Live tape, push alert, stream alert

**Dossier Alert**:
A V1 Batch Intelligence Alert emitted when one user-scoped Wallet Dossier score passes a configured threshold. It is not a global recent-trade feed.
_Avoid_: Market-wide alert, live alert, trade tape

**Stream Intelligence Alert**:
A future Wallet Intelligence Signal produced from WebSocket lifecycle data. It is deferred until batch alert semantics, formula versions, and source authority rules are proven.
_Avoid_: V1 alert, simple alert, batch alert

**Neutral Intelligence Prior**:
The source-free default prior used by pure wallet-intelligence formulas when no caller-provided baseline exists. For `shrinkage_win_rate_v1`, the neutral prior is 10 wins over 20 bets, representing a 50% baseline.
_Avoid_: Category prior, global live prior, hidden default

**Co-Positioning Signal**:
A Wallet Intelligence Signal indicating that public Data API trades or activity show wallets repeatedly taking related positions. It is a candidate research signal, not proof of coordination or common funding.
_Avoid_: Funding cluster, coordinated ring, shared wallet owner

**Funding Cluster**:
A future wallet-intelligence concept backed by reproducible Polygon transfer or indexer evidence that multiple wallets share a funding relationship. Funding clusters are out of scope for V1 until Polygolem owns a dedicated source adapter for them.
_Avoid_: Co-positioning signal, inferred cluster, visual guess

### CLOB Auth & Orders

**Polymarket API Interface**:
Polygolem is an interface into Polymarket APIs and contracts. It has first-class support for wallet setup, approvals, signing primitives, and user-directed order transactions. It does not choose markets, sides, prices, sizes, timing, or whether to trade.
_Avoid_: Bot, strategy engine, autonomous trader, decision maker, trading decisions

**EOA-Bound CLOB Auth**:
Polygolem's validated CLOB L1/L2 authentication path: `POLY_ADDRESS` is the EOA and `POLY_SIGNATURE` is a standard 65-byte EOA ECDSA signature. The deposit wallet identity is carried by the order payload (`maker`, `signer`, `signatureType=3`), not by ClobAuth headers.
_Avoid_: Deposit-wallet-owned API key, deposit-wallet-bound ClobAuth, ERC-7739 wrapped auth

**POLY_1271 Order Signing**:
The sigtype-3 signing path where order `maker` and `signer` are the deposit wallet, `signatureType` is 3, and the `signature` is an ERC-7739 TypedDataSign wrapper of an EOA-signed V2 Order struct.
_Avoid_: Sigtype-0 signing, EOA maker signing

**ERC-7739 Wrapped Order Signature**:
The 636-hex-char order signature format: `innerSig(65) || appDomainSep(32) || contents(32) || contentsType(186) || uint16BE(186)`. Required for all POLY_1271 orders to pass on-chain `isValidSignature` verification on the deposit wallet.
_Avoid_: Raw ECDSA order signature, 65-byte order sig

**Safety-First Mutating Surface**:
Any operation that could move funds or mutate protocol state must ship through a dry-run / unsupported-submit pattern before live implementation. Bridge withdrawals, RFQ, and CTF split/merge each enter as typed DTOs with validation and an explicit guard error. Only the CLI `deposit-wallet` and `clob create-order` groups have live execution paths.
_Avoid_: Ungated mutating endpoints, live-first implementation

**Read-Only by Default**:
Polygolem requires no credentials for market data, discovery, streaming, orderbook reads, data analytics, health checks, and diagnostics. All authenticated paths are opt-in through `SIGNER_PRIVATE_KEY` (legacy `POLYMARKET_PRIVATE_KEY` is accepted as a fallback).
_Avoid_: Authenticated-default mode, credential-required discovery, POLYMARKET_PRIVATE_KEY as the primary credential name

**Capability Map**:
The typed `pkg/capabilities` metadata describing each Polymarket surface Polygolem exposes: service, auth requirement, wallet mode, and read-only/mutating classification. It is the single source of truth for surface/auth/wallet-mode/version claims; `docs/COMPATIBILITY.md` is generated from it rather than hand-written.
_Avoid_: Feature flags, compatibility table (for the package itself), hand-maintained surface list, capability meaning CLI permissions

## Example dialogue — Wallet Intelligence

Developer: "Should this high score call the wallet an insider?"
Domain expert: "No. It is a Wallet Intelligence Signal. Show the Formula Version, source rows, reasons, and the disclaimer that it is not a misconduct finding."

Developer: "Trades reconstruction disagrees with Data API closed positions. Which value do we score?"
Domain expert: "Use the Wallet Dossier Source Authority. For V1, keep Data API closed positions as the PnL/win-loss source and mark the dossier as conflicted."

Developer: "Should alerts subscribe to the CLOB WebSocket immediately?"
Domain expert: "No. V1 uses Dossier Alerts from Polygolem read adapters; Stream Intelligence Alerts come later."

Developer: "What prior does `shrinkage_win_rate_v1` use before category baselines exist?"
Domain expert: "Use the Neutral Intelligence Prior and expose the prior values in raw metrics."

Developer: "Can V1 show clusters like CrowdIntel?"
Domain expert: "No. V1 may show a Co-Positioning Signal from Polygolem read adapters, but Funding Clusters wait for reproducible transfer/indexer evidence."

## Example dialogue — CLOB Auth

Developer: "Does L1 ClobAuth need the ERC-7739 wrapper for deposit wallets?"
Domain expert: "No. CLOB L1/L2 auth is EOA-bound. The ERC-7739 wrapper is only for the order signature. The deposit wallet identity rides on `maker`, `signer`, and `signatureType=3` in the order payload."

Developer: "Can we create a deposit-wallet-bound API key headlessly?"
Domain expert: "No — CLOB auth is EOA-bound. We don't need one. The API key's `POLY_ADDRESS` is the EOA. Polymarket's HTTP layer authenticates the EOA; the deposit wallet validates orders on-chain via ERC-1271."

Developer: "Should we guard new mutating surfaces behind a compile-time error?"
Domain expert: "Yes. Every new mutating surface follows the Safety-First pattern: typed DTOs, validation, dry-run preview, and an explicit `Err*Unsupported` error before live implementation."
