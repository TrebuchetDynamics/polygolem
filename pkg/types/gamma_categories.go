package types

// PolymarketCategory is a curated polymarket.com navigation category.
// Polymarket does not expose one public endpoint for this menu; feed-capable
// rows map to Gamma /events/keyset tag_slug queries.
type PolymarketCategory struct {
	Label    string   `json:"label"`
	Slug     string   `json:"slug"`
	Route    string   `json:"route"`
	TagSlugs []string `json:"tag_slugs,omitempty"`
	FeedMode string   `json:"feed_mode"`
	Note     string   `json:"note,omitempty"`
}

// CategoryEventsParams controls a PolymarketCategory Gamma /events/keyset feed.
type CategoryEventsParams struct {
	Limit     int    `json:"limit,omitempty"`
	Cursor    string `json:"cursor,omitempty"`
	Order     string `json:"order,omitempty"`
	Ascending *bool  `json:"ascending,omitempty"`
	Closed    *bool  `json:"closed,omitempty"`
}

// CategoryEventsResponse is a keyset-paginated category feed result.
type CategoryEventsResponse struct {
	Category   PolymarketCategory `json:"category"`
	Events     []Event            `json:"events"`
	NextCursor string             `json:"next_cursor,omitempty"`
	HasMore    bool               `json:"has_more"`
}
