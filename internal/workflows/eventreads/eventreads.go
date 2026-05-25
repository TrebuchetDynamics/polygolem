// Package eventreads owns read-only Gamma event-list behavior without Cobra coupling.
package eventreads

import (
	"context"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
)

// Request contains one read-only event-list request.
type Request struct {
	Limit int
}

// Reader is the Gamma read adapter used by this workflow.
type Reader interface {
	Events(context.Context, *polytypes.GetEventsParams) ([]polytypes.Event, error)
}

// Runner executes read-only event-list requests.
type Runner struct {
	reader Reader
}

// New creates an event reads workflow runner.
func New(reader Reader) *Runner {
	return &Runner{reader: reader}
}

// Run lists Gamma events for req.
func (r *Runner) Run(ctx context.Context, req Request) ([]polytypes.Event, error) {
	return r.reader.Events(ctx, &polytypes.GetEventsParams{Limit: req.Limit})
}
