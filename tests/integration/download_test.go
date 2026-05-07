package integration

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nilaonai/bbdown-go/internal/app"
	"github.com/nilaonai/bbdown-go/internal/cli"
	"github.com/nilaonai/bbdown-go/internal/core/fetcher"
	"github.com/nilaonai/bbdown-go/internal/core/parser"
	"github.com/nilaonai/bbdown-go/internal/download"
	"github.com/nilaonai/bbdown-go/internal/muxer"
	"github.com/nilaonai/bbdown-go/pkg/httpclient"
)

// createMockBinary writes a shell script that acts as a mock external binary.
func createMockBinary(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	script := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("create mock binary %s: %v", name, err)
	}
}

// mockHTTPClient routes Bilibili API requests to a local httptest.Server.
type mockHTTPClient struct {
	server *httptest.Server
}

func (c *mockHTTPClient) rewriteURL(urlStr string) string {
	if strings.Contains(urlStr, "bilibili.com") || strings.Contains(urlStr, "bilibili.tv") {
		u, err := url.Parse(urlStr)
		if err == nil {
			result := c.server.URL + u.Path
			if u.RawQuery != "" {
				result += "?" + u.RawQuery
			}
			return result
		}
	}
	return urlStr
}

func (c *mockHTTPClient) Get(ctx context.Context, urlStr string, opts ...httpclient.RequestOption) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.rewriteURL(urlStr), nil)
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		opt(req)
	}
	return http.DefaultClient.Do(req)
}

func (c *mockHTTPClient) Post(ctx context.Context, urlStr string, body []byte, opts ...httpclient.RequestOption) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.rewriteURL(urlStr), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		opt(req)
	}
	return http.DefaultClient.Do(req)
}

func (c *mockHTTPClient) Head(ctx context.Context, urlStr string, opts ...httpclient.RequestOption) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, c.rewriteURL(urlStr), nil)
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		opt(req)
	}
	return http.DefaultClient.Do(req)
}

func (c *mockHTTPClient) GetRedirectLocation(ctx context.Context, urlStr string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, c.rewriteURL(urlStr), nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.Request != nil && resp.Request.URL != nil {
		return resp.Request.URL.String(), nil
	}
	return urlStr, nil
}

func (c *mockHTTPClient) GetContentLength(ctx context.Context, urlStr string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, c.rewriteURL(urlStr), nil)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.ContentLength, nil
}

// mockDownloader is a test double for download.Downloader.
type mockDownloader struct {
	downloadFunc          func(ctx context.Context, url, path string, opts download.Options) error
	downloadMultiThreadFn func(ctx context.Context, url, path string, opts download.Options) error
}

func (m *mockDownloader) Download(ctx context.Context, url, path string, opts download.Options) error {
	if m.downloadFunc != nil {
		return m.downloadFunc(ctx, url, path, opts)
	}
	return fmt.Errorf("mock Download not implemented")
}

func (m *mockDownloader) DownloadMultiThread(ctx context.Context, url, path string, opts download.Options) error {
	if m.downloadMultiThreadFn != nil {
		return m.downloadMultiThreadFn(ctx, url, path, opts)
	}
	return fmt.Errorf("mock DownloadMultiThread not implemented")
}

// mockMuxer is a test double for muxer.Muxer.
type mockMuxer struct {
	muxFn      func(cfg muxer.MuxConfig) error
	mergeFLVFn func(ctx context.Context, files []string, outPath string) error
}

func (m *mockMuxer) Mux(ctx context.Context, cfg muxer.MuxConfig) error {
	if m.muxFn != nil {
		return m.muxFn(cfg)
	}
	return fmt.Errorf("mock Mux not implemented")
}

func (m *mockMuxer) MergeFLV(ctx context.Context, files []string, outPath string) error {
	if m.mergeFLVFn != nil {
		return m.mergeFLVFn(ctx, files, outPath)
	}
	return fmt.Errorf("mock MergeFLV not implemented")
}

