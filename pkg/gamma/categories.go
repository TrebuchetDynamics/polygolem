package gamma

import (
	"context"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/polygolem/pkg/types"
)

const (
	CategoryFeedEventsKeyset    = "events_keyset"
	CategoryFeedAllEventsKeyset = "events_keyset_all"
	CategoryFeedRouteOnly       = "route_only"
)

var polymarketCategories = []types.PolymarketCategory{
	{Label: "Trending", Slug: "trending", Route: "/", FeedMode: CategoryFeedAllEventsKeyset, Note: "Homepage feed; no public category-list endpoint observed."},
	{Label: "World Cup", Slug: "world-cup", Route: "/sports/world-cup", TagSlugs: []string{"fifa-world-cup", "2026-fifa-world-cup", "world-cup", "wc-tournament-futures"}, FeedMode: CategoryFeedEventsKeyset},
	{Label: "Breaking", Slug: "breaking", Route: "/breaking", FeedMode: CategoryFeedRouteOnly, Note: "Special polymarket.com route; no stable Gamma tag_slug found."},
	{Label: "Politics", Slug: "politics", Route: "/politics", TagSlugs: []string{"politics"}, FeedMode: CategoryFeedEventsKeyset},
	{Label: "Sports", Slug: "sports", Route: "/sports/live", TagSlugs: []string{"sports"}, FeedMode: CategoryFeedEventsKeyset},
	{Label: "Crypto", Slug: "crypto", Route: "/crypto", TagSlugs: []string{"crypto"}, FeedMode: CategoryFeedEventsKeyset},
	{Label: "Esports", Slug: "esports", Route: "/esports", TagSlugs: []string{"esports"}, FeedMode: CategoryFeedEventsKeyset},
	{Label: "Iran", Slug: "iran", Route: "/iran", TagSlugs: []string{"iran"}, FeedMode: CategoryFeedEventsKeyset},
	{Label: "Finance", Slug: "finance", Route: "/finance", TagSlugs: []string{"finance"}, FeedMode: CategoryFeedEventsKeyset},
	{Label: "Geopolitics", Slug: "geopolitics", Route: "/geopolitics", TagSlugs: []string{"geopolitics"}, FeedMode: CategoryFeedEventsKeyset},
	{Label: "Tech", Slug: "tech", Route: "/tech", TagSlugs: []string{"tech"}, FeedMode: CategoryFeedEventsKeyset},
	{Label: "Culture", Slug: "culture", Route: "/pop-culture", TagSlugs: []string{"pop-culture"}, FeedMode: CategoryFeedEventsKeyset},
	{Label: "Economy", Slug: "economy", Route: "/economy", TagSlugs: []string{"economy"}, FeedMode: CategoryFeedEventsKeyset},
	{Label: "Weather", Slug: "weather", Route: "/weather", TagSlugs: []string{"weather"}, FeedMode: CategoryFeedEventsKeyset},
	{Label: "Mentions", Slug: "mentions", Route: "/mentions", TagSlugs: []string{"tweets-markets"}, FeedMode: CategoryFeedEventsKeyset},
	{Label: "Elections", Slug: "elections", Route: "/elections", TagSlugs: []string{"elections"}, FeedMode: CategoryFeedEventsKeyset},
	{Label: "Art", Slug: "art", Route: "/pop-culture/art", TagSlugs: []string{"art"}, FeedMode: CategoryFeedEventsKeyset},
	{Label: "All", Slug: "all", Route: "/predictions", FeedMode: CategoryFeedAllEventsKeyset, Note: "All active events feed; exact polymarket.com count is UI state, not a dedicated API field."},
}

// PolymarketCategories returns the curated polymarket.com category navigation.
func PolymarketCategories() []types.PolymarketCategory {
	out := make([]types.PolymarketCategory, len(polymarketCategories))
	copy(out, polymarketCategories)
	return out
}

// PolymarketCategoryBySlug finds a curated category by slug or route-like slug.
func PolymarketCategoryBySlug(slug string) (types.PolymarketCategory, bool) {
	needle := normalizeCategorySlug(slug)
	for _, category := range polymarketCategories {
		if category.Slug == needle || normalizeCategorySlug(category.Label) == needle || normalizeCategorySlug(category.Route) == needle {
			return category, true
		}
		for _, tagSlug := range category.TagSlugs {
			if normalizeCategorySlug(tagSlug) == needle {
				return category, true
			}
		}
	}
	return types.PolymarketCategory{}, false
}

// CategoryEvents returns a keyset-paginated Gamma event feed for a curated category.
func (c *Client) CategoryEvents(ctx context.Context, slug string, params *types.CategoryEventsParams) (*types.CategoryEventsResponse, error) {
	category, ok := PolymarketCategoryBySlug(slug)
	if !ok {
		return nil, fmt.Errorf("unknown polymarket category %q", slug)
	}
	if category.FeedMode == CategoryFeedRouteOnly {
		return nil, fmt.Errorf("category %q is route-only and has no Gamma events/keyset feed", category.Slug)
	}
	events, cursor, err := c.inner.CategoryEvents(ctx, category, params)
	if err != nil {
		return nil, err
	}
	return &types.CategoryEventsResponse{Category: category, Events: events, NextCursor: cursor, HasMore: cursor != ""}, nil
}

func normalizeCategorySlug(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.Trim(value, "/")
	value = strings.TrimPrefix(value, "predictions/")
	value = strings.TrimPrefix(value, "sports/")
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	if value == "pop-culture" {
		return "culture"
	}
	return value
}
