package gamma

import (
	"context"
	"strings"

	"github.com/TrebuchetDynamics/polygolem/pkg/pagination"
	"github.com/TrebuchetDynamics/polygolem/pkg/types"
)

const (
	defaultEventPageSize = 100
	defaultMaxEventPages = 50
)

// ActiveEventsAll collects non-closed Gamma events across offset pages and
// deduplicates by slug, falling back to ID when slug is empty.
func (c *Client) ActiveEventsAll(ctx context.Context) ([]types.Event, error) {
	closed := false
	items, err := pagination.CollectOffset(ctx, func(ctx context.Context, offset, limit int) ([]types.Event, int, error) {
		if offset >= defaultEventPageSize*defaultMaxEventPages {
			return []types.Event{}, 0, nil
		}
		events, err := c.Events(ctx, &types.GetEventsParams{
			Closed: &closed,
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return nil, 0, err
		}
		return events, len(events), nil
	}, defaultEventPageSize)
	if err != nil {
		return nil, err
	}
	return DeduplicateEventsBySlugOrID(items), nil
}

// DeduplicateEventsBySlugOrID returns events in input order, dropping repeated
// slugs or repeated IDs when slug is empty.
func DeduplicateEventsBySlugOrID(events []types.Event) []types.Event {
	seen := make(map[string]struct{}, len(events))
	out := make([]types.Event, 0, len(events))
	for _, event := range events {
		key := strings.TrimSpace(event.Slug)
		if key == "" {
			key = strings.TrimSpace(event.ID)
		}
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, event)
	}
	return out
}

// FilterEventsByCategory applies the same Polymarket-like category aliases used
// by market browsing to event categories.
func FilterEventsByCategory(events []types.Event, category string) []types.Event {
	selected := strings.TrimSpace(strings.ToLower(category))
	if selected == "" || selected == "all" {
		return events
	}
	out := make([]types.Event, 0, len(events))
	for _, event := range events {
		if MarketMatchesCategory(event.Category, selected) {
			out = append(out, event)
		}
	}
	return out
}
