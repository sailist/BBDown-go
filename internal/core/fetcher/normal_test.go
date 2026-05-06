package fetcher

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/nilaonai/bbdown-go/internal/config"
	"github.com/nilaonai/bbdown-go/internal/core/entity"
	"github.com/nilaonai/bbdown-go/pkg/httpclient"
)

type testHTTPClient struct {
	server *httptest.Server
}

func (c *testHTTPClient) Get(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	var path string
	switch {
	case strings.Contains(url, "x/web-interface/view"):
		idx := strings.Index(url, "?")
		if idx >= 0 {
			path = "/x/web-interface/view?" + url[idx+1:]
		} else {
			path = "/x/web-interface/view"
		}
	case strings.Contains(url, "x/player.so"):
		idx := strings.Index(url, "?")
		if idx >= 0 {
			path = "/x/player.so?" + url[idx+1:]
		} else {
			path = "/x/player.so"
		}
	case strings.Contains(url, "x/stein/edgeinfo_v2"):
		idx := strings.Index(url, "?")
		if idx >= 0 {
			path = "/x/stein/edgeinfo_v2?" + url[idx+1:]
		} else {
			path = "/x/stein/edgeinfo_v2"
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

func (c *testHTTPClient) Post(ctx context.Context, url string, body []byte, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *testHTTPClient) Head(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *testHTTPClient) GetRedirectLocation(ctx context.Context, url string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (c *testHTTPClient) GetContentLength(ctx context.Context, url string) (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestNormalFetcher_SinglePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/x/web-interface/view") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"data": {
					"title": "Test Video",
					"desc": "A test description",
					"pic": "http://example.com/pic.jpg",
					"pubdate": 1609459200,
					"bvid": "BV1test",
					"cid": 12345,
					"owner": {"mid": 999, "name": "TestOwner"},
					"rights": {"is_stein_gate": 0},
					"pages": [
						{"page": 1, "cid": 12345, "part": "P1", "duration": 120, "dimension": {"width": 1920, "height": 1080}}
					]
				}
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &testHTTPClient{server: server}
	cfg := config.NewConfig()
	logger := newTestLogger()
	fetcher := NewNormalFetcher(client, cfg, logger)

	vInfo, err := fetcher.Fetch(context.Background(), "123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vInfo.Title != "Test Video" {
		t.Errorf("expected title 'Test Video', got %q", vInfo.Title)
	}
	if vInfo.Desc != "A test description" {
		t.Errorf("expected desc 'A test description', got %q", vInfo.Desc)
	}
	if vInfo.PubTime != 1609459200 {
		t.Errorf("expected pubTime 1609459200, got %d", vInfo.PubTime)
	}
	if vInfo.IsBangumi {
		t.Error("expected IsBangumi false")
	}
	if vInfo.IsSteinGate {
		t.Error("expected IsSteinGate false")
	}
	if len(vInfo.PagesInfo) != 1 {
		t.Fatalf("expected 1 page, got %d", len(vInfo.PagesInfo))
	}
	p := vInfo.PagesInfo[0]
	if p.Index != 1 {
		t.Errorf("expected page index 1, got %d", p.Index)
	}
	if p.Aid != "123" {
		t.Errorf("expected aid '123', got %q", p.Aid)
	}
	if p.Cid != "12345" {
		t.Errorf("expected cid '12345', got %q", p.Cid)
	}
	if p.Title != "P1" {
		t.Errorf("expected title 'P1', got %q", p.Title)
	}
	if p.Dur != 120 {
		t.Errorf("expected duration 120, got %d", p.Dur)
	}
	if p.Res != "1920x1080" {
		t.Errorf("expected res '1920x1080', got %q", p.Res)
	}
	if p.OwnerName != "TestOwner" {
		t.Errorf("expected owner name 'TestOwner', got %q", p.OwnerName)
	}
	if p.OwnerMid != "999" {
		t.Errorf("expected owner mid '999', got %q", p.OwnerMid)
	}
}

func TestNormalFetcher_MultiPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/x/web-interface/view") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"data": {
					"title": "Multi P Video",
					"desc": "multi desc",
					"pic": "http://example.com/pic2.jpg",
					"pubdate": 1609459201,
					"bvid": "BV1multi",
					"cid": 111,
					"owner": {"mid": 100, "name": "Owner"},
					"rights": {"is_stein_gate": 0},
					"pages": [
						{"page": 1, "cid": 111, "part": "Part 1", "duration": 100, "dimension": {"width": 1280, "height": 720}},
						{"page": 2, "cid": 222, "part": "Part 2", "duration": 200, "dimension": {"width": 1920, "height": 1080}}
					]
				}
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &testHTTPClient{server: server}
	cfg := config.NewConfig()
	logger := newTestLogger()
	fetcher := NewNormalFetcher(client, cfg, logger)

	vInfo, err := fetcher.Fetch(context.Background(), "456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(vInfo.PagesInfo) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(vInfo.PagesInfo))
	}
	if vInfo.PagesInfo[0].Index != 1 {
		t.Errorf("expected page 1 index 1, got %d", vInfo.PagesInfo[0].Index)
	}
	if vInfo.PagesInfo[1].Index != 2 {
		t.Errorf("expected page 2 index 2, got %d", vInfo.PagesInfo[1].Index)
	}
	if vInfo.PagesInfo[1].Cid != "222" {
		t.Errorf("expected page 2 cid '222', got %q", vInfo.PagesInfo[1].Cid)
	}
	if vInfo.PagesInfo[1].Res != "1920x1080" {
		t.Errorf("expected page 2 res '1920x1080', got %q", vInfo.PagesInfo[1].Res)
	}
}

func TestNormalFetcher_SteinGate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/x/web-interface/view"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"data": {
					"title": "SteinGate Video",
					"desc": "interactive desc",
					"pic": "http://example.com/pic3.jpg",
					"pubdate": 1609459202,
					"bvid": "BV1stein",
					"cid": 333,
					"owner": {"mid": 200, "name": "InteractiveOwner"},
					"rights": {"is_stein_gate": 1},
					"pages": [
						{"page": 1, "cid": 333, "part": "Main", "duration": 300, "dimension": {"width": 1920, "height": 1080}}
					]
				}
			}`))
		case strings.HasPrefix(r.URL.Path, "/x/player.so"):
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<interaction>{"graph_version": 42}</interaction>`))
		case strings.HasPrefix(r.URL.Path, "/x/stein/edgeinfo_v2"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"data": {
					"edges": {
						"questions": [
							{
								"choices": [
									{"cid": 444, "option": "Choice A"},
									{"cid": 555, "option": "Choice B"}
								]
							}
						]
					}
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &testHTTPClient{server: server}
	cfg := config.NewConfig()
	logger := newTestLogger()
	fetcher := NewNormalFetcher(client, cfg, logger)

	vInfo, err := fetcher.Fetch(context.Background(), "789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !vInfo.IsSteinGate {
		t.Error("expected IsSteinGate true")
	}
	if len(vInfo.PagesInfo) != 3 {
		t.Fatalf("expected 3 pages (1 base + 2 choices), got %d", len(vInfo.PagesInfo))
	}

	// base page
	if vInfo.PagesInfo[0].Index != 1 {
		t.Errorf("expected base page index 1, got %d", vInfo.PagesInfo[0].Index)
	}

	// choice pages start at index 2
	if vInfo.PagesInfo[1].Index != 2 {
		t.Errorf("expected choice 1 index 2, got %d", vInfo.PagesInfo[1].Index)
	}
	if vInfo.PagesInfo[1].Cid != "444" {
		t.Errorf("expected choice 1 cid '444', got %q", vInfo.PagesInfo[1].Cid)
	}
	if vInfo.PagesInfo[1].Title != "Choice A" {
		t.Errorf("expected choice 1 title 'Choice A', got %q", vInfo.PagesInfo[1].Title)
	}

	if vInfo.PagesInfo[2].Index != 3 {
		t.Errorf("expected choice 2 index 3, got %d", vInfo.PagesInfo[2].Index)
	}
	if vInfo.PagesInfo[2].Cid != "555" {
		t.Errorf("expected choice 2 cid '555', got %q", vInfo.PagesInfo[2].Cid)
	}
	if vInfo.PagesInfo[2].Title != "Choice B" {
		t.Errorf("expected choice 2 title 'Choice B', got %q", vInfo.PagesInfo[2].Title)
	}
}

func TestNormalFetcher_BangumiRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/x/web-interface/view") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"data": {
					"title": "Bangumi Video",
					"desc": "bangumi desc",
					"pic": "http://example.com/pic4.jpg",
					"pubdate": 1609459203,
					"bvid": "BV1bangu",
					"cid": 666,
					"redirect_url": "https://www.bilibili.com/bangumi/play/ep12345",
					"owner": {"mid": 300, "name": "BangumiOwner"},
					"rights": {"is_stein_gate": 0},
					"pages": [
						{"page": 1, "cid": 666, "part": "EP1", "duration": 1500, "dimension": {"width": 1920, "height": 1080}}
					]
				}
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &testHTTPClient{server: server}
	cfg := config.NewConfig()
	logger := newTestLogger()
	fetcher := NewNormalFetcher(client, cfg, logger)

	vInfo, err := fetcher.Fetch(context.Background(), "999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !vInfo.IsBangumi {
		t.Error("expected IsBangumi true")
	}
	if len(vInfo.PagesInfo) != 1 {
		t.Fatalf("expected 1 page, got %d", len(vInfo.PagesInfo))
	}
	if vInfo.PagesInfo[0].Epid != "12345" {
		t.Errorf("expected epid '12345', got %q", vInfo.PagesInfo[0].Epid)
	}
}

