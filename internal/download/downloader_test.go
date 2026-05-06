package download

import (
	"io"
	"log/slog"
	"testing"

	"github.com/nilaonai/bbdown-go/internal/config"
)

func TestDownloaderInterface(t *testing.T) {
	client := newMockClient()
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	var _ Downloader = NewSingleDownloader(client, cfg, logger)
	var _ Downloader = NewMultiThreadDownloader(client, cfg, logger)
}