// newMockBilibiliServer creates an httptest.Server that mimics the Bilibili API.
func newMockBilibiliServer(t *testing.T) *httptest.Server {
	t.Helper()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/x/web-interface/nav"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"data":{"isLogin":false,"wbi_img":{"img_url":"","sub_url":""}}}`))

		case strings.HasPrefix(r.URL.Path, "/x/web-interface/view"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"data": {
					"title": "Test Video",
					"desc": "Test Description",
					"pic": "http://example.com/cover.jpg",
					"pubdate": 1609459200,
					"bvid": "BV1xx411c7mD",
					"cid": 987654321,
					"owner": {"mid": 123456, "name": "TestUP"},
					"rights": {"is_stein_gate": 0},
					"pages": [
						{"page": 1, "cid": 987654321, "part": "P1", "duration": 120, "dimension": {"width": 1920, "height": 1080}}
					]
				}
			}`))

		case strings.HasPrefix(r.URL.Path, "/x/player/wbi/playurl"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(fmt.Sprintf(`{"data":{"duration":120,"dash":{"video":[{"id":"80","bandwidth":1000000,"base_url":"%s/video.m4s","codecid":"7","width":"1920","height":"1080","frame_rate":"30"}],"audio":[{"id":"30280","bandwidth":128000,"base_url":"%s/audio.m4s","codecs":"mp4a.40.2"}]}}}`, server.URL, server.URL)))

		case strings.HasPrefix(r.URL.Path, "/x/player/wbi/v2"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"data":{"view_points":[]}}`))

		case strings.HasPrefix(r.URL.Path, "/video/av"):
			w.WriteHeader(http.StatusOK)

		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	return server
}

// discardLogger returns a logger that discards all output.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

