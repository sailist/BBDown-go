package util

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

type subtitleMockClient struct {
	server *httptest.Server
}

func (c *subtitleMockClient) Get(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	var path string
	switch {
	case strings.Contains(url, "x/web-interface/view"):
		idx := strings.Index(url, "?")
		if idx >= 0 {
			path = "/x/web-interface/view?" + url[idx+1:]
		} else {
			path = "/x/web-interface/view"
		}
	case strings.Contains(url, "x/player/wbi/v2"):
		idx := strings.Index(url, "?")
		if idx >= 0 {
			path = "/x/player/wbi/v2?" + url[idx+1:]
		} else {
			path = "/x/player/wbi/v2"
		}
	case strings.Contains(url, "intl/gateway/web/v2/subtitle"):
		idx := strings.Index(url, "?")
		if idx >= 0 {
			path = "/intl/gateway/web/v2/subtitle?" + url[idx+1:]
		} else {
			path = "/intl/gateway/web/v2/subtitle"
		}
	case strings.Contains(url, "intl/gateway/v2/ogv/view/app/season"):
		idx := strings.Index(url, "?")
		if idx >= 0 {
			path = "/intl/gateway/v2/ogv/view/app/season?" + url[idx+1:]
		} else {
			path = "/intl/gateway/v2/ogv/view/app/season"
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

func (c *subtitleMockClient) Post(ctx context.Context, url string, body []byte, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *subtitleMockClient) Head(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *subtitleMockClient) GetRedirectLocation(ctx context.Context, url string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (c *subtitleMockClient) GetContentLength(ctx context.Context, url string) (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

func TestGetSubtitles_Normal_LoggedIn_API2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/x/player/wbi/v2"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"data": {
					"subtitle": {
						"subtitles": [
							{"lan": "zh-CN", "subtitle_url": "//example.com/sub1.json"},
							{"lan": "en-US", "subtitle_url": "https://example.com/sub2.json"}
						]
					}
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &subtitleMockClient{server: server}
	cfg := config.NewConfig()
	cfg.Cookie = "SESSDATA=test"

	subs, err := GetSubtitles(context.Background(), client, cfg, "123", "456", "", 1, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2 subtitles, got %d", len(subs))
	}

	if subs[0].Lan != "zh-CN" {
		t.Errorf("expected lan zh-CN, got %q", subs[0].Lan)
	}
	if subs[0].Url != "https://example.com/sub1.json" {
		t.Errorf("expected url fixed to https://, got %q", subs[0].Url)
	}
	if subs[0].Path != "123/123.456.zh-CN.srt" {
		t.Errorf("unexpected path: %q", subs[0].Path)
	}

	if subs[1].Lan != "en-US" {
		t.Errorf("expected lan en-US, got %q", subs[1].Lan)
	}
	if subs[1].Url != "https://example.com/sub2.json" {
		t.Errorf("unexpected url: %q", subs[1].Url)
	}
}

func TestGetSubtitles_Normal_LoggedIn_FallbackToAPI1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/x/player/wbi/v2"):
			w.WriteHeader(http.StatusNotFound)
		case strings.HasPrefix(r.URL.Path, "/x/web-interface/view"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"data": {
					"subtitle": {
						"list": [
							{"lan": "ja", "subtitle_url": "https://example.com/sub_ja.json"}
						]
					}
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &subtitleMockClient{server: server}
	cfg := config.NewConfig()
	cfg.Cookie = "SESSDATA=test"

	subs, err := GetSubtitles(context.Background(), client, cfg, "123", "456", "", 1, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("expected 1 subtitle, got %d", len(subs))
	}
	if subs[0].Lan != "ja" {
		t.Errorf("expected lan ja, got %q", subs[0].Lan)
	}
}

func TestGetSubtitles_Normal_NoCookie(t *testing.T) {
	client := &subtitleMockClient{}
	cfg := config.NewConfig()
	cfg.Cookie = ""

	subs, err := GetSubtitles(context.Background(), client, cfg, "123", "456", "", 1, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subs == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(subs) != 0 {
		t.Fatalf("expected 0 subtitles, got %d", len(subs))
	}
}

func TestGetSubtitles_Intl_API1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/intl/gateway/web/v2/subtitle"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"data": {
					"subtitles": [
						{"lang_key": "zh-Hans", "url": "https://example.com/intl_sub.json"},
						{"lang_key": "en-US", "url": "https://example.com/intl_en.ass"}
					]
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &subtitleMockClient{server: server}
	cfg := config.NewConfig()

	subs, err := GetSubtitles(context.Background(), client, cfg, "123", "456", "789", 1, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2 subtitles, got %d", len(subs))
	}
	if subs[0].Lan != "zh-Hans" {
		t.Errorf("expected lan zh-Hans, got %q", subs[0].Lan)
	}
	if subs[0].Path != "123/123.456.zh-Hans.srt" {
		t.Errorf("unexpected path: %q", subs[0].Path)
	}
	if subs[1].Path != "123/123.456.en-US.ass" {
		t.Errorf("unexpected path: %q", subs[1].Path)
	}
}

func TestGetSubtitles_Intl_FallbackToAPI2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/intl/gateway/web/v2/subtitle"):
			w.WriteHeader(http.StatusNotFound)
		case strings.HasPrefix(r.URL.Path, "/intl/gateway/v2/ogv/view/app/season"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"result": {
					"modules": [
						{
							"data": {
								"episodes": [
									{
										"subtitles": [
											{"key": "ja", "url": "https:\\/\\/example.com\\/ja.json"}
										]
									}
								]
							}
						}
					]
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &subtitleMockClient{server: server}
	cfg := config.NewConfig()

	subs, err := GetSubtitles(context.Background(), client, cfg, "123", "456", "789", 1, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("expected 1 subtitle, got %d", len(subs))
	}
	if subs[0].Lan != "ja" {
		t.Errorf("expected lan ja, got %q", subs[0].Lan)
	}
	if subs[0].Url != "https://example.com/ja.json" {
		t.Errorf("unexpected url: %q", subs[0].Url)
	}
}

func TestConvertSubFromJSON(t *testing.T) {
	jsonStr := `{
		"body": [
			{"from": 0.5, "to": 3.123, "content": "Hello world"},
			{"from": 4.0, "to": 6.999, "content": "Second line"},
			{"to": 10.0, "content": "No from property"}
		]
	}`

	expected := `1
00:00:00,500 --> 00:00:03,123
Hello world

2
00:00:04,000 --> 00:00:06,999
Second line

3
00:00:00,000 --> 00:00:10,000
No from property

`

	got, err := ConvertSubFromJSON(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expected {
		t.Errorf("SRT mismatch.\nExpected:\n%s\nGot:\n%s", expected, got)
	}
}

func TestConvertSubFromJSON_EmptyContent(t *testing.T) {
	jsonStr := `{"body": [{"from": 1.0, "to": 2.0, "content": ""}]}`
	got, err := ConvertSubFromJSON(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "1\n") {
		t.Error("expected sequence number 1")
	}
	if !strings.Contains(got, "00:00:01,000 --> 00:00:02,000") {
		t.Error("expected correct time line")
	}
}

func TestConvertSubFromJSON_LargeTime(t *testing.T) {
	jsonStr := `{"body": [{"from": 3661.999, "to": 3662.5, "content": "Large time"}]}`
	got, err := ConvertSubFromJSON(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "01:01:01,999 --> 01:01:02,500") {
		t.Errorf("unexpected time formatting: %s", got)
	}
}

func TestGetSubtitleCode_CommonMappings(t *testing.T) {
	tests := []struct {
		key         string
		wantCode    string
		wantName    string
	}{
		{"zh-CN", "chi", "Chinese (Simplified)"},
		{"zh-hans", "chi", "Chinese (Simplified)"},
		{"en-US", "eng", "English (USA)"},
		{"ja", "jpn", "Japanese"},
		{"ko", "kor", "Korean"},
		{"unknown", "und", "Undetermined"},
	}

	for _, tt := range tests {
		code, name := GetSubtitleCode(tt.key)
		if code != tt.wantCode {
			t.Errorf("GetSubtitleCode(%q) code = %q, want %q", tt.key, code, tt.wantCode)
		}
		if name != tt.wantName {
			t.Errorf("GetSubtitleCode(%q) name = %q, want %q", tt.key, name, tt.wantName)
		}
	}
}

func TestGetSubtitleCode_CaseFix(t *testing.T) {
	code, name := GetSubtitleCode("zh-hans")
	if code != "chi" {
		t.Errorf("expected code chi, got %q", code)
	}
	if name != "Chinese (Simplified)" {
		t.Errorf("expected name Chinese (Simplified), got %q", name)
	}
}

func TestGetSubtitleCode_MoreLanguages(t *testing.T) {
	cases := []struct {
		key  string
		code string
		name string
	}{
		{"fr", "fre", "French"},
		{"de", "ger", "German"},
		{"es", "spa", "Spanish"},
		{"ru", "rus", "Russian"},
		{"pt-BR", "por", "Portuguese (Brazil)"},
		{"yue-HK", "chi", "Cantonese (Hong Kong)"},
	}
	for _, c := range cases {
		code, name := GetSubtitleCode(c.key)
		if code != c.code {
			t.Errorf("GetSubtitleCode(%q) code = %q, want %q", c.key, code, c.code)
		}
		if name != c.name {
			t.Errorf("GetSubtitleCode(%q) name = %q, want %q", c.key, name, c.name)
		}
	}
}

func TestGetSubtitles_ReturnsEmptySliceOnNil(t *testing.T) {
	// When all APIs fail, GetSubtitles should return an empty slice, not nil.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &subtitleMockClient{server: server}
	cfg := config.NewConfig()
	cfg.Cookie = "SESSDATA=test"

	subs, err := GetSubtitles(context.Background(), client, cfg, "123", "456", "", 1, false)
	if err != nil {
		t.Fatalf("unexpected error when all APIs fail: %v", err)
	}
	if subs == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(subs) != 0 {
		t.Fatalf("expected 0 subtitles, got %d", len(subs))
	}
}

func TestGetSubtitles_FixesProtocolRelativeURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/x/player/wbi/v2") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"data": {
					"subtitle": {
						"subtitles": [
							{"lan": "zh-CN", "subtitle_url": "//example.com/sub.json"}
						]
					}
				}
			}`))
		}
	}))
	defer server.Close()

	client := &subtitleMockClient{server: server}
	cfg := config.NewConfig()
	cfg.Cookie = "SESSDATA=test"

	subs, err := GetSubtitles(context.Background(), client, cfg, "1", "2", "", 1, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("expected 1 subtitle, got %d", len(subs))
	}
	if subs[0].Url != "https://example.com/sub.json" {
		t.Errorf("expected https://example.com/sub.json, got %q", subs[0].Url)
	}
}

func TestGetSubtitles_Intl_API2_IndexBounds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/intl/gateway/web/v2/subtitle"):
			w.WriteHeader(http.StatusNotFound)
		case strings.HasPrefix(r.URL.Path, "/intl/gateway/v2/ogv/view/app/season"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"result": {
					"modules": [
						{
							"data": {
								"episodes": [
									{"subtitles": []},
									{"subtitles": [{"key": "ko", "url": "https://example.com/ko.json"}]}
								]
							}
						}
					]
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &subtitleMockClient{server: server}
	cfg := config.NewConfig()

	// index 2 should pick the second episode
	subs, err := GetSubtitles(context.Background(), client, cfg, "123", "456", "789", 2, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("expected 1 subtitle, got %d", len(subs))
	}
	if subs[0].Lan != "ko" {
		t.Errorf("expected lan ko, got %q", subs[0].Lan)
	}
}
