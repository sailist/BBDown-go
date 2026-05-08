package parser

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/sailist/BBDown-go/internal/config"
	"github.com/sailist/BBDown-go/pkg/httpclient"
)

type mockClient struct {
	getFunc func(url string) (*http.Response, error)
	calls   []string
}

func (m *mockClient) Get(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	m.calls = append(m.calls, url)
	if m.getFunc != nil {
		return m.getFunc(url)
	}
	return nil, fmt.Errorf("no getFunc set")
}

func (m *mockClient) Post(ctx context.Context, url string, body []byte, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, fmt.Errorf("Post not implemented")
}

func (m *mockClient) Head(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, fmt.Errorf("Head not implemented")
}

func (m *mockClient) GetRedirectLocation(ctx context.Context, url string) (string, error) {
	return "", fmt.Errorf("GetRedirectLocation not implemented")
}

func (m *mockClient) GetContentLength(ctx context.Context, url string) (int64, error) {
	return 0, fmt.Errorf("GetContentLength not implemented")
}

func newMockClientWithResponse(body string) *mockClient {
	return &mockClient{
		getFunc: func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		},
	}
}

func TestExtractTracks_Dash(t *testing.T) {
	dashJSON := `{"data":{"dash":{"duration":300,"video":[{"id":80,"base_url":"https://example.com/v.mp4","backup_url":[],"bandwidth":2000000,"codecid":7,"width":1920,"height":1080,"frame_rate":"30","size":75.5}],"audio":[{"id":30280,"base_url":"https://example.com/a.m4a","backup_url":[],"bandwidth":128000,"codecs":"mp4a.40.2"}]}}}`

	client := newMockClientWithResponse(dashJSON)
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	result, err := ExtractTracks(context.Background(), client, cfg, logger, "av123", "123", "456", "", ExtractOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.VideoTracks) != 1 {
		t.Fatalf("expected 1 video track, got %d", len(result.VideoTracks))
	}
	if result.VideoTracks[0].ID != "80" {
		t.Errorf("video.ID = %s, want 80", result.VideoTracks[0].ID)
	}
	if len(result.AudioTracks) != 1 {
		t.Errorf("expected 1 audio track, got %d", len(result.AudioTracks))
	}
	// non-appApi DASH triggers a re-fetch with max qn
	if len(client.calls) != 2 {
		t.Errorf("expected 2 HTTP calls, got %d", len(client.calls))
	}
}

