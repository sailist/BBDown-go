package fetcher

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nilaonai/bbdown-go/internal/config"
	"github.com/nilaonai/bbdown-go/pkg/httpclient"
)

type intlBangumiTestClient struct {
	server *httptest.Server
}

func (c *intlBangumiTestClient) Get(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	var path string
	switch {
	case strings.Contains(url, "intl/gateway/v2/ogv/view/app/season"):
		idx := strings.Index(url, "?")
		if idx >= 0 {
			path = "/intl/gateway/v2/ogv/view/app/season?" + url[idx+1:]
		} else {
			path = "/intl/gateway/v2/ogv/view/app/season"
		}
	case strings.Contains(url, "bangumi.bilibili.com/anime/"):
		idx := strings.Index(url, "/anime/")
		if idx >= 0 {
			path = url[idx:]
		} else {
			path = "/anime/"
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

func (c *intlBangumiTestClient) Post(ctx context.Context, url string, body []byte, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *intlBangumiTestClient) Head(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *intlBangumiTestClient) GetRedirectLocation(ctx context.Context, url string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (c *intlBangumiTestClient) GetContentLength(ctx context.Context, url string) (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

func TestIntlBangumiFetcher_Normal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/intl/gateway/v2/ogv/view/app/season") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"result": {
					"season_id": "12345",
					"cover": "http://example.com/cover.jpg",
					"title": "Test Intl Bangumi",
					"evaluate": "A great intl bangumi",
					"publish": {"pub_time": "2021-01-01 12:00:00"},
					"episodes": [
						{"badge": "", "dimension": {"width": 1920, "height": 1080}, "title": "EP1", "long_title": "First Episode", "aid": 1001, "cid": 2001, "id": 3001, "pub_time": 1609459200},
						{"badge": "", "dimension": {"width": 1280, "height": 720}, "title": "EP2", "long_title": "Second Episode", "aid": 1002, "cid": 2002, "id": 3002, "pub_time": 1609545600}
					]
				}
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &intlBangumiTestClient{server: server}
	cfg := config.NewConfig()
	logger := newTestLogger()
	fetcher := NewIntlBangumiFetcher(client, cfg, logger)

	vInfo, err := fetcher.Fetch(context.Background(), "ep:3001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vInfo.Title != "Test Intl Bangumi" {
		t.Errorf("expected title 'Test Intl Bangumi', got %q", vInfo.Title)
	}
	if vInfo.Desc != "A great intl bangumi" {
		t.Errorf("expected desc 'A great intl bangumi', got %q", vInfo.Desc)
	}
	if vInfo.Pic != "http://example.com/cover.jpg" {
		t.Errorf("expected pic 'http://example.com/cover.jpg', got %q", vInfo.Pic)
	}
	if vInfo.PubTime != 1609502400 {
		t.Errorf("expected pubTime 1609502400, got %d", vInfo.PubTime)
	}
	if !vInfo.IsBangumi {
		t.Error("expected IsBangumi true")
	}
	if !vInfo.IsCheese {
		t.Error("expected IsCheese true")
	}
	if vInfo.Index != "1" {
		t.Errorf("expected index '1', got %q", vInfo.Index)
	}
	if len(vInfo.PagesInfo) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(vInfo.PagesInfo))
	}

	p1 := vInfo.PagesInfo[0]
	if p1.Index != 1 {
		t.Errorf("expected page 1 index 1, got %d", p1.Index)
	}
	if p1.Aid != "1001" {
		t.Errorf("expected aid '1001', got %q", p1.Aid)
	}
	if p1.Cid != "2001" {
		t.Errorf("expected cid '2001', got %q", p1.Cid)
	}
	if p1.Epid != "3001" {
		t.Errorf("expected epid '3001', got %q", p1.Epid)
	}
	if p1.Title != "EP1 First Episode" {
		t.Errorf("expected title 'EP1 First Episode', got %q", p1.Title)
	}
	if p1.Res != "1920x1080" {
		t.Errorf("expected res '1920x1080', got %q", p1.Res)
	}

	p2 := vInfo.PagesInfo[1]
	if p2.Index != 2 {
		t.Errorf("expected page 2 index 2, got %d", p2.Index)
	}
	if p2.Epid != "3002" {
		t.Errorf("expected epid '3002', got %q", p2.Epid)
	}
}

func TestIntlBangumiFetcher_EmptyCoverFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/intl/gateway/v2/ogv/view/app/season"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"result": {
					"season_id": "12345",
					"cover": "",
					"title": "Fallback Title",
					"evaluate": "Fallback Desc",
					"publish": {"pub_time": ""},
					"episodes": [
						{"badge": "", "dimension": {"width": 0, "height": 0}, "title": "EP1", "long_title": "First", "aid": 1001, "cid": 2001, "id": 3001, "pub_time": 0}
					]
				}
			}`))
			return
		case strings.HasPrefix(r.URL.Path, "/anime/"):
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><script>window.__INITIAL_STATE__={"mediaInfo":{"cover":"http://example.com/fallback.jpg","title":"Anime Title","evaluate":"Anime Desc"}};(function(){}</script></html>`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &intlBangumiTestClient{server: server}
	cfg := config.NewConfig()
	logger := newTestLogger()
	fetcher := NewIntlBangumiFetcher(client, cfg, logger)

	vInfo, err := fetcher.Fetch(context.Background(), "ep:3001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vInfo.Pic != "http://example.com/fallback.jpg" {
		t.Errorf("expected pic 'http://example.com/fallback.jpg', got %q", vInfo.Pic)
	}
	if vInfo.Title != "Anime Title" {
		t.Errorf("expected title 'Anime Title', got %q", vInfo.Title)
	}
	if vInfo.Desc != "Anime Desc" {
		t.Errorf("expected desc 'Anime Desc', got %q", vInfo.Desc)
	}
}

