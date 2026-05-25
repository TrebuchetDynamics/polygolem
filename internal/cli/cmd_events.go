package cli

import (
	"context"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/eventreads"
	"github.com/spf13/cobra"
)

type eventsRunner interface {
	Run(context.Context, eventreads.Request) ([]polytypes.Event, error)
}

func eventsCmd(jsonOut bool) *cobra.Command {
	w := newWire(jsonOut)
	return newEventsCommand(eventreads.New(w.gamma))
}

func newEventsCommand(runner eventsRunner) *cobra.Command {
	var limit int
	cmd := commandGroup("events", "List Polymarket events")
	cmd.AddCommand(&cobra.Command{
		Use: "list", Short: "List events", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			events, err := runner.Run(cmd.Context(), eventreads.Request{Limit: limit})
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, events)
		},
	})
	return cmd
}
