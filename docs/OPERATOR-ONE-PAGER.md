# Operator One-Pager

If you only read one page before trying polygolem, read this one.

## No-wallet path

```bash
polygolem health --json
polygolem discover crypto-5m --enrich --json
polygolem paper reset --cash 100 --json
polygolem paper trade --asset BTC --interval 5m --side up --size 1 --json
polygolem paper positions --json
```

This path should not need `SIGNER_PRIVATE_KEY`, CLOB credentials, relayer credentials, or Polygon RPC signing.

## Before live money

1. Set a tiny cap:

   ```bash
   export POLYGOLEM_MAX_LIVE_ORDER_USD=1
   ```

2. Verify identity and readiness:

   ```bash
   polygolem diag --json
   polygolem auth status --check-deposit-key --json
   polygolem deposit-wallet status --check-enable-trading --json
   polygolem clob update-balance --asset-type collateral --json
   ```

3. Verify the exact market:

   ```bash
   polygolem discover crypto-window --asset BTC --interval 5m --enrich --json
   polygolem clob book --token <TOKEN_ID> --json
   ```

4. Start with a maker-only order you can afford to lose:

   ```bash
   polygolem clob create-order \
     --token <TOKEN_ID> \
     --side buy \
     --price 0.01 \
     --size 1 \
     --post-only \
     --json
   ```

## Stop immediately if

- token ID, condition ID, market window, or outcome is ambiguous;
- any readiness command fails;
- the order would exceed `POLYGOLEM_MAX_LIVE_ORDER_USD`;
- CLOB/Gamma/Data/Relayer returns a protocol, network, or chain error;
- you are unsure whether a balance belongs to the EOA or deposit wallet.

## What to read next

- [SAFE-HAPPY-PATH.md](SAFE-HAPPY-PATH.md) for the full cautious flow.
- [THREAT-MODEL.md](THREAT-MODEL.md) for private-key/API-key/approval risks.
- [ONBOARDING.md](ONBOARDING.md) for deposit-wallet setup.
