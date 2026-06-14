# CLOB Trade Gate — Design

**Date:** 2026-06-13
**Status:** Approved design, pending implementation plan
**Author:** brainstorming session

## Problem

`internal/risk` ships a fully-built, tested circuit breaker (`Breaker`: daily-loss
limit, per-market/total position limits, consecutive-error halt, manual halt) but
it is imported **nowhere** outside its own tests. Nothing on the order path consults
it, so a bot built on the polygolem SDK has no way to make order submission respect
a halt decision without manually checking the breaker at every call site.

polygolem is an **SDK that real trading bots build on** — not a bot itself. It must
not impose an opinionated risk policy on every API call. The goal is therefore to
let a bot *opt in*: attach a halt gate to the CLOB order client so that order
submission refuses to fire while the bot has halted trading. Everything else —
limit math, what counts as an error, loss/position accounting, persistence — stays
the bot's responsibility.

## Scope

### In scope
- An optional, opt-in **halt gate** on the CLOB order client.
- The three order-creation methods (`CreateLimitOrder`, `CreateMarketOrder`,
  `CreateBatchOrders`) consult the gate **before** signing or sending, and refuse
  to submit when the gate reports trading is halted.
- Public SDK exposure via `pkg/clob` so bots can attach a gate through `Config`.
- A detectable sentinel error so bots can branch on "blocked by gate".

### Explicitly out of scope (deliberate decisions)
- **No limit enforcement in the SDK.** `MaxOrderUSD`, `MaxOpenOrders`, daily-loss,
  and position limits are the bot's to compute and enforce. The SDK order call does
  not know realized PnL or net positions.
- **No automatic `RecordError`/`RecordSuccess`/`RecordLoss`/`RecordPosition`.** The
  bot decides what trips/resets its breaker.
- **No disk persistence.** Breaker state lives in the bot's process; lifecycle is
  the bot's concern.
- **No `pkg/universal` change** in this iteration. It can thread the gate through in
  a follow-up once this lands.
- **Cancellation is never gated.** A halted bot must always be able to cancel orders
  to *reduce* exposure.

## Design

### 1. `internal/clob` — the gate

- New interface (one method, defined in `clob` so the package takes no dependency on
  `internal/risk`):
  ```go
  // TradeGate decides whether new orders may be submitted. A nil gate means
  // "always allowed". internal/risk.Breaker satisfies this interface.
  type TradeGate interface {
      CanProceed() bool
  }
  ```
- New sentinel error:
  ```go
  // ErrTradingHalted is returned by order-creation methods when an attached
  // TradeGate reports that trading is halted. Detect with errors.Is.
  var ErrTradingHalted = errors.New("clob: trading halted by risk gate")
  ```
- `Client` gains an unexported field `gate TradeGate`.
- Construction stays backward compatible by making options variadic:
  ```go
  type Option func(*Client)

  func WithTradeGate(g TradeGate) Option // sets c.gate = g

  func NewClient(baseURL string, tc *transport.Client, opts ...Option) *Client
  ```
  All 16 existing two-argument `NewClient(baseURL, tc)` call sites continue to
  compile unchanged.
- Private helper:
  ```go
  func (c *Client) ensureCanTrade() error {
      if c.gate != nil && !c.gate.CanProceed() {
          return ErrTradingHalted
      }
      return nil
  }
  ```
- Call `ensureCanTrade()` as the **first statement** of `CreateLimitOrder`,
  `CreateMarketOrder`, and `CreateBatchOrders` — before key derivation, signing,
  tick-size lookups, or any network call. On halt, return the error immediately
  having signed and sent nothing.
- Do **not** call it in `CancelOrder`, `CancelOrders`, `CancelAll`, or any read
  method.

### 2. `pkg/clob` — public SDK exposure

- Re-export the gate type and error so bots import only the public package:
  ```go
  type TradeGate = internalclob.TradeGate
  var ErrTradingHalted = internalclob.ErrTradingHalted
  ```
- Add a field to `Config`:
  ```go
  // TradeGate, when set, blocks order submission while it reports trading is
  // halted (CanProceed()==false). Cancellation is never blocked. Nil = no gate.
  TradeGate TradeGate
  ```
- In `NewClient`, when `cfg.TradeGate != nil`, construct the inner client with
  `internalclob.WithTradeGate(cfg.TradeGate)`.

### 3. Behavior contract

| Condition | Behavior |
|---|---|
| No gate attached (default; the CLI) | Byte-for-byte current behavior. |
| Gate attached, `CanProceed()==true` | Order proceeds normally. |
| Gate attached, halted | Create method returns `ErrTradingHalted` immediately; nothing signed, nothing sent. |
| Any cancel method, gate halted | Proceeds normally (risk-reduction always allowed). |

The bot owns all limit math and trips/resets the breaker via the existing
`RecordError` / `RecordLoss` / `RecordPosition` / `Halt` / `Reset` methods.

### 4. Consumer usage sketch

```go
breaker := risk.NewBreaker(risk.DefaultPolicy()) // satisfies clob.TradeGate
c := clob.NewClient(clob.Config{BaseURL: url, TradeGate: breaker})

// bot logic decides to halt:
breaker.Halt()

_, err := c.CreateLimitOrder(ctx, pk, params)
if errors.Is(err, clob.ErrTradingHalted) {
    // expected: submission was refused
}
```

## Error handling

`ensureCanTrade` returns the `ErrTradingHalted` sentinel (or an error wrapping it
with `%w`) so callers can use `errors.Is(err, clob.ErrTradingHalted)`. No other
error semantics change.

## Testing

`internal/clob`:
- Gate returns false → each of `CreateLimitOrder`, `CreateMarketOrder`,
  `CreateBatchOrders` returns `ErrTradingHalted` and **never** contacts the server
  (httptest handler calls `t.Fatal` if `/order` or `/orders` is hit).
- Gate returns true → normal flow (order posted).
- Cancel methods succeed while the gate is halted.
- Nil gate → existing order tests remain green (no signature/behavior change).

`pkg/clob`:
- `Config.TradeGate` set to a halted gate → public `CreateLimitOrder` returns
  `ErrTradingHalted`; `errors.Is` against the re-exported sentinel holds.
- `Config.TradeGate` nil → unchanged behavior.

`internal/risk`:
- A compile-time assertion that `*risk.Breaker` satisfies `clob.TradeGate`
  (placed in a clob test to avoid a risk→clob dependency), or simply exercised by
  using a `*Breaker` as the gate in a clob test.

## Components and boundaries

- **`clob.TradeGate`** — the one-method seam. Purpose: decide if submission is
  allowed. Depends on nothing. Consumers: the three create methods.
- **`clob.Client` order methods** — gain a single guard call; otherwise unchanged.
- **`risk.Breaker`** — unchanged; already implements `CanProceed()`. Continues to
  be driven entirely by the bot.
- **`pkg/clob`** — thin pass-through of the gate from public `Config` to the inner
  client.

No import cycle: `clob` defines the interface; `risk` implements it incidentally;
neither imports the other.
