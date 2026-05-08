package fetcher

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sailist/BBDown-go/internal/config"
	"github.com/sailist/BBDown-go/pkg/httpclient"
)

type mediaListTestClient struct {
	server *httptest.Server
}

func (c *mediaListTestClient) Get(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	var path string
	switch {
	case strings.Contains(url, "x/v1/medialist/info"):
		idx := strings.Index(url, "?")
		if idx >= 0 {
			path = "/x/v1/medialist/info?" + url[idx+1:]
		} else {
			path = "/x/v1/medialist/info"
		}
	case strings.Contains(url, "x/v2/medialist/resource/list"):
		idx := strings.Index(url, "?")
		if idx >= 0 {
			path = "/x/v2/medialist/resource/list?" + url[idx+1:]
		} else {
			path = "/x/v2/medialist/resource/list"
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

func (c *mediaListTestClient) Post(ctx context.Context, url string, body []byte, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *mediaListTestClient) Head(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *mediaListTestClient) GetRedirectLocation(ctx context.Context, url string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (c *mediaListTestClient) GetContentLength(ctx context.Context, url string) (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

func TestMediaListFetcher_Normal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/x/v1/medialist/info"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"code": 0,
				"message": "0",
				"data": {
					"title": "Test Media List",
					"intro": "A test media list",
					"ctime": 1609459200
				}
			}`))
		case strings.HasPrefix(r.URL.Path, "/x/v2/medialist/resource/list"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"code": 0,
				"message": "0",
				"data": {
					"has_more": false,
					"media_list": [
						{
							"attr": 0,
							"id": 1001,
							"title": "Video 1",
							"intro": "Intro 1",
							"page": 1,
							"pubtime": 1609459200,
							"cover": "http://example.com/cover1.jpg",
							"upper": {"name": "Uploader1", "mid": 101},
							"pages": [
								{"id": 2001, "page": 1, "title": "P1", "duration": 120, "dimension": {"width": 1920, "height": 1080}}
							]
						}
					]
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &mediaListTestClient{server: server}
	cfg := config.NewConfig()
	logger := newTestLogger()
	fetcher := NewMediaListFetcher(client, cfg, logger)

	vInfo, err := fetcher.Fetch(context.Background(), "listBizId:123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vInfo.Title != "Test Media List" {
		t.Errorf("expected title 'Test Media List', got %q", vInfo.Title)
	}
	if vInfo.Desc != "A test media list" {
		t.Errorf("expected desc 'A test media list', got %q", vInfo.Desc)
	}
	if vInfo.PubTime != 1609459200 {
		t.Errorf("expected pubTime 1609459200, got %d", vInfo.PubTime)
	}
	if len(vInfo.PagesInfo) != 1 {
		t.Fatalf("expected 1 page, got %d", len(vInfo.PagesInfo))
	}
	p := vInfo.PagesInfo[0]
	if p.Index != 1 {
		t.Errorf("expected page index 1, got %d", p.Index)
	}
	if p.Aid != "1001" {
		t.Errorf("expected aid '1001', got %q", p.Aid)
	}
	if p.Cid != "2001" {
		t.Errorf("expected cid '2001', got %q", p.Cid)
	}
	if p.Title != "Video 1" {
		t.Errorf("expected title 'Video 1', got %q", p.Title)
	}
	if p.Dur != 120 {
		t.Errorf("expected duration 120, got %d", p.Dur)
	}
	if p.Res != "1920x1080" {
		t.Errorf("expected res '1920x1080', got %q", p.Res)
	}
	if p.OwnerName != "Uploader1" {
		t.Errorf("expected owner name 'Uploader1', got %q", p.OwnerName)
	}
	if p.OwnerMid != "101" {
		t.Errorf("expected owner mid '101', got %q", p.OwnerMid)
	}
}

func TestMediaListFetcher_MultiPageVideo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/x/v1/medialist/info"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"code": 0,
				"message": "0",
				"data": {
					"title": "Multi Page List",
					"intro": "List with multi-page video",
					"ctime": 1609459200
				}
			}`))
		case strings.HasPrefix(r.URL.Path, "/x/v2/medialist/resource/list"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"code": 0,
				"message": "0",
				"data": {
					"has_more": false,
					"media_list": [
						{
							"attr": 0,
							"id": 1001,
							"title": "Multi P Video",
							"intro": "Intro",
							"page": 2,
							"pubtime": 1609459200,
							"cover": "http://example.com/cover.jpg",
							"upper": {"name": "Uploader", "mid": 101},
							"pages": [
								{"id": 2001, "page": 1, "title": "Part 1", "duration": 100, "dimension": {"width": 1280, "height": 720}},
								{"id": 2002, "page": 2, "title": "Part 2", "duration": 200, "dimension": {"width": 1920, "height": 1080}}
							]
						}
					]
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &mediaListTestClient{server: server}
	cfg := config.NewConfig()
	logger := newTestLogger()
	fetcher := NewMediaListFetcher(client, cfg, logger)

	vInfo, err := fetcher.Fetch(context.Background(), "listBizId:123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(vInfo.PagesInfo) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(vInfo.PagesInfo))
	}
	if vInfo.PagesInfo[0].Title != "Multi P Video_P1_Part 1" {
		t.Errorf("expected title 'Multi P Video_P1_Part 1', got %q", vInfo.PagesInfo[0].Title)
	}
	if vInfo.PagesInfo[1].Title != "Multi P Video_P2_Part 2" {
		t.Errorf("expected title 'Multi P Video_P2_Part 2', got %q", vInfo.PagesInfo[1].Title)
	}
}

func TestMediaListFetcher_FallbackToSeries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/x/v1/medialist/info"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"code": -404,
				"message": "Not Found",
				"data": null
			}`))
		case strings.HasPrefix(r.URL.Path, "/x/v2/medialist/resource/list"):
			// This should be called by the fallback SeriesListFetcher
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"code": 0,
				"message": "0",
				"data": {
					"has_more": false,
					"media_list": [
						{
							"attr": 0,
							"id": 2001,
							"title": "Series Video",
							"intro": "Series Intro",
							"page": 1,
							"pubtime": 1609545600,
							"cover": "http://example.com/series_cover.jpg",
							"upper": {"name": "SeriesUploader", "mid": 202},
							"pages": [
								{"id": 3001, "page": 1, "title": "SP1", "duration": 180, "dimension": {"width": 1920, "height": 1080}}
							]
						}
					]
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &mediaListTestClient{server: server}
	cfg := config.NewConfig()
	logger := newTestLogger()
	fetcher := NewMediaListFetcher(client, cfg, logger)

	vInfo, err := fetcher.Fetch(context.Background(), "listBizId:123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(vInfo.PagesInfo) != 1 {
		t.Fatalf("expected 1 page, got %d", len(vInfo.PagesInfo))
	}
	if vInfo.PagesInfo[0].Aid != "2001" {
		t.Errorf("expected aid '2001', got %q", vInfo.PagesInfo[0].Aid)
	}
}

func TestMediaListFetcher_Pagination(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/x/v1/medialist/info"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"code": 0,
				"message": "0",
				"data": {
					"title": "Paginated List",
					"intro": "",
					"ctime": 1609459200
				}
			}`))
		case strings.HasPrefix(r.URL.Path, "/x/v2/medialist/resource/list"):
			callCount++
			w.Header().Set("Content-Type", "application/json")
			if callCount == 1 {
				w.Write([]byte(`{
					"code": 0,
					"message": "0",
					"data": {
						"has_more": true,
						"media_list": [
							{
								"attr": 0,
								"id": 1001,
								"title": "Video 1",
								"intro": "",
								"page": 1,
								"pubtime": 1609459200,
								"cover": "",
								"upper": {"name": "Uploader", "mid": 101},
								"pages": [
									{"id": 2001, "page": 1, "title": "P1", "duration": 60, "dimension": {"width": 0, "height": 0}}
								]
							}
						]
					}
				}`))
			} else {
				w.Write([]byte(`{
					"code": 0,
					"message": "0",
					"data": {
						"has_more": false,
						"media_list": [
							{
								"attr": 0,
								"id": 1002,
								"title": "Video 2",
								"intro": "",
								"page": 1,
								"pubtime": 1609545600,
								"cover": "",
								"upper": {"name": "Uploader", "mid": 101},
								"pages": [
									{"id": 2002, "page": 1, "title": "P1", "duration": 90, "dimension": {"width": 0, "height": 0}}
								]
							}
						]
					}
				}`))
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &mediaListTestClient{server: server}
	cfg := config.NewConfig()
	logger := newTestLogger()
	fetcher := NewMediaListFetcher(client, cfg, logger)

	vInfo, err := fetcher.Fetch(context.Background(), "listBizId:123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(vInfo.PagesInfo) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(vInfo.PagesInfo))
	}
	if vInfo.PagesInfo[0].Aid != "1001" {
		t.Errorf("expected aid '1001', got %q", vInfo.PagesInfo[0].Aid)
	}
	if vInfo.PagesInfo[1].Aid != "1002" {
		t.Errorf("expected aid '1002', got %q", vInfo.PagesInfo[1].Aid)
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
}
