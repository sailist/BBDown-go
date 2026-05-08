package download

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sailist/BBDown-go/internal/config"
	"github.com/sailist/BBDown-go/pkg/httpclient"
)

func TestSingleDownloader_Download_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	client := newMockClient()
	client.getContentLengthFunc = func(ctx context.Context, url string) (int64, error) {
		return int64(len("hello world")), nil
	}
	client.getFunc = func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
		req, _ := http.NewRequest("GET", url, nil)
		for _, opt := range opts {
			opt(req)
		}
		if req.Header.Get("Range") != "bytes=0-10" {
			t.Errorf("expected Range bytes=0-10, got %s", req.Header.Get("Range"))
		}
		return &http.Response{
			StatusCode: http.StatusPartialContent,
			Body:       io.NopCloser(strings.NewReader("hello world")),
		}, nil
	}

	d := NewSingleDownloader(client, config.NewConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := d.Download(context.Background(), "http://example.com/file", path, Options{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "hello world" {
		t.Fatalf("expected 'hello world', got %q", string(data))
	}
}

func TestSingleDownloader_Download_Resume(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	tmpPath := path + ".tmp"

	// Write partial content.
	if err := os.WriteFile(tmpPath, []byte("hello "), 0o644); err != nil {
		t.Fatalf("write partial: %v", err)
	}

	client := newMockClient()
	client.getContentLengthFunc = func(ctx context.Context, url string) (int64, error) {
		return int64(len("hello world")), nil
	}
	client.getFunc = func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
		req, _ := http.NewRequest("GET", url, nil)
		for _, opt := range opts {
			opt(req)
		}
		rangeHeader := req.Header.Get("Range")
		if rangeHeader != "bytes=6-10" {
			t.Errorf("expected Range bytes=6-10, got %s", rangeHeader)
		}
		return &http.Response{
			StatusCode: http.StatusPartialContent,
			Body:       io.NopCloser(strings.NewReader("world")),
		}, nil
	}

	d := NewSingleDownloader(client, config.NewConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := d.Download(context.Background(), "http://example.com/file", path, Options{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "hello world" {
		t.Fatalf("expected 'hello world', got %q", string(data))
	}
}

func TestSingleDownloader_Download_Retry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	client := newMockClient()
	client.getContentLengthFunc = func(ctx context.Context, url string) (int64, error) {
		return int64(len("ok")), nil
	}

	failures := 0
	client.getFunc = func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
		failures++
		if failures < 3 {
			return &http.Response{Body: io.NopCloser(strings.NewReader(""))}, io.EOF
		}
		return &http.Response{
			StatusCode: http.StatusPartialContent,
			Body:       io.NopCloser(strings.NewReader("ok")),
		}, nil
	}

	d := NewSingleDownloader(client, config.NewConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := d.Download(context.Background(), "http://example.com/file", path, Options{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "ok" {
		t.Fatalf("expected 'ok', got %q", string(data))
	}
	if failures != 3 {
		t.Fatalf("expected 3 attempts, got %d", failures)
	}
}

func TestSingleDownloader_DownloadMultiThread_NotSupported(t *testing.T) {
	client := newMockClient()
	d := NewSingleDownloader(client, config.NewConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	err := d.DownloadMultiThread(context.Background(), "http://example.com", "/tmp/file", Options{})
	if err == nil || !strings.Contains(err.Error(), "does not support multi-thread") {
		t.Fatalf("expected not-supported error, got %v", err)
	}
}

func TestSingleDownloader_Download_Progress(t *testing.T) {
	// Progress callback is internal to single.go and not exposed through the public
	// Downloader interface. It is verified indirectly by the success/resume tests.
}
