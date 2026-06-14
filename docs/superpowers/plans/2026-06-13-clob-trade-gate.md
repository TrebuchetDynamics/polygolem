# CLOB Trade Gate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a bot opt into a halt gate on the CLOB order client so order submission refuses to fire while the bot has halted trading, with zero behavior change when no gate is attached.

**Architecture:** Define a one-method `TradeGate` interface in `internal/clob` (no dependency on `internal/risk`). Attach it via a backward-compatible variadic option on `NewClient`. The three order-creation methods consult the gate first and return a detectable sentinel error when halted. Cancellations are never gated. Re-export the gate through `pkg/clob.Config`.

**Tech Stack:** Go, standard library (`errors`, `net/http/httptest`), existing `internal/transport` test harness.

**Note on git:** This repo is currently on `main` with unrelated uncommitted work. Follow the repo owner's standing rule — branch before committing and only commit when asked. The commit steps below are written for completeness; adapt to the owner's git workflow at execution time.

Spec: `docs/superpowers/specs/2026-06-13-clob-trade-gate-design.md`

## File Structure

- Create: `internal/clob/gate.go` — `TradeGate` interface, `ErrTradingHalted` sentinel, `Option` type, `WithTradeGate`, `ensureCanTrade` helper.
- Create: `internal/clob/gate_test.go` — gating behavior tests, cancels-not-gated test, `*risk.Breaker` compatibility test.
- Modify: `internal/clob/client.go` — add `gate TradeGate` field to `Client`; make `NewClient` variadic on options.
- Modify: `internal/clob/orders.go` — add `ensureCanTrade()` guard to `CreateLimitOrder` (line 410), `CreateBatchOrders` (line 460), `CreateMarketOrder` (line 952).
- Modify: `pkg/clob/client.go` — add `Config.TradeGate`, re-export `TradeGate` + `ErrTradingHalted`, pass option through in `NewClient`.
- Create: `pkg/clob/gate_test.go` — public-surface thread-through test.

---

### Task 1: Gate plumbing + gate-blocks-CreateLimitOrder

**Files:**
- Create: `internal/clob/gate.go`
- Create: `internal/clob/gate_test.go`
- Modify: `internal/clob/client.go` (struct `Client` ~line 22, `NewClient` ~line 67)

- [ ] **Step 1: Write the failing test**

Create `internal/clob/gate_test.go`:

```go
package clob

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/transport"
)

// fakeGate is a test TradeGate with a fixed CanProceed result.
type fakeGate struct{ ok bool }

func (g fakeGate) CanProceed() bool { return g.ok }

// failingServer fails the test if any HTTP request reaches it.
func failingServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request to %s while trading halted", r.URL.Path)
	}))
}

func TestCreateLimitOrderBlockedWhenGateHalted(t *testing.T) {
	server := failingServer(t)
	defer server.Close()

	tc := transport.New(server.Client(), transport.DefaultConfig(server.URL+"/"))
	client := NewClient(server.URL+"/", tc, WithTradeGate(fakeGate{ok: false}))

	_, err := client.CreateLimitOrder(context.Background(), testOrderPrivateKey, CreateOrderParams{})
	if !errors.Is(err, ErrTradingHalted) {
		t.Fatalf("err = %v, want ErrTradingHalted", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/clob/ -run TestCreateLimitOrderBlockedWhenGateHalted`
Expected: FAIL — compile error (`undefined: WithTradeGate`, `undefined: ErrTradingHalted`).

- [ ] **Step 3: Create the gate file**

Create `internal/clob/gate.go`:

```go
package clob

import "errors"

// TradeGate decides whether new orders may be submitted. A nil gate means
// "always allowed". *risk.Breaker satisfies this interface via its CanProceed
// method, but any type with the same method works.
type TradeGate interface {
	CanProceed() bool
}

// ErrTradingHalted is returned by the order-creation methods (CreateLimitOrder,
// CreateMarketOrder, CreateBatchOrders) when an attached TradeGate reports that
// trading is halted. Detect it with errors.Is. Cancellation methods are never
// gated and never return this error.
var ErrTradingHalted = errors.New("clob: trading halted by risk gate")

// Option configures a Client at construction time.
type Option func(*Client)

// WithTradeGate attaches a TradeGate that is consulted before each order
// submission. When the gate reports CanProceed()==false, order-creation methods
// return ErrTradingHalted without signing or sending anything.
func WithTradeGate(g TradeGate) Option {
	return func(c *Client) { c.gate = g }
}

// ensureCanTrade returns ErrTradingHalted when an attached gate reports halted.
// A nil gate always permits trading.
func (c *Client) ensureCanTrade() error {
	if c.gate != nil && !c.gate.CanProceed() {
		return ErrTradingHalted
	}
	return nil
}
```

- [ ] **Step 4: Add the gate field and make NewClient variadic**

In `internal/clob/client.go`, change the `Client` struct (currently):

```go
type Client struct {
	transport     *transport.Client
	builderCode   string
	l2Credentials *auth.APIKey
}
```

to:

```go
type Client struct {
	transport     *transport.Client
	builderCode   string
	l2Credentials *auth.APIKey
	gate          TradeGate
}
```

And change `NewClient` (currently):

```go
func NewClient(baseURL string, tc *transport.Client) *Client {
	if tc == nil {
		tc = transport.New(nil, transport.DefaultConfig(baseURL))
	}
	return &Client{transport: tc, builderCode: bytes32Zero}
}
```

to:

```go
func NewClient(baseURL string, tc *transport.Client, opts ...Option) *Client {
	if tc == nil {
		tc = transport.New(nil, transport.DefaultConfig(baseURL))
	}
	c := &Client{transport: tc, builderCode: bytes32Zero}
	for _, opt := range opts {
		opt(c)
	}
	return c
}
```

- [ ] **Step 5: Add the guard to CreateLimitOrder**

In `internal/clob/orders.go`, make `ensureCanTrade()` the first statement of `CreateLimitOrder` (line 410). Change:

```go
func (c *Client) CreateLimitOrder(ctx context.Context, privateKey string, params CreateOrderParams) (*OrderPlacementResponse, error) {
	side, err := normalizeOrderSide(params.Side)
```

to:

```go
func (c *Client) CreateLimitOrder(ctx context.Context, privateKey string, params CreateOrderParams) (*OrderPlacementResponse, error) {
	if err := c.ensureCanTrade(); err != nil {
		return nil, err
	}
	side, err := normalizeOrderSide(params.Side)
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./internal/clob/ -run TestCreateLimitOrderBlockedWhenGateHalted -v`
Expected: PASS.

- [ ] **Step 7: Verify no regression on existing callers / nil-gate path**

Run: `go build ./... && go test ./internal/clob/`
Expected: build succeeds (all 16 two-arg `NewClient` callers still compile via variadic), all existing clob tests PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/clob/gate.go internal/clob/gate_test.go internal/clob/client.go internal/clob/orders.go
git commit -m "feat(clob): add opt-in TradeGate, block CreateLimitOrder when halted"
```

---

### Task 2: Gate CreateMarketOrder + CreateBatchOrders; cancels stay open

**Files:**
- Modify: `internal/clob/orders.go` (`CreateMarketOrder` line 952, `CreateBatchOrders` line 460)
- Modify: `internal/clob/gate_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/clob/gate_test.go`:

```go
func TestCreateMarketOrderBlockedWhenGateHalted(t *testing.T) {
	server := failingServer(t)
	defer server.Close()

	tc := transport.New(server.Client(), transport.DefaultConfig(server.URL+"/"))
	client := NewClient(server.URL+"/", tc, WithTradeGate(fakeGate{ok: false}))

	_, err := client.CreateMarketOrder(context.Background(), testOrderPrivateKey, MarketOrderParams{})
	if !errors.Is(err, ErrTradingHalted) {
		t.Fatalf("err = %v, want ErrTradingHalted", err)
	}
}

