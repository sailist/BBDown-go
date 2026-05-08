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

type cheeseTestClient struct {
	server *httptest.Server
}

func (c *cheeseTestClient) Get(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	var path string
	switch {
	case strings.Contains(url, "pugv/view/web/season"):
		idx := strings.Index(url, "?")
		if idx >= 0 {
			path = "/pugv/view/web/season?" + url[idx+1:]
		} else {
			path = "/pugv/view/web/season"
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

func (c *cheeseTestClient) Post(ctx context.Context, url string, body []byte, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *cheeseTestClient) Head(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *cheeseTestClient) GetRedirectLocation(ctx context.Context, url string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (c *cheeseTestClient) GetContentLength(ctx context.Context, url string) (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

func TestCheeseFetcher_Normal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/pugv/view/web/season") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"data": {
					"cover": "http://example.com/cheese.jpg",
					"title": "Test Cheese",
					"subtitle": "Learn something",
					"up_info": {"uname": "Teacher", "mid": 12345},
					"episodes": [
						{"index": 1, "aid": 1001, "cid": 2001, "id": 3001, "title": "Lesson 1", "duration": 600, "release_date": 1609459200},
						{"index": 2, "aid": 1002, "cid": 2002, "id": 3002, "title": "Lesson 2", "duration": 900, "release_date": 1609545600}
					]
				}
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &cheeseTestClient{server: server}
	cfg := config.NewConfig()
	logger := newTestLogger()
	fetcher := NewCheeseFetcher(client, cfg, logger)

	vInfo, err := fetcher.Fetch(context.Background(), "cheese:3001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vInfo.Title != "Test Cheese" {
		t.Errorf("expected title 'Test Cheese', got %q", vInfo.Title)
	}
	if vInfo.Desc != "Learn something" {
		t.Errorf("expected desc 'Learn something', got %q", vInfo.Desc)
	}
	if vInfo.Pic != "http://example.com/cheese.jpg" {
		t.Errorf("expected pic 'http://example.com/cheese.jpg', got %q", vInfo.Pic)
	}
	if vInfo.PubTime != 1609459200 {
		t.Errorf("expected pubTime 1609459200, got %d", vInfo.PubTime)
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
	if p1.Title != "Lesson 1" {
		t.Errorf("expected title 'Lesson 1', got %q", p1.Title)
	}
	if p1.Dur != 600 {
		t.Errorf("expected duration 600, got %d", p1.Dur)
	}
	if p1.OwnerName != "Teacher" {
		t.Errorf("expected owner name 'Teacher', got %q", p1.OwnerName)
	}
	if p1.OwnerMid != "12345" {
		t.Errorf("expected owner mid '12345', got %q", p1.OwnerMid)
	}

	p2 := vInfo.PagesInfo[1]
	if p2.Index != 2 {
		t.Errorf("expected page 2 index 2, got %d", p2.Index)
	}
	if p2.Epid != "3002" {
		t.Errorf("expected epid '3002', got %q", p2.Epid)
	}
}

func TestCheeseFetcher_EmptyEpisodes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/pugv/view/web/season") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"data": {
					"cover": "http://example.com/empty.jpg",
					"title": "Empty Cheese",
					"subtitle": "No lessons",
					"up_info": {"uname": "Nobody", "mid": 0},
					"episodes": []
				}
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &cheeseTestClient{server: server}
	cfg := config.NewConfig()
	logger := newTestLogger()
	fetcher := NewCheeseFetcher(client, cfg, logger)

	vInfo, err := fetcher.Fetch(context.Background(), "cheese:9999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(vInfo.PagesInfo) != 0 {
		t.Fatalf("expected 0 pages, got %d", len(vInfo.PagesInfo))
	}
	if vInfo.PubTime != 0 {
		t.Errorf("expected pubTime 0, got %d", vInfo.PubTime)
	}
	if vInfo.Title != "Empty Cheese" {
		t.Errorf("expected title 'Empty Cheese', got %q", vInfo.Title)
	}
}
