// Package bridgeassets owns read-only bridge supported-assets behavior without Cobra coupling.
package bridgeassets

import (
	"context"

	"github.com/TrebuchetDynamics/polygolem/pkg/bridge"
)

// Reader is the Bridge read adapter used by this workflow.
type Reader interface {
	GetSupportedAssets(context.Context) (*bridge.SupportedAssetsResponse, error)
}

// Runner executes read-only bridge supported-assets requests.
type Runner struct {
	reader Reader
}

// New creates a bridge assets workflow runner.
func New(reader Reader) *Runner {
	return &Runner{reader: reader}
}

// Run returns the assets currently supported by the Bridge.
func (r *Runner) Run(ctx context.Context) (*bridge.SupportedAssetsResponse, error) {
	return r.reader.GetSupportedAssets(ctx)
}
