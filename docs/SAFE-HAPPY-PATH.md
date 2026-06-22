# Safe Happy Path

This is the smallest operator path from zero credentials to a tiny capped live order. Stop at any failed check; do not skip ahead.

For an automated read-only smoke check, run `scripts/live-smoke.sh`. It does not place orders unless `POLYGOLEM_SMOKE_LIVE_ORDER=1` and the live-order variables are set.

## 1. Read-only health

```bash
polygolem health --json
polygolem diag --json
```

Expected: Gamma and CLOB are reachable, endpoints are the ones you intended, and no private key is required.

## 2. Discover a market

```bash
polygolem discover crypto-5m --enrich --json
polygolem discover crypto-window --asset BTC --interval 5m --enrich --json
```

Record the event slug, condition ID, token IDs, prices, spread, and market window. If the window does not match your strategy decision time, stop.

## 3. Paper trade first

```bash
polygolem paper reset --cash 100 --json
polygolem paper trade --asset BTC --interval 5m --side up --size 1 --json
polygolem paper positions --json
```

Expected: local-only state changes. No private key, signing, CLOB auth, relayer call, or chain transaction should be needed.

## 4. Live readiness check

Only after paper behavior looks right:

```bash
export POLYMARKET_PRIVATE_KEY="0x..."
polygolem auth status --check-deposit-key --json
polygolem deposit-wallet status --check-enable-trading --json
polygolem clob update-balance --asset-type collateral --json
```

Expected: deposit wallet is derived/deployed, CLOB credentials are available, enable-trading approvals are ready, and the deposit wallet has enough pUSD. The EOA balance is not the trading balance.

## 5. Tiny capped live order

Use a tiny amount you are willing to lose. Prefer a maker-only order first so an accidental market order cannot immediately take liquidity:

```bash
export POLYGOLEM_MAX_LIVE_ORDER_USD=1

polygolem clob create-order \
  --token <TOKEN_ID> \
  --side buy \
  --price 0.01 \
  --size 1 \
  --post-only \
  --json
```

Then immediately inspect and cancel if the order is live:

```bash
polygolem clob orders --json
polygolem clob cancel --order-id <ORDER_ID> --json
```

## Stop conditions

Stop instead of retrying in a different mode when:

- the market window, condition ID, or token ID is ambiguous;
- `diag`, `auth status`, `deposit-wallet status`, or `update-balance` fails;
- the deposit wallet is not deployed or has no pUSD;
- CLOB credentials or relayer credentials are missing;
- upstream returns a protocol, network, or chain error envelope;
- the calculated worst-case spend is above your cap.
