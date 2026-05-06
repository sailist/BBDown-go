package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/nilaonai/bbdown-go/internal/cli"
	"github.com/nilaonai/bbdown-go/internal/core/entity"
	"github.com/nilaonai/bbdown-go/internal/core/fetcher"
	"github.com/nilaonai/bbdown-go/pkg/httpclient"
)

// mockFetcher is a test double for fetcher.Fetcher.
type mockFetcher struct {
	vInfo *entity.VInfo
	err   error
}

func (m *mockFetcher) Fetch(ctx context.Context, id string) (*entity.VInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.vInfo, nil
}

func TestGetVideoInfo_NormalVideo(t *testing.T) {
	opt := cli.NewOption()
	input := "BV1xx411c7mD"

	vInfo := &entity.VInfo{
		Title:   "Test Video",
		PubTime: 1609459200,
		PagesInfo: []entity.Page{
			{Index: 1, Aid: "2", Cid: "123", Title: "P1", Dur: 120, OwnerMid: "456"},
		},
	}

	deps := Deps{
		FetcherFactory: func(id string, useIntl bool) (fetcher.Fetcher, error) {
			return &mockFetcher{vInfo: vInfo}, nil
		},
		HTTPClient: &mockClient{
			getRedirectLocationFunc: func(ctx context.Context, url string) (string, error) {
				return "", errors.New("not found")
			},
		},
		Logger: discardLogger(),
	}

	aidOri, info, apiType, err := GetVideoInfo(context.Background(), opt, input, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if aidOri != "2" {
		t.Errorf("expected aidOri '2', got %q", aidOri)
	}
	if info.Title != "Test Video" {
		t.Errorf("expected title 'Test Video', got %q", info.Title)
	}
	if apiType != "WEB" {
		t.Errorf("expected apiType 'WEB', got %q", apiType)
	}
}

func TestGetVideoInfo_EpNotFoundFallbackToCheese(t *testing.T) {
	opt := cli.NewOption()
	input := "ep123"

	callCount := 0
	deps := Deps{
		FetcherFactory: func(id string, useIntl bool) (fetcher.Fetcher, error) {
			callCount++
			if id == "ep:123" {
				return &mockFetcher{err: fetcher.ErrKeyNotFound}, nil
			}
			if id == "cheese:123" {
				return &mockFetcher{vInfo: &entity.VInfo{
					Title:    "Cheese Course",
					IsCheese: true,
					PagesInfo: []entity.Page{
						{Index: 1, Aid: "999", Cid: "888", Title: "Lesson 1"},
					},
				}}, nil
			}
			t.Fatalf("unexpected factory call with id %q", id)
			return nil, nil
		},
		HTTPClient: &mockClient{},
		Logger:     discardLogger(),
	}

	aidOri, info, apiType, err := GetVideoInfo(context.Background(), opt, input, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if aidOri != "cheese:123" {
		t.Errorf("expected aidOri 'cheese:123', got %q", aidOri)
	}
	if info.Title != "Cheese Course" {
		t.Errorf("expected title 'Cheese Course', got %q", info.Title)
	}
	if callCount != 2 {
		t.Errorf("expected factory called 2 times, got %d", callCount)
	}
	if apiType != "WEB" {
		t.Errorf("expected apiType 'WEB', got %q", apiType)
	}
}

func TestGetVideoInfo_InteractiveVideoDowngradesTV(t *testing.T) {
	opt := cli.NewOption()
	opt.UseTvApi = true
	input := "BV1xx411c7mD"

	vInfo := &entity.VInfo{
		Title:       "Interactive Video",
		IsSteinGate: true,
		PagesInfo: []entity.Page{
			{Index: 1, Aid: "2", Cid: "123", Title: "P1"},
		},
	}

	deps := Deps{
		FetcherFactory: func(id string, useIntl bool) (fetcher.Fetcher, error) {
			return &mockFetcher{vInfo: vInfo}, nil
		},
		HTTPClient: &mockClient{
			getRedirectLocationFunc: func(ctx context.Context, url string) (string, error) {
				return "", errors.New("not found")
			},
		},
		Logger: discardLogger(),
	}

	_, _, apiType, err := GetVideoInfo(context.Background(), opt, input, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opt.UseTvApi {
		t.Error("expected UseTvApi to be downgraded to false")
	}
	if apiType != "WEB" {
		t.Errorf("expected apiType 'WEB' after downgrade, got %q", apiType)
	}
}

func TestGetVideoInfo_FactoryRouting(t *testing.T) {
	tests := []struct {
		name     string
		opt      *cli.Option
		wantIntl bool
		wantType string
	}{
		{
			name:     "intl",
			opt:      func() *cli.Option { o := cli.NewOption(); o.UseIntlApi = true; return o }(),
			wantIntl: true,
			wantType: "INTL",
		},
		{
			name:     "tv",
			opt:      func() *cli.Option { o := cli.NewOption(); o.UseTvApi = true; return o }(),
			wantIntl: false,
			wantType: "TV",
		},
		{
			name:     "app",
			opt:      func() *cli.Option { o := cli.NewOption(); o.UseAppApi = true; return o }(),
			wantIntl: false,
			wantType: "APP",
		},
		{
			name:     "web",
			opt:      cli.NewOption(),
			wantIntl: false,
			wantType: "WEB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotID string
			var gotIntl bool

			vInfo := &entity.VInfo{
				Title:     "Routed",
				PagesInfo: []entity.Page{{Index: 1, Aid: "2", Cid: "1", Title: "P1"}},
			}

			deps := Deps{
				FetcherFactory: func(id string, useIntl bool) (fetcher.Fetcher, error) {
					gotID = id
					gotIntl = useIntl
					return &mockFetcher{vInfo: vInfo}, nil
				},
				HTTPClient: &mockClient{
					getRedirectLocationFunc: func(ctx context.Context, url string) (string, error) {
						return "", errors.New("not found")
					},
				},
				Logger: discardLogger(),
			}

			_, _, apiType, err := GetVideoInfo(context.Background(), tt.opt, "BV1xx411c7mD", deps)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotID != "2" {
				t.Errorf("expected factory id '2', got %q", gotID)
			}
			if gotIntl != tt.wantIntl {
				t.Errorf("expected useIntl %v, got %v", tt.wantIntl, gotIntl)
			}
			if apiType != tt.wantType {
				t.Errorf("expected apiType %q, got %q", tt.wantType, apiType)
			}
		})
	}
}

func TestCheckLogin_Success(t *testing.T) {
	body := `{"data":{"isLogin":true,"wbi_img":{"img_url":"https://i0.hdslb.com/bfs/wbi/7cd084941338484aae1ad9425b84077c.png","sub_url":"https://i0.hdslb.com/bfs/wbi/4932caff0ff746eab6f01bf08b70ac45.png"}}}`
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			if !strings.Contains(url, "x/web-interface/nav") {
				t.Errorf("unexpected url: %s", url)
			}
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(body)),
				StatusCode: 200,
			}, nil
		},
	}

	isLogin, wbiKey, err := CheckLogin(context.Background(), client, "testcookie")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isLogin {
		t.Error("expected isLogin true")
	}
	if wbiKey == "" {
		t.Error("expected non-empty wbiKey")
	}
	if len(wbiKey) != 32 {
		t.Errorf("expected wbiKey length 32, got %d", len(wbiKey))
	}
}

