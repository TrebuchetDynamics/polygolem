# Polygolem Context

Polygolem is a Go SDK and CLI for Polymarket protocol access, read-only research, paper workflows, and gated deposit-wallet operations. This context defines project-specific language that keeps protocol safety, wallet identity, and research intelligence unambiguous.

## Language

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

## Example dialogue

Developer: “Should this high score call the wallet an insider?”
Domain expert: “No. It is a Wallet Intelligence Signal. Show the Formula Version, source rows, reasons, and the disclaimer that it is not a misconduct finding.”

Developer: “Trades reconstruction disagrees with Data API closed positions. Which value do we score?”
Domain expert: “Use the Wallet Dossier Source Authority. For V1, keep Data API closed positions as the PnL/win-loss source and mark the dossier as conflicted.”

Developer: “Should alerts subscribe to the CLOB WebSocket immediately?”
Domain expert: “No. V1 uses Dossier Alerts from Polygolem read adapters; Stream Intelligence Alerts come later.”

Developer: “What prior does `shrinkage_win_rate_v1` use before category baselines exist?”
Domain expert: “Use the Neutral Intelligence Prior and expose the prior values in raw metrics.”

Developer: “Can V1 show clusters like CrowdIntel?”
Domain expert: “No. V1 may show a Co-Positioning Signal from Polygolem read adapters, but Funding Clusters wait for reproducible transfer/indexer evidence.”