// TestDownloadWorkflow_DASH tests the complete download workflow using mocked
// Bilibili APIs and mocked external binaries.
func TestDownloadWorkflow_DASH(t *testing.T) {
	if os.Getenv("BBDOWN_TEST_NETWORK") == "1" {
		t.Skip("skipping mock test in network mode")
	}

	tmpDir := t.TempDir()

	// Create mock external binaries so SetupWork passes validation.
	createMockBinary(t, tmpDir, "ffmpeg")
	createMockBinary(t, tmpDir, "mp4box")
	createMockBinary(t, tmpDir, "aria2c")

	// Change cwd to tmpDir so findBinaries and the real FFmpegMuxer
	// (if used) can locate the mock scripts via current directory.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origDir)

	server := newMockBilibiliServer(t)
	defer server.Close()

	client := &mockHTTPClient{server: server}

	opt := cli.NewOption()
	opt.URL = "av123456789"
	opt.FFmpegPath = filepath.Join(tmpDir, "ffmpeg")
	opt.Mp4boxPath = filepath.Join(tmpDir, "mp4box")
	opt.Aria2cPath = filepath.Join(tmpDir, "aria2c")
	opt.WorkDir = tmpDir
	opt.Debug = false
	opt.HideStreams = true
	opt.SkipSubtitle = true
	opt.SkipCover = true
	opt.Interactive = false
	opt.MultiThread = false // use single-thread to keep the test simple

	logger := discardLogger()

	workCfg, err := app.SetupWork(opt, logger)
	if err != nil {
		t.Fatalf("SetupWork failed: %v", err)
	}

	// GetVideoInfo with real fetcher factory backed by mock HTTP client.
	aidOri, vInfo, _, err := app.GetVideoInfo(context.Background(), opt, workCfg.Input, app.Deps{
		FetcherFactory: func(id string, useIntl bool) (fetcher.Fetcher, error) {
			factory := fetcher.NewFactory(client, workCfg.Config, logger)
			return factory.Create(id, useIntl)
		},
		HTTPClient: client,
		Logger:     logger,
	})
	if err != nil {
		t.Fatalf("GetVideoInfo failed: %v", err)
	}
	workCfg.AidOri = aidOri

	if vInfo.Title != "Test Video" {
		t.Errorf("expected title 'Test Video', got %q", vInfo.Title)
	}
	if len(vInfo.PagesInfo) != 1 {
		t.Fatalf("expected 1 page, got %d", len(vInfo.PagesInfo))
	}

	// Track the output path produced by the muxer.
	var muxedPath string
	mockMX := &mockMuxer{
		muxFn: func(cfg muxer.MuxConfig) error {
			muxedPath = cfg.OutPath
			if dir := filepath.Dir(cfg.OutPath); dir != "" && dir != "." {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return fmt.Errorf("create output dir: %w", err)
				}
			}
			return os.WriteFile(cfg.OutPath, []byte("mock output"), 0o644)
		},
	}

	mockDL := &mockDownloader{
		downloadFunc: func(ctx context.Context, url, path string, opts download.Options) error {
			if dir := filepath.Dir(path); dir != "" && dir != "." {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return fmt.Errorf("create download dir: %w", err)
				}
			}
			return os.WriteFile(path, []byte("mock media data"), 0o644)
		},
		downloadMultiThreadFn: func(ctx context.Context, url, path string, opts download.Options) error {
			if dir := filepath.Dir(path); dir != "" && dir != "." {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return fmt.Errorf("create download dir: %w", err)
				}
			}
			return os.WriteFile(path, []byte("mock media data"), 0o644)
		},
	}

	deps := app.DownloadDeps{
		HTTPClient:    client,
		Logger:        logger,
		Downloader:    mockDL,
		Muxer:         mockMX,
		Config:        workCfg.Config,
		ExtractTracks: parser.ExtractTracks,
	}

	if err := app.DownloadPages(context.Background(), opt, vInfo, workCfg, deps); err != nil {
		t.Fatalf("DownloadPages failed: %v", err)
	}

	if muxedPath == "" {
		t.Fatal("muxer was not called")
	}
	if _, err := os.Stat(muxedPath); err != nil {
		t.Fatalf("expected output file %q to exist: %v", muxedPath, err)
	}
}

// TestDownloadWorkflow_RealNetwork performs a lightweight real-network check
// against Bilibili when BBDOWN_TEST_NETWORK=1 is set.
func TestDownloadWorkflow_RealNetwork(t *testing.T) {
	if os.Getenv("BBDOWN_TEST_NETWORK") != "1" {
		t.Skip("skipping real network test; set BBDOWN_TEST_NETWORK=1 to enable")
	}

	tmpDir := t.TempDir()
	createMockBinary(t, tmpDir, "ffmpeg")

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origDir)

	opt := cli.NewOption()
	opt.URL = "BV1GJ411x7h7"
	opt.FFmpegPath = filepath.Join(tmpDir, "ffmpeg")
	opt.WorkDir = tmpDir
	opt.OnlyShowInfo = true // do not attempt to download
	opt.HideStreams = true

	logger := discardLogger()

	workCfg, err := app.SetupWork(opt, logger)
	if err != nil {
		t.Fatalf("SetupWork failed: %v", err)
	}

	client := httpclient.NewStandardClient(logger)
	_, vInfo, _, err := app.GetVideoInfo(context.Background(), opt, workCfg.Input, app.Deps{
		FetcherFactory: func(id string, useIntl bool) (fetcher.Fetcher, error) {
			factory := fetcher.NewFactory(client, workCfg.Config, logger)
			return factory.Create(id, useIntl)
		},
		HTTPClient: client,
		Logger:     logger,
	})
	if err != nil {
		t.Fatalf("GetVideoInfo failed: %v", err)
	}

	if vInfo.Title == "" {
		t.Error("expected non-empty video title")
	}
	if len(vInfo.PagesInfo) == 0 {
		t.Error("expected at least one page")
	}
}