func TestCreateBatchOrdersBlockedWhenGateHalted(t *testing.T) {
	server := failingServer(t)
	defer server.Close()

	tc := transport.New(server.Client(), transport.DefaultConfig(server.URL+"/"))
	client := NewClient(server.URL+"/", tc, WithTradeGate(fakeGate{ok: false}))

	_, err := client.CreateBatchOrders(context.Background(), testOrderPrivateKey, []CreateOrderParams{{}})
	if !errors.Is(err, ErrTradingHalted) {
		t.Fatalf("err = %v, want ErrTradingHalted", err)
	}
}

// Cancellation must always be allowed so a halted bot can reduce exposure.
func TestCancelOrdersNotBlockedWhenGateHalted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth/derive-api-key":
			_, _ = w.Write([]byte(`{"apiKey":"owner-key","secret":"c2VjcmV0","passphrase":"pass"}`))
		case "/orders":
			_, _ = w.Write([]byte(`{"canceled":["0x1"],"not_canceled":{}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tc := transport.New(server.Client(), transport.DefaultConfig(server.URL+"/"))
	client := NewClient(server.URL+"/", tc, WithTradeGate(fakeGate{ok: false}))

	res, err := client.CancelOrders(context.Background(), testOrderPrivateKey, []string{"0x1"})
	if err != nil {
		t.Fatalf("cancel returned error while halted: %v", err)
	}
	if errors.Is(err, ErrTradingHalted) {
		t.Fatal("cancel must not be blocked by the trade gate")
	}
	if len(res.Canceled) != 1 {
		t.Fatalf("res = %+v", res)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/clob/ -run 'TestCreateMarketOrderBlockedWhenGateHalted|TestCreateBatchOrdersBlockedWhenGateHalted'`
Expected: FAIL — the create-market and create-batch tests fail because the guard is not yet present (the server's `t.Fatalf` fires, or the call proceeds past the gate). `TestCancelOrdersNotBlockedWhenGateHalted` should already PASS (cancels were never gated).

- [ ] **Step 3: Add the guard to CreateMarketOrder**

In `internal/clob/orders.go`, change `CreateMarketOrder` (line 952):

```go
func (c *Client) CreateMarketOrder(ctx context.Context, privateKey string, params MarketOrderParams) (*OrderPlacementResponse, error) {
	side, err := normalizeOrderSide(params.Side)
```

to:

```go
func (c *Client) CreateMarketOrder(ctx context.Context, privateKey string, params MarketOrderParams) (*OrderPlacementResponse, error) {
	if err := c.ensureCanTrade(); err != nil {
		return nil, err
	}
	side, err := normalizeOrderSide(params.Side)
```

- [ ] **Step 4: Add the guard to CreateBatchOrders**

In `internal/clob/orders.go`, change `CreateBatchOrders` (line 460):

```go
func (c *Client) CreateBatchOrders(ctx context.Context, privateKey string, params []CreateOrderParams) (*BatchOrderResponse, error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("no orders to post")
```

to:

```go
func (c *Client) CreateBatchOrders(ctx context.Context, privateKey string, params []CreateOrderParams) (*BatchOrderResponse, error) {
	if err := c.ensureCanTrade(); err != nil {
		return nil, err
	}
	if len(params) == 0 {
		return nil, fmt.Errorf("no orders to post")
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/clob/ -run 'Gate' -v`
Expected: PASS for all gate tests (limit, market, batch blocked; cancels open).

- [ ] **Step 6: Run the full clob package**

Run: `go test ./internal/clob/`
Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/clob/orders.go internal/clob/gate_test.go
git commit -m "feat(clob): gate CreateMarketOrder and CreateBatchOrders; keep cancels open"
```

---

### Task 3: Confirm risk.Breaker works as a TradeGate

**Files:**
- Modify: `internal/clob/gate_test.go`

- [ ] **Step 1: Write the test**

Append to `internal/clob/gate_test.go` (add the import `"github.com/TrebuchetDynamics/polygolem/internal/risk"` to the file's import block):

```go
// A *risk.Breaker must be usable directly as a TradeGate. This is the intended
// production wiring: a bot constructs a Breaker and passes it as the gate.
func TestRiskBreakerSatisfiesTradeGate(t *testing.T) {
	var _ TradeGate = risk.NewBreaker(risk.DefaultPolicy())

	server := failingServer(t)
	defer server.Close()

	breaker := risk.NewBreaker(risk.DefaultPolicy())
	breaker.Halt() // bot decides to stop trading

	tc := transport.New(server.Client(), transport.DefaultConfig(server.URL+"/"))
	client := NewClient(server.URL+"/", tc, WithTradeGate(breaker))

	_, err := client.CreateLimitOrder(context.Background(), testOrderPrivateKey, CreateOrderParams{})
	if !errors.Is(err, ErrTradingHalted) {
		t.Fatalf("err = %v, want ErrTradingHalted", err)
	}
}
```

- [ ] **Step 2: Run test to verify it passes**

Run: `go test ./internal/clob/ -run TestRiskBreakerSatisfiesTradeGate -v`
Expected: PASS. (If it fails to compile, confirm `risk.NewBreaker`, `risk.DefaultPolicy`, and `(*Breaker).Halt` exist — they do per `internal/risk/breaker.go`.)

- [ ] **Step 3: Verify no import cycle**

Run: `go build ./...`
Expected: build succeeds. `internal/clob` does not import `internal/risk` in non-test code; the test file (`package clob`) importing `risk` is fine because `risk` does not import `clob`.

- [ ] **Step 4: Commit**

```bash
git add internal/clob/gate_test.go
git commit -m "test(clob): verify risk.Breaker satisfies TradeGate"
```

---

### Task 4: Expose the gate through pkg/clob

**Files:**
- Modify: `pkg/clob/client.go` (`Config` struct ~line 22, `NewClient` ~line 44)
- Create: `pkg/clob/gate_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/clob/gate_test.go`:

```go
package clob

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type haltedGate struct{}

func (haltedGate) CanProceed() bool { return false }

func TestConfigTradeGateBlocksCreateLimitOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request to %s while trading halted", r.URL.Path)
	}))
	defer server.Close()

	c := NewClient(Config{BaseURL: server.URL, TradeGate: haltedGate{}})

	_, err := c.CreateLimitOrder(context.Background(), "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d", CreateOrderParams{})
	if !errors.Is(err, ErrTradingHalted) {
		t.Fatalf("err = %v, want ErrTradingHalted", err)
	}
}

