package download

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/nilaonai/bbdown-go/internal/config"
	"github.com/nilaonai/bbdown-go/pkg/httpclient"
	"golang.org/x/sync/errgroup"
)

const chunkSize = 20 * 1024 * 1024 // 20MB

// MultiThreadDownloader performs multi-threaded HTTP downloads using range requests.
type MultiThreadDownloader struct {
	client httpclient.Client
	cfg    *config.Config
	logger *slog.Logger
}

// NewMultiThreadDownloader creates a new multi-threaded downloader.
func NewMultiThreadDownloader(client httpclient.Client, cfg *config.Config, logger *slog.Logger) Downloader {
	return &MultiThreadDownloader{
		client: client,
		cfg:    cfg,
		logger: logger,
	}
}

// Download delegates to DownloadMultiThread.
func (d *MultiThreadDownloader) Download(ctx context.Context, url, path string, opts Options) error {
	return d.DownloadMultiThread(ctx, url, path, opts)
}

// DownloadMultiThread downloads a file using multiple concurrent connections.
func (d *MultiThreadDownloader) DownloadMultiThread(ctx context.Context, url, path string, opts Options) error {
	totalSize, err := d.client.GetContentLength(ctx, url)
	if err != nil {
		return fmt.Errorf("get content length: %w", err)
	}

	// Already downloaded?
	fi, err := os.Stat(path)
	if err == nil && fi.Size() == totalSize {
		return nil
	}

	if totalSize <= 0 {
		return fmt.Errorf("unknown or zero content length")
	}

	numChunks := int((totalSize + chunkSize - 1) / chunkSize)

	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	dir := filepath.Dir(path)

	var g errgroup.Group
	g.SetLimit(8)

	for i := 0; i < numChunks; i++ {
		i := i
		g.Go(func() error {
			start := int64(i) * chunkSize
			end := start + chunkSize - 1
			if end >= totalSize {
				end = totalSize - 1
			}

			tmpExt := ".aclip"
			if ext == ".mp4" {
				tmpExt = ".vclip"
			}
			tmpName := fmt.Sprintf("%05d_%s%s", i, name, tmpExt)
			tmpPath := filepath.Join(dir, tmpName)

			return d.downloadChunk(ctx, url, tmpPath, start, end)
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("chunk download failed: %w", err)
	}

	// Merge chunks in order.
	out, err := os.Create(path + ".tmp")
	if err != nil {
		return fmt.Errorf("create merge file: %w", err)
	}

	for i := 0; i < numChunks; i++ {
		tmpExt := ".aclip"
		if ext == ".mp4" {
			tmpExt = ".vclip"
		}
		tmpName := fmt.Sprintf("%05d_%s%s", i, name, tmpExt)
		tmpPath := filepath.Join(dir, tmpName)

		in, err := os.Open(tmpPath)
		if err != nil {
			out.Close()
			return fmt.Errorf("open chunk %d: %w", i, err)
		}
		_, err = io.Copy(out, in)
		in.Close()
		if err != nil {
			out.Close()
			return fmt.Errorf("copy chunk %d: %w", i, err)
		}
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("close merge file: %w", err)
	}

	// Clean up temp files.
	for i := 0; i < numChunks; i++ {
		tmpExt := ".aclip"
		if ext == ".mp4" {
			tmpExt = ".vclip"
		}
		tmpName := fmt.Sprintf("%05d_%s%s", i, name, tmpExt)
		tmpPath := filepath.Join(dir, tmpName)
		os.Remove(tmpPath)
	}

	return os.Rename(path+".tmp", path)
}

func (d *MultiThreadDownloader) downloadChunk(ctx context.Context, url, path string, start, end int64) error {
	// Check if chunk already exists and is complete.
	fi, err := os.Stat(path)
	if err == nil {
		expectedSize := end - start + 1
		if fi.Size() == expectedSize {
			return nil
		}
	}

	rangeHeader := fmt.Sprintf("bytes=%d-%d", start, end)
	resp, err := d.client.Get(ctx, url, httpclient.WithHeader("Range", rangeHeader))
	if err != nil {
		return fmt.Errorf("get chunk: %w", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("range not supported, status: %d", resp.StatusCode)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create chunk file: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("write chunk: %w", err)
	}

	return nil
}
