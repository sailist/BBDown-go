package download

import (
	"context"
)

// Downloader defines the contract for downloading files.
type Downloader interface {
	Download(ctx context.Context, url, path string, opts Options) error
	DownloadMultiThread(ctx context.Context, url, path string, opts Options) error
}

// Options controls download behavior.
type Options struct {
	UseAria2c    bool
	Aria2cArgs   string
	ForceHTTP    bool
	MultiThread  bool
	ShowProgress bool
}

// Ensure implementations satisfy the interface.
var (
	_ Downloader = (*SingleDownloader)(nil)
	_ Downloader = (*MultiThreadDownloader)(nil)
)
