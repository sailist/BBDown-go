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
	"testing"

	"github.com/nilaonai/bbdown-go/internal/config"
	"github.com/nilaonai/bbdown-go/pkg/httpclient"
)

func TestMultiThreadDownloader_DownloadMultiThread_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mp4")

	content := strings.Repeat("a", 25) // 25 bytes, fits in one chunk
	client := newMockClient()
	client.getContentLengthFunc = func(ctx context.Context, url string) (int64, error) {
		return int64(len(content)), nil
	}
	client.getFunc = func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
		req, _ := http.NewRequest("GET", url, nil)
		for _, opt := range opts {
			opt(req)
		}
		rangeHeader := req.Header.Get("Range")
		var start, end int64
		if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end); err != nil {
			t.Fatalf("parse range: %v", err)
		}
		if end >= int64(len(content)) {
			end = int64(len(content)) - 1
		}
		return &http.Response{
			StatusCode: http.StatusPartialContent,
			Body:       io.NopCloser(strings.NewReader(content[start : end+1])),
		}, nil
	}

	d := NewMultiThreadDownloader(client, config.NewConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := d.DownloadMultiThread(context.Background(), "http://example.com/file", path, Options{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != content {
		t.Fatalf("content mismatch: got %q", string(data))
	}
}

func TestMultiThreadDownloader_DownloadMultiThread_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mp4")

	content := strings.Repeat("b", 100)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	client := newMockClient()
	client.getContentLengthFunc = func(ctx context.Context, url string) (int64, error) {
		return int64(len(content)), nil
	}

	d := NewMultiThreadDownloader(client, config.NewConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := d.DownloadMultiThread(context.Background(), "http://example.com/file", path, Options{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != content {
		t.Fatalf("content mismatch")
	}
}

func TestMultiThreadDownloader_DownloadMultiThread_RangeNotSupported(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mp4")

	client := newMockClient()
	client.getContentLengthFunc = func(ctx context.Context, url string) (int64, error) {
		return 100, nil
	}
	client.getFunc = func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("full content")),
		}, nil
	}

	d := NewMultiThreadDownloader(client, config.NewConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	err := d.DownloadMultiThread(context.Background(), "http://example.com/file", path, Options{})
	if err == nil || !strings.Contains(err.Error(), "range not supported") {
		t.Fatalf("expected range not supported error, got %v", err)
	}
}

func TestMultiThreadDownloader_Download_Delegates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mp4")

	content := strings.Repeat("x", 25)
	client := newMockClient()
	client.getContentLengthFunc = func(ctx context.Context, url string) (int64, error) {
		return int64(len(content)), nil
	}
	client.getFunc = func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
		req, _ := http.NewRequest("GET", url, nil)
		for _, opt := range opts {
			opt(req)
		}
		rangeHeader := req.Header.Get("Range")
		var start, end int64
		fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)
		if end >= int64(len(content)) {
			end = int64(len(content)) - 1
		}
		return &http.Response{
			StatusCode: http.StatusPartialContent,
			Body:       io.NopCloser(strings.NewReader(content[start : end+1])),
		}, nil
	}

	d := NewMultiThreadDownloader(client, config.NewConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := d.Download(context.Background(), "http://example.com/file", path, Options{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