func TestConfigNoTradeGateUnchanged(t *testing.T) {
	// A nil TradeGate must not block: this client reaches the server (and fails
	// there for unrelated reasons), proving the gate did not short-circuit.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient(Config{BaseURL: server.URL})
	_, err := c.CreateLimitOrder(context.Background(), "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d", CreateOrderParams{})
	if errors.Is(err, ErrTradingHalted) {
		t.Fatalf("nil gate must not return ErrTradingHalted, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/clob/ -run TestConfigTradeGate`
Expected: FAIL — compile error (`Config has no field TradeGate`, `undefined: ErrTradingHalted`).

- [ ] **Step 3: Add re-exports and Config field**

In `pkg/clob/client.go`, add after the imports / `defaultBaseURL` const (before `type Config struct`):

```go
// TradeGate decides whether new orders may be submitted. Attach one via
// Config.TradeGate to make order submission respect a halt decision. A
// *risk.Breaker (internal/risk) satisfies this interface.
type TradeGate = internalclob.TradeGate

// ErrTradingHalted is returned by order-creation methods when an attached
// TradeGate reports trading is halted. Detect it with errors.Is. Cancellation
// is never blocked.
var ErrTradingHalted = internalclob.ErrTradingHalted
```

Add a field to `Config`:

```go
type Config struct {
	BaseURL string
	// BuilderCode is the optional V2 order builder attribution bytes32.
	// Empty values sign orders with the zero bytes32 builder code.
	BuilderCode string
	// Credentials are pre-provisioned CLOB L2 HMAC credentials. When set,
	// authenticated deposit-wallet calls use them instead of deriving a key
	// through /auth/derive-api-key.
	Credentials APIKey
	// TradeGate, when set, blocks order submission while it reports trading is
	// halted (CanProceed()==false). Cancellation is never blocked. Nil = no gate.
	TradeGate TradeGate
}
```

- [ ] **Step 4: Thread the gate through NewClient**

In `pkg/clob/client.go`, change `NewClient`:

```go
func NewClient(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	inner := internalclob.NewClient(cfg.BaseURL, nil)
	inner.SetBuilderCode(cfg.BuilderCode)
	if apiKeyConfigured(cfg.Credentials) {
		inner.SetL2Credentials(apiKeyToInternal(cfg.Credentials))
	}
	return &Client{inner: inner}
}
```

to:

```go
func NewClient(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	var opts []internalclob.Option
	if cfg.TradeGate != nil {
		opts = append(opts, internalclob.WithTradeGate(cfg.TradeGate))
	}
	inner := internalclob.NewClient(cfg.BaseURL, nil, opts...)
	inner.SetBuilderCode(cfg.BuilderCode)
	if apiKeyConfigured(cfg.Credentials) {
		inner.SetL2Credentials(apiKeyToInternal(cfg.Credentials))
	}
	return &Client{inner: inner}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./pkg/clob/ -run TestConfig -v`
Expected: PASS for both `TestConfigTradeGateBlocksCreateLimitOrder` and `TestConfigNoTradeGateUnchanged`.

- [ ] **Step 6: Commit**

```bash
git add pkg/clob/client.go pkg/clob/gate_test.go
git commit -m "feat(clob/pkg): expose TradeGate via Config and re-export ErrTradingHalted"
```

---

### Task 5: Full verification

- [ ] **Step 1: Vet and full test suite**

Run: `go vet ./... && go test ./...`
Expected: vet reports no issues; full suite PASSES (the existing ~1005 tests plus the new gate tests).

- [ ] **Step 2: Confirm zero behavior change for default callers**

Run: `git grep -n "clob.NewClient(" -- 'internal/*' 'cmd/*' 'pkg/*' | grep -v _test`
Expected: all call sites still use the two-argument form and compile (variadic option is additive). No call site needs editing.

- [ ] **Step 3: Final commit (if any staged docs/cleanup remain)**

```bash
git add -A
git commit -m "chore(clob): finalize trade gate wiring" || true
```

---

## Self-Review

**Spec coverage:**
- Opt-in halt gate on order client → Task 1 (`TradeGate`, `WithTradeGate`, `gate` field).
- Three create methods consult gate before signing/sending → Task 1 (limit), Task 2 (market, batch).
- Cancellation never gated → Task 2 (`TestCancelOrdersNotBlockedWhenGateHalted`; no guard added to cancel methods).
- Detectable sentinel error → Task 1 (`ErrTradingHalted`, `errors.Is`).
- `risk.Breaker` usable as gate, no import cycle → Task 3.
- Public SDK exposure via `pkg/clob.Config` + re-exports → Task 4.
- Zero behavior change when no gate → Task 1 Step 7, Task 4 `TestConfigNoTradeGateUnchanged`, Task 5 Step 2.
- Out-of-scope items (no limit math, no auto RecordX, no persistence, no pkg/universal) → not implemented, by design.

**Placeholder scan:** No TBD/TODO; every code step shows complete code; every command shows expected output. None found.

**Type consistency:** `TradeGate`, `ErrTradingHalted`, `Option`, `WithTradeGate`, `ensureCanTrade`, `gate` field, and `Config.TradeGate` are named identically across all tasks. `fakeGate`/`haltedGate` are test-local. Method signatures match the real source (`CreateLimitOrder`/`CreateMarketOrder` → `*OrderPlacementResponse`; `CreateBatchOrders` → `*BatchOrderResponse`; `CancelOrders` → `*CancelOrdersResponse`).
