package fetcher

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"unicode"

	"github.com/nilaonai/bbdown-go/internal/config"
	"github.com/nilaonai/bbdown-go/internal/core/entity"
	"github.com/nilaonai/bbdown-go/pkg/httpclient"
)

// Fetcher fetches video metadata for a given ID.
type Fetcher interface {
	Fetch(ctx context.Context, id string) (*entity.VInfo, error)
}

// Factory creates a Fetcher based on the ID prefix and API preference.
type Factory struct {
	client httpclient.Client
	cfg    *config.Config
	logger *slog.Logger
}

// NewFactory returns a new Factory with the given dependencies.
func NewFactory(client httpclient.Client, cfg *config.Config, logger *slog.Logger) *Factory {
	return &Factory{
		client: client,
		cfg:    cfg,
		logger: logger,
	}
}

// Create returns a Fetcher suitable for the given ID.
func (f *Factory) Create(id string, useIntl bool) (Fetcher, error) {
	switch {
	case strings.HasPrefix(id, "cheese:"):
		return NewCheeseFetcher(f.client, f.cfg, f.logger), nil
	case strings.HasPrefix(id, "ep:"):
		if useIntl {
			return NewIntlBangumiFetcher(f.client, f.cfg, f.logger), nil
		}
		return NewBangumiFetcher(f.client, f.cfg, f.logger), nil
	case strings.HasPrefix(id, "mid"):
		return nil, errors.New("not implemented")
	case strings.HasPrefix(id, "listBizId"):
		return nil, errors.New("not implemented")
	case strings.HasPrefix(id, "seriesBizId"):
		return nil, errors.New("not implemented")
	case strings.HasPrefix(id, "favId"):
		return nil, errors.New("not implemented")
	case isNumeric(id):
		return NewNormalFetcher(f.client, f.cfg, f.logger), nil
	default:
		return nil, errors.New("not implemented")
	}
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
