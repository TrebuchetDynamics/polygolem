package gamma

import (
	"context"
	"strings"

	"github.com/TrebuchetDynamics/polygolem/pkg/pagination"
	"github.com/TrebuchetDynamics/polygolem/pkg/types"
)

const (
	defaultSeriesPageSize = 100
	defaultMaxSeriesPages = 50
	defaultTagPageSize    = 100
	defaultMaxTagPages    = 50
)

// ActiveSeriesAll collects non-closed Gamma series across offset pages and
// deduplicates by slug, falling back to ID when slug is empty.
func (c *Client) ActiveSeriesAll(ctx context.Context) ([]types.Series, error) {
	closed := false
	items, err := pagination.CollectOffset(ctx, func(ctx context.Context, offset, limit int) ([]types.Series, int, error) {
		if offset >= defaultSeriesPageSize*defaultMaxSeriesPages {
			return []types.Series{}, 0, nil
		}
		series, err := c.Series(ctx, &types.GetSeriesParams{
			Closed: &closed,
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return nil, 0, err
		}
		return series, len(series), nil
	}, defaultSeriesPageSize)
	if err != nil {
		return nil, err
	}
	return DeduplicateSeriesBySlugOrID(items), nil
}

// TagsAll collects Gamma tags across offset pages and deduplicates by slug,
// falling back to ID when slug is empty.
func (c *Client) TagsAll(ctx context.Context) ([]types.Tag, error) {
	items, err := pagination.CollectOffset(ctx, func(ctx context.Context, offset, limit int) ([]types.Tag, int, error) {
		if offset >= defaultTagPageSize*defaultMaxTagPages {
			return []types.Tag{}, 0, nil
		}
		tags, err := c.Tags(ctx, &types.GetTagsParams{Limit: limit, Offset: offset})
		if err != nil {
			return nil, 0, err
		}
		return tags, len(tags), nil
	}, defaultTagPageSize)
	if err != nil {
		return nil, err
	}
	return DeduplicateTagsBySlugOrID(items), nil
}

func DeduplicateSeriesBySlugOrID(series []types.Series) []types.Series {
	seen := make(map[string]struct{}, len(series))
	out := make([]types.Series, 0, len(series))
	for _, item := range series {
		key := strings.TrimSpace(item.Slug)
		if key == "" {
			key = strings.TrimSpace(item.ID)
		}
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func DeduplicateTagsBySlugOrID(tags []types.Tag) []types.Tag {
	seen := make(map[string]struct{}, len(tags))
	out := make([]types.Tag, 0, len(tags))
	for _, tag := range tags {
		key := strings.TrimSpace(tag.Slug)
		if key == "" {
			key = strings.TrimSpace(tag.ID)
		}
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, tag)
	}
	return out
}
