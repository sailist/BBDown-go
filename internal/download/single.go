package download

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/sailist/BBDown-go/internal/config"
	"github.com/sailist/BBDown-go/pkg/httpclient"
)

// SingleDownloader performs single-threaded HTTP downloads with Range support.
type SingleDownloader struct {
	client httpclient.Client
	cfg    *config.Config
	logger *slog.Logger
}

// NewSingleDownloader creates a new single-threaded downloader.
func NewSingleDownloader(client httpclient.Client, cfg *config.Config, logger *slog.Logger) Downloader {
	return &SingleDownloader{
		client: client,
		cfg:    cfg,
		logger: logger,
	}
}

// Download downloads a file using a single connection with resume support.
func (d *SingleDownloader) Download(ctx context.Context, url, path string, opts Options) error {
	var progress func(downloaded, total int64)
	if opts.ShowProgress {
		totalSize, err := d.client.GetContentLength(ctx, url)
		if err == nil && totalSize > 0 {
			reporter := NewConsoleReporter(totalSize)
			progress = reporter.Report
		}
	}
	return d.downloadWithProgress(ctx, url, path, opts, progress)
}

// DownloadMultiThread is not supported by SingleDownloader.
func (d *SingleDownloader) DownloadMultiThread(ctx context.Context, url, path string, opts Options) error {
	return fmt.Errorf("single downloader does not support multi-thread")
}

func (d *SingleDownloader) downloadWithProgress(ctx context.Context, url, path string, opts Options, progress func(downloaded, total int64)) error {
	tmpPath := path + ".tmp"

	totalSize, err := d.client.GetContentLength(ctx, url)
	if err != nil {
		return fmt.Errorf("get content length: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			d.logger.Warn("retrying download", "attempt", attempt+1, "error", lastErr)
		}

		lastErr = d.attemptDownload(ctx, url, tmpPath, totalSize, progress)
		if lastErr == nil {
			break
		}
	}

	if lastErr != nil {
		return fmt.Errorf("download failed after 3 attempts: %w", lastErr)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename downloaded file: %w", err)
	}
	return nil
}

func (d *SingleDownloader) attemptDownload(ctx context.Context, url, tmpPath string, totalSize int64, progress func(downloaded, total int64)) error {
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open tmp file: %w", err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat tmp file: %w", err)
	}
	downloadedBytes := fi.Size()

	// Check if already complete.
	if totalSize > 0 && downloadedBytes >= totalSize {
		return nil
	}

	var reqOpts []httpclient.RequestOption
	if totalSize > 0 {
		rangeHeader := fmt.Sprintf("bytes=%d-%d", downloadedBytes, totalSize-1)
		reqOpts = append(reqOpts, httpclient.WithHeader("Range", rangeHeader))

		if downloadedBytes > 0 {
			// If-Range based on file mod time to ensure the server file hasn't changed.
			reqOpts = append(reqOpts, httpclient.WithHeader("If-Range", fi.ModTime().UTC().Format(http.TimeFormat)))
		}
	}

	resp, err := d.client.Get(ctx, url, reqOpts...)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	// If server returns 200 instead of 206, restart from beginning.
	if resp != nil && resp.StatusCode == http.StatusOK {
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("seek to beginning: %w", err)
		}
		if err := f.Truncate(0); err != nil {
			return fmt.Errorf("truncate file: %w", err)
		}
		downloadedBytes = 0
	} else if resp != nil && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	buf := make([]byte, 32*1024)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return fmt.Errorf("write: %w", werr)
			}
			downloadedBytes += int64(n)
			if progress != nil {
				progress(downloadedBytes, totalSize)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}
	}

	return nil
}
