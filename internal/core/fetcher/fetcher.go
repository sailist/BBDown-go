package fetcher

import (
	"context"
	"errors"
	"strings"

	"github.com/nilaonai/bbdown-go/internal/core/entity"
)

// Fetcher fetches video metadata for a given ID.
type Fetcher interface {
	Fetch(ctx context.Context, id string) (*entity.VInfo, error)
}

// Factory creates a Fetcher based on the ID prefix and API preference.
type Factory func(id string, useIntl bool) (Fetcher, error)

// NewFetcher returns a Fetcher suitable for the given ID.
// Actual implementations will be provided in subsequent commits.
func NewFetcher(id string, useIntl bool) (Fetcher, error) {
	switch {
	case strings.HasPrefix(id, "cheese"):
		return nil, errors.New("not implemented")
	case strings.HasPrefix(id, "ep"):
		_ = useIntl // routing placeholder for intl vs domestic bangumi
		return nil, errors.New("not implemented")
	case strings.HasPrefix(id, "mid"):
		return nil, errors.New("not implemented")
	case strings.HasPrefix(id, "listBizId"):
		return nil, errors.New("not implemented")
	case strings.HasPrefix(id, "seriesBizId"):
		return nil, errors.New("not implemented")
	case strings.HasPrefix(id, "favId"):
		return nil, errors.New("not implemented")
	default:
		return nil, errors.New("not implemented")
	}
}
