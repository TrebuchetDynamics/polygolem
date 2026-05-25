// Package streamuser streams authenticated CLOB user events without Cobra coupling.
package streamuser

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	"github.com/TrebuchetDynamics/polygolem/internal/stream"
)

// Request contains authenticated user-stream options.
type Request struct {
	Markets     []string
	MarketsRaw  string
	URL         string
	MaxMessages int
	Credentials auth.APIKey
}

// StreamConfig contains user-stream connection options.
type StreamConfig struct {
	URL string
}

// Handlers contains event callbacks used by a Streamer adapter.
type Handlers struct {
	OnOrder func(stream.UserOrderMessage)
	OnTrade func(stream.UserTradeMessage)
	OnError func(error)
}

// Streamer is the streaming seam used by Runner.
type Streamer interface {
	SetHandlers(Handlers)
	Connect(ctx context.Context) error
	SubscribeUser(ctx context.Context, markets []string) error
	Wait(ctx context.Context, done <-chan struct{}) error
	Close()
}

// StreamFactory creates a stream adapter for one run.
type StreamFactory func(StreamConfig, auth.APIKey) Streamer

// EmitFunc receives stream events.
type EmitFunc func(interface{})

// ErrorFunc receives asynchronous stream errors for unbounded streams.
type ErrorFunc func(error)

// Runner owns authenticated user-stream orchestration.
type Runner struct {
	newStreamer StreamFactory
}

// New creates an authenticated user-stream runner.
func New(newStreamer StreamFactory) *Runner {
	if newStreamer == nil {
		newStreamer = NewInternalStreamer
	}
	return &Runner{newStreamer: newStreamer}
}

// Run connects, authenticates the user stream, and emits order/trade events until stopped.
func (r *Runner) Run(ctx context.Context, req Request, emit EmitFunc, reportError ErrorFunc) error {
	if err := req.Credentials.Validate(); err != nil {
		return fmt.Errorf("configured CLOB L2 credentials invalid: %w", err)
	}
	if emit == nil {
		emit = func(interface{}) {}
	}

	done := make(chan struct{})
	var closeDone sync.Once
	count := 0
	emitEvent := func(v interface{}) {
		if req.MaxMessages > 0 && count >= req.MaxMessages {
			return
		}
		count++
		emit(v)
		if req.MaxMessages > 0 && count >= req.MaxMessages {
			closeDone.Do(func() { close(done) })
		}
	}

	streamer := r.newStreamer(StreamConfig{URL: req.URL}, req.Credentials)
	streamer.SetHandlers(Handlers{
		OnOrder: func(msg stream.UserOrderMessage) { emitEvent(msg) },
		OnTrade: func(msg stream.UserTradeMessage) { emitEvent(msg) },
		OnError: func(err error) {
			if req.MaxMessages == 0 && reportError != nil {
				reportError(err)
			}
		},
	})

	if err := streamer.Connect(ctx); err != nil {
		return err
	}
	defer streamer.Close()
	if err := streamer.SubscribeUser(ctx, req.markets()); err != nil {
		return err
	}
	return streamer.Wait(ctx, done)
}

func (r Request) markets() []string {
	if len(r.Markets) > 0 {
		return append([]string(nil), r.Markets...)
	}
	return splitCSV(r.MarketsRaw)
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

type internalStreamer struct {
	client *stream.UserClient
}

// NewInternalStreamer creates a Streamer backed by the internal WebSocket client.
func NewInternalStreamer(config StreamConfig, credentials auth.APIKey) Streamer {
	cfg := stream.DefaultConfig(config.URL)
	cfg.PingInterval = 10 * time.Second
	return &internalStreamer{client: stream.NewUserClient(cfg, credentials)}
}

func (s *internalStreamer) SetHandlers(handlers Handlers) {
	s.client.OnOrder = handlers.OnOrder
	s.client.OnTrade = handlers.OnTrade
	s.client.OnError = handlers.OnError
}

func (s *internalStreamer) Connect(ctx context.Context) error {
	return s.client.Connect(ctx)
}

func (s *internalStreamer) SubscribeUser(ctx context.Context, markets []string) error {
	return s.client.SubscribeUser(ctx, markets)
}

func (s *internalStreamer) Wait(ctx context.Context, done <-chan struct{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (s *internalStreamer) Close() {
	s.client.Close()
}
