# Threat Model

Short checklist for operating polygolem with funds. This is not legal or financial advice.

## Assets to protect

| Asset | Risk | Guardrail |
|---|---|---|
| `POLYMARKET_PRIVATE_KEY` | Full wallet compromise | Keep local, never paste into tickets/logs, use `diag` redaction, avoid `auth export-key` unless importing into a temporary browser wallet. |
| CLOB L2 API key/secret/passphrase | Authenticated account actions | Store outside source control, redact output, rotate if exposed. |
| Relayer V2 key / builder credentials | Deposit-wallet deploy/batch authority | Treat as separate from CLOB auth; do not reuse or print. |
| Deposit wallet pUSD and CTF positions | Direct trading funds | Verify deposit-wallet balance, not EOA balance. |
| Approvals | Unexpected token movement through approved contracts | Approve only documented Polymarket contracts; re-check spender addresses before live batches. |
| Token IDs / condition IDs | Buying the wrong market/outcome | Resolve current market window and token IDs immediately before order placement. |
| Market data freshness | Trading stale prices/windows | Bind decisions to a timestamped window; stop on mismatch. |

## Operator checklist before live orders

1. Run read-only discovery and `diag` first.
2. Run the paper strategy with the same asset/interval/side.
3. Confirm the deposit wallet is deployed and funded.
4. Confirm enable-trading and CLOB auth readiness.
5. Confirm token ID, outcome, condition ID, and market window.
6. Calculate worst-case spend and compare it to a hard cap.
7. Prefer a tiny `--post-only` limit order before any taking order.
8. Save tx hashes, order IDs, and JSON envelopes for audit.

## Things polygolem should not do

- Print private keys except the explicit high-risk `auth export-key` flow.
- Require credentials for read-only commands.
- Fall back from a failed live path into a different wallet/signature mode.
- Treat EOA pUSD as deposit-wallet trading balance.
- Continue after upstream protocol, network, or chain readiness failures.