func TestExtractTracks_DashAppApi(t *testing.T) {
	dashJSON := `{"data":{"dash":{"duration":300,"video":[{"id":80,"base_url":"https://example.com/v.mp4","backup_url":[],"bandwidth":2000000,"codecid":7,"width":1920,"height":1080,"frame_rate":"30","size":75.5}],"audio":[]}}}`

	client := newMockClientWithResponse(dashJSON)
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	result, err := ExtractTracks(context.Background(), client, cfg, logger, "av123", "123", "456", "", ExtractOptions{AppApi: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.VideoTracks) != 1 {
		t.Fatalf("expected 1 video track, got %d", len(result.VideoTracks))
	}
	// appApi DASH does NOT re-fetch with max qn
	if len(client.calls) != 1 {
		t.Errorf("expected 1 HTTP call for appApi DASH, got %d", len(client.calls))
	}
}

func TestExtractTracks_Flv(t *testing.T) {
	flvJSON := `{"data":{"quality":80,"video_codecid":7,"durl":[{"url":"https://example.com/seg.flv","size":50.5,"length":300000}],"accept_quality":[80,64]}}`

	client := newMockClientWithResponse(flvJSON)
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	result, err := ExtractTracks(context.Background(), client, cfg, logger, "av123", "123", "456", "", ExtractOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.VideoTracks) != 1 {
		t.Fatalf("expected 1 video track, got %d", len(result.VideoTracks))
	}
	if len(result.Clips) != 1 {
		t.Errorf("expected 1 clip, got %d", len(result.Clips))
	}
	// FLV: first call detects FLV, second call fetches max qn
	if len(client.calls) != 2 {
		t.Errorf("expected 2 HTTP calls, got %d", len(client.calls))
	}
}

func TestExtractTracks_FlvUsesMaxQn(t *testing.T) {
	flvJSON := `{"data":{"quality":80,"video_codecid":7,"durl":[{"url":"https://example.com/seg.flv","size":50.5,"length":300000}],"accept_quality":[80,64]}}`

	client := newMockClientWithResponse(flvJSON)
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	_, err := ExtractTracks(context.Background(), client, cfg, logger, "av123", "123", "456", "", ExtractOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Second call should have qn=127 (max)
	if len(client.calls) < 2 {
		t.Fatalf("expected at least 2 calls, got %d", len(client.calls))
	}
	if !strings.Contains(client.calls[1], "qn=127") {
		t.Errorf("expected second call to use max qn (127), got URL: %s", client.calls[1])
	}
}

func TestExtractTracks_Intl(t *testing.T) {
	intlCode0 := `{"data":{"video_info":{"timelength":300000,"stream_list":[{"stream_info":{"quality":64},"dash_video":{"base_url":"https://example.com/v64.mp4","backup_url":[],"bandwidth":1000000,"codecid":7,"size":40.0}}],"dash_audio":[{"id":30280,"base_url":"https://example.com/a.m4a","backup_url":[],"bandwidth":128000}]}}}`
	intlCode1 := `{"data":{"video_info":{"timelength":300000,"stream_list":[{"stream_info":{"quality":80},"dash_video":{"base_url":"https://example.com/v80.mp4","backup_url":[],"bandwidth":2000000,"codecid":7,"size":75.5}}],"dash_audio":[{"id":30280,"base_url":"https://example.com/a.m4a","backup_url":[],"bandwidth":128000}]}}}`

	client := &mockClient{
		getFunc: func(url string) (*http.Response, error) {
			if strings.Contains(url, "prefer_code_type=0") {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(intlCode0))}, nil
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(intlCode1))}, nil
		},
	}
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	result, err := ExtractTracks(context.Background(), client, cfg, logger, "ep:123", "123", "456", "789", ExtractOptions{IntlApi: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.VideoTracks) != 1 {
		t.Fatalf("expected 1 video track, got %d", len(result.VideoTracks))
	}
	// Should be quality 80 from code=1 overwrite
	if result.VideoTracks[0].ID != "80" {
		t.Errorf("video.ID = %s, want 80 (code=1 overwrite)", result.VideoTracks[0].ID)
	}
	if len(client.calls) != 2 {
		t.Errorf("expected 2 HTTP calls, got %d", len(client.calls))
	}
}

func TestExtractTracks_IntlNoStreamList(t *testing.T) {
	// intl response without stream_list falls through to normal parsing
	normalJSON := `{"data":{"dash":{"duration":300,"video":[],"audio":[]}}}`

	client := newMockClientWithResponse(normalJSON)
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	result, err := ExtractTracks(context.Background(), client, cfg, logger, "ep:123", "123", "456", "789", ExtractOptions{IntlApi: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should parse as DASH (non-appApi => 2 calls)
	if len(client.calls) != 2 {
		t.Errorf("expected 2 HTTP calls, got %d", len(client.calls))
	}
	if result.WebJsonString != normalJSON {
		t.Errorf("unexpected webJsonString")
	}
}

func TestExtractTracks_VipFallback(t *testing.T) {
	vipJSON := `{"message":"大会员专享限制"}`
	playInfoJSON := `{"data":{"dash":{"duration":300,"video":[{"id":80,"base_url":"https://example.com/v.mp4","backup_url":[],"bandwidth":2000000,"codecid":7,"width":1920,"height":1080,"frame_rate":"30","size":75.5}],"audio":[]}}}`
	html := `<html><script>window.__playinfo__=` + playInfoJSON + `</script></html>`

	client := &mockClient{
		getFunc: func(url string) (*http.Response, error) {
			if strings.Contains(url, "/bangumi/play/ep") {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(html))}, nil
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(vipJSON))}, nil
		},
	}
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	result, err := ExtractTracks(context.Background(), client, cfg, logger, "ep:123", "123", "456", "789", ExtractOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.VideoTracks) != 1 {
		t.Fatalf("expected 1 video track after fallback, got %d", len(result.VideoTracks))
	}
	if result.VideoTracks[0].ID != "80" {
		t.Errorf("video.ID = %s, want 80", result.VideoTracks[0].ID)
	}
}

