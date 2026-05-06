package fetcher

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/nilaonai/bbdown-go/internal/config"
	"github.com/nilaonai/bbdown-go/pkg/httpclient"
)

type spaceVideoTestClient struct {
	server *httptest.Server
}

func (c *spaceVideoTestClient) Get(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	var path string
	switch {
	case strings.Contains(url, "live_user/v1/Master/info"):
		idx := strings.Index(url, "?")
		if idx >= 0 {
			path = "/live_user/v1/Master/info?" + url[idx+1:]
		} else {
			path = "/live_user/v1/Master/info"
		}
	case strings.Contains(url, "x/space/wbi/arc/search"):
		idx := strings.Index(url, "?")
		if idx >= 0 {
			path = "/x/space/wbi/arc/search?" + url[idx+1:]
		} else {
			path = "/x/space/wbi/arc/search"
		}
	default:
		return nil, fmt.Errorf("unexpected url: %s", url)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.server.URL+path, nil)
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		opt(req)
	}
	return http.DefaultClient.Do(req)
}

func (c *spaceVideoTestClient) Post(ctx context.Context, url string, body []byte, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *spaceVideoTestClient) Head(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *spaceVideoTestClient) GetRedirectLocation(ctx context.Context, url string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (c *spaceVideoTestClient) GetContentLength(ctx context.Context, url string) (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

func TestSpaceVideoFetcher_WritesUrlsAndReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/live_user/v1/Master/info"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"code": 0,
				"message": "0",
				"data": {
					"info": {
						"uname": "TestUser"
					}
				}
			}`))
		case strings.HasPrefix(r.URL.Path, "/x/space/wbi/arc/search"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"code": 0,
				"message": "0",
				"data": {
					"list": {
						"vlist": [
							{"aid": 1001},
							{"aid": 1002}
						]
					},
					"page": {
						"count": 2
					}
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &spaceVideoTestClient{server: server}
	cfg := config.NewConfig()
	logger := newTestLogger()
	fetcher := NewSpaceVideoFetcher(client, cfg, logger)

	_, err := fetcher.Fetch(context.Background(), "mid:123")
	if err == nil {
		t.Fatal("expected error for space video fetcher")
	}
	if !strings.Contains(err.Error(), "暂不支持该功能") {
		t.Errorf("expected error to contain '暂不支持该功能', got: %v", err)
	}

	fileName := "TestUser的投稿视频.txt"
	defer os.Remove(fileName)

	content, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatalf("expected file to be written: %v", err)
	}

	expected := "https://www.bilibili.com/video/av1001\nhttps://www.bilibili.com/video/av1002"
	if string(content) != expected {
		t.Errorf("expected file content %q, got %q", expected, string(content))
	}
}

func TestSpaceVideoFetcher_InvalidFileName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/live_user/v1/Master/info"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"code": 0,
				"message": "0",
				"data": {
					"info": {
						"uname": "User/Name"
					}
				}
			}`))
		case strings.HasPrefix(r.URL.Path, "/x/space/wbi/arc/search"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"code": 0,
				"message": "0",
				"data": {
					"list": {
						"vlist": [
							{"aid": 1001}
						]
					},
					"page": {
						"count": 1
					}
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &spaceVideoTestClient{server: server}
	cfg := config.NewConfig()
	logger := newTestLogger()
	fetcher := NewSpaceVideoFetcher(client, cfg, logger)

	_, err := fetcher.Fetch(context.Background(), "mid:123")
	if err == nil {
		t.Fatal("expected error for space video fetcher")
	}

	fileName := "User.Name的投稿视频.txt"
	defer os.Remove(fileName)

	if _, err := os.Stat(fileName); err != nil {
		t.Fatalf("expected file %s to exist: %v", fileName, err)
	}
}