func TestCheckLogin_NotLoggedIn(t *testing.T) {
	body := `{"data":{"isLogin":false,"wbi_img":{"img_url":"","sub_url":""}}}`
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(body)),
				StatusCode: 200,
			}, nil
		},
	}

	isLogin, wbiKey, err := CheckLogin(context.Background(), client, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isLogin {
		t.Error("expected isLogin false")
	}
	if wbiKey != "" {
		t.Errorf("expected empty wbiKey, got %q", wbiKey)
	}
}

func TestCheckLogin_RequestError(t *testing.T) {
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			return nil, errors.New("network error")
		},
	}

	_, _, err := CheckLogin(context.Background(), client, "")
	if err == nil {
		t.Fatal("expected error for failed request")
	}
}

func TestCheckLogin_CookieHeader(t *testing.T) {
	var gotCookie string
	body := `{"data":{"isLogin":true,"wbi_img":{"img_url":"","sub_url":""}}}`
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			req, _ := http.NewRequest("GET", url, nil)
			for _, opt := range opts {
				opt(req)
			}
			gotCookie = req.Header.Get("Cookie")
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(body)),
				StatusCode: 200,
			}, nil
		},
	}

	CheckLogin(context.Background(), client, "SESSDATA=abc")
	if gotCookie != "SESSDATA=abc" {
		t.Errorf("expected cookie 'SESSDATA=abc', got %q", gotCookie)
	}
}