func TestExtractTracks_BangumiClipInfo(t *testing.T) {
	dashJSON := `{"data":{"dash":{"duration":300,"video":[],"audio":[]},"clip_info_list":[{"toastText":"即将跳过片头","start":0,"end":90},{"toastText":"即将跳过片尾","start":180,"end":270}]}}`

	client := newMockClientWithResponse(dashJSON)
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	result, err := ExtractTracks(context.Background(), client, cfg, logger, "ep:123", "123", "456", "789", ExtractOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ExtraPoints) == 0 {
		t.Fatalf("expected extra points, got none")
	}
	hasMain := false
	for _, p := range result.ExtraPoints {
		if p.Title == "正片" {
			hasMain = true
			break
		}
	}
	if !hasMain {
		t.Errorf("expected '正片' segment in extra points, got %+v", result.ExtraPoints)
	}
}

func TestExtractTracks_BangumiClipInfo_ResultPath(t *testing.T) {
	dashJSON := `{"result":{"video_info":{"dash":{"duration":300,"video":[],"audio":[]},"clip_info_list":[{"toastText":"即将跳过片头","start":0,"end":90}]}}}`

	client := newMockClientWithResponse(dashJSON)
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	result, err := ExtractTracks(context.Background(), client, cfg, logger, "ep:123", "123", "456", "789", ExtractOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ExtraPoints) == 0 {
		t.Fatalf("expected extra points, got none")
	}
	if result.ExtraPoints[0].Title != "片头" {
		t.Errorf("expected first point title '片头', got %s", result.ExtraPoints[0].Title)
	}
}