func TestNormalFetcher_SteinGate_MissingInteraction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/x/web-interface/view"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"data": {
					"title": "Bad Stein",
					"desc": "bad",
					"pic": "",
					"pubdate": 0,
					"bvid": "BV1bad",
					"cid": 777,
					"owner": {"mid": 0, "name": ""},
					"rights": {"is_stein_gate": 1},
					"pages": [
						{"page": 1, "cid": 777, "part": "P1", "duration": 60, "dimension": {"width": 0, "height": 0}}
					]
				}
			}`))
		case strings.HasPrefix(r.URL.Path, "/x/player.so"):
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<some>other content</some>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &testHTTPClient{server: server}
	cfg := config.NewConfig()
	logger := newTestLogger()
	fetcher := NewNormalFetcher(client, cfg, logger)

	_, err := fetcher.Fetch(context.Background(), "000")
	if err == nil {
		t.Fatal("expected error for missing interaction node")
	}
	if !strings.Contains(err.Error(), "互动视频获取分P信息失败") {
		t.Errorf("expected error to contain '互动视频获取分P信息失败', got: %v", err)
	}
}

func TestNormalFetcher_EmptyInteraction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/x/web-interface/view"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"data": {
					"title": "Empty",
					"desc": "",
					"pic": "",
					"pubdate": 0,
					"bvid": "BV1empty",
					"cid": 888,
					"owner": {"mid": 0, "name": ""},
					"rights": {"is_stein_gate": 1},
					"pages": [
						{"page": 1, "cid": 888, "part": "P1", "duration": 10, "dimension": {"width": 0, "height": 0}}
					]
				}
			}`))
		case strings.HasPrefix(r.URL.Path, "/x/player.so"):
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<interaction></interaction>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &testHTTPClient{server: server}
	cfg := config.NewConfig()
	logger := newTestLogger()
	fetcher := NewNormalFetcher(client, cfg, logger)

	_, err := fetcher.Fetch(context.Background(), "111")
	if err == nil {
		t.Fatal("expected error for empty interaction")
	}
	if !strings.Contains(err.Error(), "互动视频获取分P信息失败") {
		t.Errorf("expected error to contain '互动视频获取分P信息失败', got: %v", err)
	}
}

func TestNormalFetcher_PageEqualContract(t *testing.T) {
	// Ensure the pages produced by NormalFetcher can be compared with Equal
	page1 := entity.Page{Index: 1, Aid: "123", Cid: "456", Epid: "", Title: "T1"}
	page2 := entity.Page{Index: 1, Aid: "123", Cid: "456", Epid: "", Title: "T2"}
	if !page1.Equal(page2) {
		t.Error("expected pages with same aid/cid/epid to be equal")
	}

	page3 := entity.Page{Index: 1, Aid: "123", Cid: "456", Epid: "789", Title: "T1"}
	if page1.Equal(page3) {
		t.Error("expected pages with different epid to not be equal")
	}
}