func TestGetVideoInfo_LoadsCookieFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	origAppDir := appDir
	appDir = tmpDir
	defer func() { appDir = origAppDir }()

	cookiePath := filepath.Join(tmpDir, "BBDown.data")
	if err := os.WriteFile(cookiePath, []byte("filecookie"), 0o644); err != nil {
		t.Fatalf("write cookie file: %v", err)
	}

	var gotCookie string
	body := `{"data":{"isLogin":false,"wbi_img":{"img_url":"","sub_url":""}}}`

	deps := Deps{
		FetcherFactory: func(id string, useIntl bool) (fetcher.Fetcher, error) {
			return &mockFetcher{vInfo: &entity.VInfo{
				Title:     "Test",
				PagesInfo: []entity.Page{{Index: 1, Aid: "2", Cid: "1", Title: "P1"}},
			}}, nil
		},
		HTTPClient: &mockClient{
			getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
				if strings.Contains(url, "x/web-interface/nav") {
					req, _ := http.NewRequest("GET", url, nil)
					for _, opt := range opts {
						opt(req)
					}
					gotCookie = req.Header.Get("Cookie")
					return &http.Response{
						Body:       io.NopCloser(strings.NewReader(body)),
						StatusCode: 200,
					}, nil
				}
				return &http.Response{
					Body:       io.NopCloser(strings.NewReader("")),
					StatusCode: 200,
				}, nil
			},
			getRedirectLocationFunc: func(ctx context.Context, url string) (string, error) {
				return "", errors.New("not found")
			},
		},
		Logger: discardLogger(),
	}

	opt := cli.NewOption()
	_, _, _, err := GetVideoInfo(context.Background(), opt, "BV1xx411c7mD", deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotCookie != "filecookie" {
		t.Errorf("expected cookie 'filecookie', got %q", gotCookie)
	}
}

func TestGetVideoInfo_ShowAllPages(t *testing.T) {
	opt := cli.NewOption()
	opt.ShowAll = true

	pages := make([]entity.Page, 10)
	for i := 0; i < 10; i++ {
		pages[i] = entity.Page{Index: i + 1, Aid: "2", Cid: strconv.Itoa(i + 1), Title: fmt.Sprintf("P%d", i+1)}
	}

	vInfo := &entity.VInfo{
		Title:     "Many Pages",
		PagesInfo: pages,
	}

	deps := Deps{
		FetcherFactory: func(id string, useIntl bool) (fetcher.Fetcher, error) {
			return &mockFetcher{vInfo: vInfo}, nil
		},
		HTTPClient: &mockClient{
			getRedirectLocationFunc: func(ctx context.Context, url string) (string, error) {
				return "", errors.New("not found")
			},
		},
		Logger: discardLogger(),
	}

	_, _, _, err := GetVideoInfo(context.Background(), opt, "BV1xx411c7mD", deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetVideoInfo_LimitedPages(t *testing.T) {
	opt := cli.NewOption()
	opt.ShowAll = false

	pages := make([]entity.Page, 10)
	for i := 0; i < 10; i++ {
		pages[i] = entity.Page{Index: i + 1, Aid: "2", Cid: strconv.Itoa(i + 1), Title: fmt.Sprintf("P%d", i+1)}
	}

	vInfo := &entity.VInfo{
		Title:     "Many Pages",
		PagesInfo: pages,
	}

	deps := Deps{
		FetcherFactory: func(id string, useIntl bool) (fetcher.Fetcher, error) {
			return &mockFetcher{vInfo: vInfo}, nil
		},
		HTTPClient: &mockClient{
			getRedirectLocationFunc: func(ctx context.Context, url string) (string, error) {
				return "", errors.New("not found")
			},
		},
		Logger: discardLogger(),
	}

	_, _, _, err := GetVideoInfo(context.Background(), opt, "BV1xx411c7mD", deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