func TestExtractTracks_TvApi(t *testing.T) {
	dashJSON := `{"data":{"dash":{"duration":300,"video":[{"id":80,"base_url":"https://example.com/v.mp4","backup_url":[],"bandwidth":2000000,"codecid":7,"width":1920,"height":1080,"frame_rate":"30","size":75.5}],"audio":[]}}}`

	client := newMockClientWithResponse(dashJSON)
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	result, err := ExtractTracks(context.Background(), client, cfg, logger, "av123", "123", "456", "", ExtractOptions{TvApi: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.VideoTracks) != 1 {
		t.Fatalf("expected 1 video track, got %d", len(result.VideoTracks))
	}
	// TV API DASH also re-fetches with max qn
	if len(client.calls) != 2 {
		t.Errorf("expected 2 HTTP calls, got %d", len(client.calls))
	}
}

func TestExtractTracks_HttpError(t *testing.T) {
	client := &mockClient{
		getFunc: func(url string) (*http.Response, error) {
			return nil, fmt.Errorf("network error")
		},
	}
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	_, err := ExtractTracks(context.Background(), client, cfg, logger, "av123", "123", "456", "", ExtractOptions{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "network error") {
		t.Errorf("expected error to contain 'network error', got %v", err)
	}
}

func TestExtractTracks_AppApiFallback(t *testing.T) {
	dashJSON := `{"data":{"dash":{"duration":300,"video":[{"id":80,"base_url":"https://example.com/v.mp4","backup_url":[],"bandwidth":2000000,"codecid":7,"width":1920,"height":1080,"frame_rate":"30","size":75.5}],"audio":[]}}}`
	client := newMockClientWithResponse(dashJSON)
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	result, err := ExtractTracks(context.Background(), client, cfg, logger, "av123", "123", "456", "", ExtractOptions{AppApi: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.VideoTracks) != 1 {
		t.Fatalf("expected 1 video track, got %d", len(result.VideoTracks))
	}
	// AppApi falls back to web API until gRPC is implemented
	if len(client.calls) != 1 {
		t.Errorf("expected 1 HTTP call, got %d", len(client.calls))
	}
}

func TestExtractTracks_CheesePath(t *testing.T) {
	dashJSON := `{"data":{"dash":{"duration":300,"video":[],"audio":[]}}}`

	client := newMockClientWithResponse(dashJSON)
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	_, err := ExtractTracks(context.Background(), client, cfg, logger, "cheese:123", "123", "456", "789", ExtractOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Cheese should use bangumi path and replace /pgc/ with /pugv/
	foundPugv := false
	for _, url := range client.calls {
		if strings.Contains(url, "/pugv/") {
			foundPugv = true
			break
		}
	}
	if !foundPugv {
		t.Errorf("expected cheese URL to contain /pugv/, calls: %v", client.calls)
	}
}

func TestGetMaxQn(t *testing.T) {
	qn := getMaxQn()
	if qn != "127" {
		t.Errorf("getMaxQn() = %s, want 127", qn)
	}
}

func TestExtractTracks_NeitherDashNorFlv(t *testing.T) {
	emptyJSON := `{"data":{}}`

	client := newMockClientWithResponse(emptyJSON)
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	result, err := ExtractTracks(context.Background(), client, cfg, logger, "av123", "123", "456", "", ExtractOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.VideoTracks) != 0 {
		t.Errorf("expected 0 video tracks, got %d", len(result.VideoTracks))
	}
	if len(result.AudioTracks) != 0 {
		t.Errorf("expected 0 audio tracks, got %d", len(result.AudioTracks))
	}
}

func TestExtractTracks_DashResultVideoInfoPath(t *testing.T) {
	dashJSON := `{"result":{"video_info":{"dash":{"duration":300,"video":[{"id":80,"base_url":"https://example.com/v.mp4","backup_url":[],"bandwidth":2000000,"codecid":7,"width":1920,"height":1080,"frame_rate":"30","size":75.5}],"audio":[]}}}}`

	client := newMockClientWithResponse(dashJSON)
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	result, err := ExtractTracks(context.Background(), client, cfg, logger, "av123", "123", "456", "", ExtractOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.VideoTracks) != 1 {
		t.Fatalf("expected 1 video track, got %d", len(result.VideoTracks))
	}
}

func TestExtractTracks_FlvResultPath(t *testing.T) {
	flvJSON := `{"result":{"quality":80,"video_codecid":7,"durl":[{"url":"https://example.com/seg.flv","size":50.5,"length":300000}],"accept_quality":[80,64]}}`

	client := newMockClientWithResponse(flvJSON)
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	result, err := ExtractTracks(context.Background(), client, cfg, logger, "av123", "123", "456", "", ExtractOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.VideoTracks) != 1 {
		t.Fatalf("expected 1 video track, got %d", len(result.VideoTracks))
	}
}

func TestExtractTracks_VipFallback_NoRegexMatch(t *testing.T) {
	vipJSON := `{"message":"大会员专享限制"}`
	html := `<html><head></head><body>no playinfo</body></html>`

	client := &mockClient{
		getFunc: func(url string) (*http.Response, error) {
			if strings.Contains(url, "/bangumi/play/ep") {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(html))}, nil
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(vipJSON))}, nil
		},
	}
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	result, err := ExtractTracks(context.Background(), client, cfg, logger, "ep:123", "123", "456", "789", ExtractOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Falls through: no dash, no durl => empty result with vipJSON as webJsonString
	if result.WebJsonString != vipJSON {
		t.Errorf("expected original vip JSON when regex doesn't match")
	}
}

func TestExtractTracks_IntlAudioDedup(t *testing.T) {
	intlJSON := `{"data":{"video_info":{"timelength":300000,"stream_list":[],"dash_audio":[{"id":30280,"base_url":"https://example.com/a.m4a","backup_url":[],"bandwidth":128000},{"id":30280,"base_url":"https://example.com/a2.m4a","backup_url":[],"bandwidth":128000}]}}}`

	client := newMockClientWithResponse(intlJSON)
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	result, err := ExtractTracks(context.Background(), client, cfg, logger, "ep:123", "123", "456", "789", ExtractOptions{IntlApi: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.AudioTracks) != 1 {
		t.Errorf("expected 1 audio track after dedup, got %d", len(result.AudioTracks))
	}
}

func TestExtractTracks_BangumiNoClipInfo(t *testing.T) {
	dashJSON := `{"data":{"dash":{"duration":300,"video":[],"audio":[]}}}`

	client := newMockClientWithResponse(dashJSON)
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	result, err := ExtractTracks(context.Background(), client, cfg, logger, "ep:123", "123", "456", "789", ExtractOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ExtraPoints) != 0 {
		t.Errorf("expected 0 extra points when no clip_info_list, got %d", len(result.ExtraPoints))
	}
}

func TestExtractTracks_ContextCancellation(t *testing.T) {
	client := newMockClientWithResponse(`{}`)
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ExtractTracks(ctx, client, cfg, logger, "av123", "123", "456", "", ExtractOptions{})
	// Context may or may not be checked by the mock; if not checked, the call succeeds.
	// This test mainly verifies the function accepts a context and doesn't panic.
	_ = err
}

func TestExtractTracks_VideoTracksEqual(t *testing.T) {
	// Verify that video track deduplication works in intl parser
	intlJSON := `{"data":{"video_info":{"timelength":300000,"stream_list":[{"stream_info":{"quality":80},"dash_video":{"base_url":"https://example.com/v.mp4","backup_url":[],"bandwidth":2000000,"codecid":7,"size":75.5}},{"stream_info":{"quality":80},"dash_video":{"base_url":"https://example.com/v2.mp4","backup_url":[],"bandwidth":2000000,"codecid":7,"size":75.5}}],"dash_audio":[]}}}`

	client := newMockClientWithResponse(intlJSON)
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	result, err := ExtractTracks(context.Background(), client, cfg, logger, "ep:123", "123", "456", "789", ExtractOptions{IntlApi: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.VideoTracks) != 1 {
		t.Errorf("expected 1 video track after dedup, got %d", len(result.VideoTracks))
	}
}

func TestExtractTracks_FlvQnExtras(t *testing.T) {
	flvJSON := `{"data":{"quality":80,"video_codecid":7,"durl":[{"url":"https://example.com/seg.flv","size":50.5,"length":300000}],"qn_extras":[{"qn":127},{"qn":120},{"qn":80}]}}`

	client := newMockClientWithResponse(flvJSON)
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	result, err := ExtractTracks(context.Background(), client, cfg, logger, "av123", "123", "456", "", ExtractOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Dfns) != 3 {
		t.Errorf("expected 3 dfns, got %d", len(result.Dfns))
	}
	if result.Dfns[0] != "127" {
		t.Errorf("dfn[0] = %s, want 127", result.Dfns[0])
	}
}

func TestExtractTracks_EmptyResponse(t *testing.T) {
	client := newMockClientWithResponse(`{}`)
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	result, err := ExtractTracks(context.Background(), client, cfg, logger, "av123", "123", "456", "", ExtractOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.WebJsonString != `{}` {
		t.Errorf("unexpected webJsonString")
	}
}