func TestIntlBangumiFetcher_ModulesFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/intl/gateway/v2/ogv/view/app/season") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"result": {
					"season_id": "12345",
					"cover": "http://example.com/cover.jpg",
					"title": "Modules Bangumi",
					"evaluate": "With modules",
					"publish": {"pub_time": "2022-06-01 00:00:00"},
					"episodes": [
						{"badge": "", "dimension": {"width": 0, "height": 0}, "title": "Main", "long_title": "Main Story", "aid": 1001, "cid": 2001, "id": 3001, "pub_time": 0}
					],
					"modules": [
						{
							"data": {
								"episodes": [
									{"badge": "", "link": "https://www.bilibili.com/bangumi/play/ep4003", "uri": "/4003", "dimension": {"width": 1920, "height": 1080}, "title": "SP1", "long_title": "Special Episode", "aid": 1003, "cid": 2003, "id": 4003, "pub_time": 1654041600}
								]
							}
						}
					]
				}
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &intlBangumiTestClient{server: server}
	cfg := config.NewConfig()
	logger := newTestLogger()
	fetcher := NewIntlBangumiFetcher(client, cfg, logger)

	vInfo, err := fetcher.Fetch(context.Background(), "ep:4003")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vInfo.Title != "Modules Bangumi" {
		t.Errorf("expected title 'Modules Bangumi', got %q", vInfo.Title)
	}
	if len(vInfo.PagesInfo) != 1 {
		t.Fatalf("expected 1 page, got %d", len(vInfo.PagesInfo))
	}
	if vInfo.PagesInfo[0].Epid != "4003" {
		t.Errorf("expected epid '4003', got %q", vInfo.PagesInfo[0].Epid)
	}
	if vInfo.PagesInfo[0].Title != "SP1 Special Episode" {
		t.Errorf("expected title 'SP1 Special Episode', got %q", vInfo.PagesInfo[0].Title)
	}
	if vInfo.PagesInfo[0].Res != "1920x1080" {
		t.Errorf("expected res '1920x1080', got %q", vInfo.PagesInfo[0].Res)
	}
	if vInfo.Index != "1" {
		t.Errorf("expected index '1', got %q", vInfo.Index)
	}
}

func TestIntlBangumiFetcher_SkipTrailer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/intl/gateway/v2/ogv/view/app/season") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"result": {
					"season_id": "12345",
					"cover": "http://example.com/cover.jpg",
					"title": "Trailer Bangumi",
					"evaluate": "With trailer",
					"publish": {"pub_time": "2023-01-01 00:00:00"},
					"episodes": [
						{"badge": "预告", "dimension": {"width": 1920, "height": 1080}, "title": "Trailer", "long_title": "Coming Soon", "aid": 1001, "cid": 2001, "id": 3001, "pub_time": 1672531200},
						{"badge": "", "dimension": {"width": 1920, "height": 1080}, "title": "EP1", "long_title": "First", "aid": 1002, "cid": 2002, "id": 3002, "pub_time": 1672617600}
					]
				}
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &intlBangumiTestClient{server: server}
	cfg := config.NewConfig()
	logger := newTestLogger()
	fetcher := NewIntlBangumiFetcher(client, cfg, logger)

	vInfo, err := fetcher.Fetch(context.Background(), "ep:3002")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(vInfo.PagesInfo) != 1 {
		t.Fatalf("expected 1 page, got %d", len(vInfo.PagesInfo))
	}
	if vInfo.PagesInfo[0].Title != "EP1 First" {
		t.Errorf("expected title 'EP1 First', got %q", vInfo.PagesInfo[0].Title)
	}
	if vInfo.PagesInfo[0].Index != 1 {
		t.Errorf("expected index 1, got %d", vInfo.PagesInfo[0].Index)
	}
}
