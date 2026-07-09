package gamma

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
)

type categoryEventsWireResponse struct {
	Events     []polytypes.Event `json:"events"`
	NextCursor string            `json:"next_cursor"`
}

func (c *Client) CategoryEvents(ctx context.Context, category polytypes.PolymarketCategory, params *polytypes.CategoryEventsParams) ([]polytypes.Event, string, error) {
	path, err := buildCategoryEventsPath(category, params)
	if err != nil {
		return nil, "", err
	}
	var result categoryEventsWireResponse
	if err := c.transport.Get(ctx, path, &result); err != nil {
		return nil, "", err
	}
	return result.Events, result.NextCursor, nil
}

func buildCategoryEventsPath(category polytypes.PolymarketCategory, params *polytypes.CategoryEventsParams) (string, error) {
	if category.FeedMode == "route_only" || (category.FeedMode != "events_keyset_all" && len(category.TagSlugs) == 0) {
		return "", fmt.Errorf("category %q does not expose a Gamma events/keyset feed", category.Slug)
	}
	if params == nil {
		params = &polytypes.CategoryEventsParams{}
	}
	u, err := url.Parse("/events/keyset")
	if err != nil {
		return "", err
	}
	q := u.Query()
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	q.Set("limit", strconv.Itoa(limit))
	if category.FeedMode != "events_keyset_all" {
		q.Set("tag_slug", category.TagSlugs[0])
	}
	if params.Cursor != "" {
		q.Set("after_cursor", params.Cursor)
	}
	order := params.Order
	if order == "" {
		order = "volume24hr"
	}
	q.Set("order", order)
	if params.Ascending != nil {
		q.Set("ascending", strconv.FormatBool(*params.Ascending))
	} else {
		q.Set("ascending", "false")
	}
	if params.Closed != nil {
		q.Set("closed", strconv.FormatBool(*params.Closed))
	} else {
		q.Set("closed", "false")
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
