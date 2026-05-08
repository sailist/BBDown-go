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

type favListTestClient struct {
	server *httptest.Server
}

func (c *favListTestClient) Get(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	var path string
	switch {
	case strings.Contains(url, "x/v3/fav/folder/created/list-all"):
		idx := strings.Index(url, "?")
		if idx >= 0 {
			path = "/x/v3/fav/folder/created/list-all?" + url[idx+1:]
		} else {
			path = "/x/v3/fav/folder/created/list-all"
		}
	case strings.Contains(url, "x/v3/fav/resource/list"):
		idx := strings.Index(url, "?")
		if idx >= 0 {
			path = "/x/v3/fav/resource/list?" + url[idx+1:]
		} else {
			path = "/x/v3/fav/resource/list"
		}
	case strings.Contains(url, "x/web-interface/view"):
		idx := strings.Index(url, "?")
		if idx >= 0 {
			path = "/x/web-interface/view?" + url[idx+1:]
		} else {
			path = "/x/web-interface/view"
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

func (c *favListTestClient) Post(ctx context.Context, url string, body []byte, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *favListTestClient) Head(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *favListTestClient) GetRedirectLocation(ctx context.Context, url string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (c *favListTestClient) GetContentLength(ctx context.Context, url string) (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

func TestFavListFetcher_Normal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/x/v3/fav/resource/list"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"code": 0,
				"message": "0",
				"data": {
					"info": {
						"title": "Test Fav List",
						"intro": "A test fav list",
						"ctime": 1609459200,
						"media_count": 1,
						"upper": {"name": "FavOwner"}
					},
					"medias": [
						{
							"attr": 0,
							"id": 1001,
							"title": "Fav Video 1",
							"intro": "Intro 1",
							"page": 1,
							"duration": 120,
							"pubtime": 1609459200,
							"cover": "http://example.com/cover1.jpg",
							"ugc": {"first_cid": 2001},
							"upper": {"name": "Uploader1", "mid": 101}
						}
					]
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &favListTestClient{server: server}
	cfg := config.NewConfig()
	logger := newTestLogger()
	fetcher := NewFavListFetcher(client, cfg, logger)

	vInfo, err := fetcher.Fetch(context.Background(), "favId:123:456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vInfo.Title != "Test Fav List" {
		t.Errorf("expected title 'Test Fav List', got %q", vInfo.Title)
	}
	if vInfo.Desc != "A test fav list" {
		t.Errorf("expected desc 'A test fav list', got %q", vInfo.Desc)
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
	if p.Title != "Fav Video 1" {
		t.Errorf("expected title 'Fav Video 1', got %q", p.Title)
	}
	if p.Dur != 120 {
		t.Errorf("expected duration 120, got %d", p.Dur)
	}
	if p.OwnerName != "Uploader1" {
		t.Errorf("expected owner name 'Uploader1', got %q", p.OwnerName)
	}
	if p.OwnerMid != "101" {
		t.Errorf("expected owner mid '101', got %q", p.OwnerMid)
	}
}

func TestFavListFetcher_DefaultFavId(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/x/v3/fav/folder/created/list-all"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"code": 0,
				"message": "0",
				"data": {
					"list": [
						{"id": 999}
					]
				}
			}`))
		case strings.HasPrefix(r.URL.Path, "/x/v3/fav/resource/list"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"code": 0,
				"message": "0",
				"data": {
					"info": {
						"title": "Default Fav",
						"intro": "",
						"ctime": 1609459200,
						"media_count": 1,
						"upper": {"name": "Owner"}
					},
					"medias": [
						{
							"attr": 0,
							"id": 1001,
							"title": "Default Video",
							"intro": "",
							"page": 1,
							"duration": 60,
							"pubtime": 1609459200,
							"cover": "",
							"ugc": {"first_cid": 2001},
							"upper": {"name": "Uploader", "mid": 101}
						}
					]
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &favListTestClient{server: server}
	cfg := config.NewConfig()
	logger := newTestLogger()
	fetcher := NewFavListFetcher(client, cfg, logger)

	vInfo, err := fetcher.Fetch(context.Background(), "favId::456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vInfo.Title != "Default Fav" {
		t.Errorf("expected title 'Default Fav', got %q", vInfo.Title)
	}
	if len(vInfo.PagesInfo) != 1 {
		t.Fatalf("expected 1 page, got %d", len(vInfo.PagesInfo))
	}
	if vInfo.PagesInfo[0].Aid != "1001" {
		t.Errorf("expected aid '1001', got %q", vInfo.PagesInfo[0].Aid)
	}
}

func TestFavListFetcher_MultiPageVideo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/x/v3/fav/resource/list"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"code": 0,
				"message": "0",
				"data": {
					"info": {
						"title": "Fav With Multi",
						"intro": "",
						"ctime": 1609459200,
						"media_count": 1,
						"upper": {"name": "Owner"}
					},
					"medias": [
						{
							"attr": 0,
							"id": 1001,
							"title": "Multi P Fav",
							"intro": "Multi Intro",
							"page": 2,
							"duration": 300,
							"pubtime": 1609459200,
							"cover": "http://example.com/cover.jpg",
							"ugc": {"first_cid": 2001},
							"upper": {"name": "Uploader", "mid": 101}
						}
					]
				}
			}`))
		case strings.HasPrefix(r.URL.Path, "/x/web-interface/view"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"data": {
					"title": "Multi P Video",
					"desc": "desc",
					"pic": "http://example.com/pic.jpg",
					"pubdate": 1609459200,
					"bvid": "BV1test",
					"cid": 111,
					"owner": {"mid": 101, "name": "Uploader"},
					"rights": {"is_stein_gate": 0},
					"pages": [
						{"page": 1, "cid": 111, "part": "Part 1", "duration": 100, "dimension": {"width": 1280, "height": 720}},
						{"page": 2, "cid": 222, "part": "Part 2", "duration": 200, "dimension": {"width": 1920, "height": 1080}}
					]
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &favListTestClient{server: server}
	cfg := config.NewConfig()
	logger := newTestLogger()
	fetcher := NewFavListFetcher(client, cfg, logger)

	vInfo, err := fetcher.Fetch(context.Background(), "favId:123:456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(vInfo.PagesInfo) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(vInfo.PagesInfo))
	}
	if vInfo.PagesInfo[0].Title != "Multi P Fav_P1_Part 1" {
		t.Errorf("expected title 'Multi P Fav_P1_Part 1', got %q", vInfo.PagesInfo[0].Title)
	}
	if vInfo.PagesInfo[1].Title != "Multi P Fav_P2_Part 2" {
		t.Errorf("expected title 'Multi P Fav_P2_Part 2', got %q", vInfo.PagesInfo[1].Title)
	}
	if vInfo.PagesInfo[0].Cover != "http://example.com/pic.jpg" {
		t.Errorf("expected cover 'http://example.com/pic.jpg', got %q", vInfo.PagesInfo[0].Cover)
	}
	if vInfo.PagesInfo[0].Desc != "Multi Intro" {
		t.Errorf("expected desc 'Multi Intro', got %q", vInfo.PagesInfo[0].Desc)
	}
}
