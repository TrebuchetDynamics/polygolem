# Boring Paper Bot

A deliberately boring read-only + paper bot example. It resolves the current crypto market, reads UP/DOWN CLOB buy prices, chooses the cheaper side when it is below a max price, and prints one JSON decision. It never signs or submits an order.

```bash
go run ./examples/boring-paper-bot
```

Optional knobs:

```bash
POLYGOLEM_BORING_ASSET=BTC \
POLYGOLEM_BORING_INTERVAL=5m \
POLYGOLEM_BORING_SIZE=1 \
POLYGOLEM_BORING_MAX_PRICE=0.55 \
go run ./examples/boring-paper-bot >> boring-paper.jsonl
```

Run it from cron/systemd for a week to build an operator-visible paper trail before funding a live wallet.
