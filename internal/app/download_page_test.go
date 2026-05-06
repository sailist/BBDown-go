package app

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nilaonai/bbdown-go/internal/cli"
	"github.com/nilaonai/bbdown-go/internal/config"
	"github.com/nilaonai/bbdown-go/internal/core/entity"
	"github.com/nilaonai/bbdown-go/pkg/httpclient"
)

func TestFetchPoints_Success(t *testing.T) {
	body := `{"data":{"view_points":[{"content":"Intro","from":0,"to":60},{"content":"Main","from":60,"to":300}]}}`
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			if !strings.Contains(url, "x/player/wbi/v2") {
				t.Errorf("unexpected url: %s", url)
			}
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(body)),
				StatusCode: 200,
			}, nil
		},
	}

	points, err := FetchPoints(context.Background(), client, "123", "456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(points))
	}
	if points[0].Title != "Intro" || points[0].Start != 0 || points[0].End != 60 {
		t.Errorf("unexpected first point: %+v", points[0])
	}
	if points[1].Title != "Main" || points[1].Start != 60 || points[1].End != 300 {
		t.Errorf("unexpected second point: %+v", points[1])
	}
}

func TestFetchPoints_EmptyResponse(t *testing.T) {
	body := `{"data":{"view_points":[]}}`
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(body)),
				StatusCode: 200,
			}, nil
		},
	}

	points, err := FetchPoints(context.Background(), client, "123", "456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(points) != 0 {
		t.Errorf("expected 0 points, got %d", len(points))
	}
}

func TestFetchPoints_RequestError(t *testing.T) {
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			return nil, io.EOF
		},
	}

	_, err := FetchPoints(context.Background(), client, "123", "456")
	if err == nil {
		t.Fatal("expected error for failed request")
	}
}

func TestDownloadPage_CoverDownload(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	vInfo := &entity.VInfo{Title: "Test Video", Pic: "https://example.com/cover.jpg"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			if url == "https://example.com/cover.jpg" {
				return &http.Response{
					Body:       io.NopCloser(strings.NewReader("fake-image-data")),
					StatusCode: 200,
				}, nil
			}
			return &http.Response{Body: io.NopCloser(strings.NewReader("{}")), StatusCode: 200}, nil
		},
	}

	deps := DownloadDeps{
		HTTPClient: client,
		Logger:     discardLogger(),
	}

	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	coverPath := filepath.Join(tmpDir, "123", "123.jpg")
	data, err := os.ReadFile(coverPath)
	if err != nil {
		t.Fatalf("cover not found: %v", err)
	}
	if string(data) != "fake-image-data" {
		t.Errorf("unexpected cover data: %s", string(data))
	}
}

func TestDownloadPage_SkipCover(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	opt.SkipCover = true
	vInfo := &entity.VInfo{Title: "Test Video", Pic: "https://example.com/cover.jpg"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	coverRequested := false
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			if url == "https://example.com/cover.jpg" {
				coverRequested = true
			}
			return &http.Response{Body: io.NopCloser(strings.NewReader("{}")), StatusCode: 200}, nil
		},
	}

	deps := DownloadDeps{HTTPClient: client, Logger: discardLogger()}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if coverRequested {
		t.Error("expected cover download to be skipped")
	}

	coverPath := filepath.Join(tmpDir, "123", "123.jpg")
	if _, err := os.Stat(coverPath); !os.IsNotExist(err) {
		t.Error("expected cover file to not exist")
	}
}

func TestDownloadPage_SubtitleDownloadAndFilter(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	opt.SkipAi = true
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	cfg := config.NewConfig()
	cfg.Cookie = "testcookie"
	workCfg := &WorkConfig{SavePathFormat: "test", Config: cfg}

	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			if strings.Contains(url, "x/player/wbi/v2") {
				body := `{"data":{"subtitle":{"subtitles":[{"lan":"zh-CN","subtitle_url":"https://example.com/sub1.json"},{"lan":"ai-zh","subtitle_url":"https://example.com/sub2.json"}]}}}`
				return &http.Response{Body: io.NopCloser(strings.NewReader(body)), StatusCode: 200}, nil
			}
			if url == "https://example.com/sub1.json" {
				body := `{"body":[{"from":0,"to":5,"content":"Hello"}]}`
				return &http.Response{Body: io.NopCloser(strings.NewReader(body)), StatusCode: 200}, nil
			}
			if url == "https://example.com/sub2.json" {
				return &http.Response{Body: io.NopCloser(strings.NewReader(`{"body":[]}`)), StatusCode: 200}, nil
			}
			return &http.Response{Body: io.NopCloser(strings.NewReader("{}")), StatusCode: 200}, nil
		},
	}

	deps := DownloadDeps{HTTPClient: client, Logger: discardLogger()}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	subPath := filepath.Join(tmpDir, "123", "123.456.zh-CN.srt")
	if _, err := os.Stat(subPath); os.IsNotExist(err) {
		t.Errorf("expected zh-CN subtitle to exist at %s", subPath)
	}

	siSubPath := filepath.Join(tmpDir, "123", "123.456.ai-zh.srt")
	if _, err := os.Stat(siSubPath); !os.IsNotExist(err) {
		t.Errorf("expected ai-zh subtitle to NOT exist")
	}
}

func TestDownloadPage_SubOnlyEarlyReturn(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	opt.SubOnly = true
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	cfg := config.NewConfig()
	cfg.Cookie = "testcookie"
	workCfg := &WorkConfig{SavePathFormat: "test_output", Config: cfg}

	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			if strings.Contains(url, "x/player/wbi/v2") {
				body := `{"data":{"subtitle":{"subtitles":[{"lan":"zh-CN","subtitle_url":"https://example.com/sub1.json"}]}}}`
				return &http.Response{Body: io.NopCloser(strings.NewReader(body)), StatusCode: 200}, nil
			}
			if url == "https://example.com/sub1.json" {
				body := `{"body":[{"from":0,"to":5,"content":"Hello"}]}`
				return &http.Response{Body: io.NopCloser(strings.NewReader(body)), StatusCode: 200}, nil
			}
			return &http.Response{Body: io.NopCloser(strings.NewReader("{}")), StatusCode: 200}, nil
		},
	}

	deps := DownloadDeps{HTTPClient: client, Logger: discardLogger()}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Subtitle should be moved to final path
	finalSubPath := filepath.Join(tmpDir, "test_output.zh-CN.srt")
	if _, err := os.Stat(finalSubPath); os.IsNotExist(err) {
		t.Errorf("expected final subtitle to exist at %s", finalSubPath)
	}

	// Temp subtitle should no longer exist
	tempSubPath := filepath.Join(tmpDir, "123", "123.456.zh-CN.srt")
	if _, err := os.Stat(tempSubPath); !os.IsNotExist(err) {
		t.Errorf("expected temp subtitle to be moved")
	}
}

func TestDownloadPage_CoverOnlyEarlyReturn(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	opt.CoverOnly = true
	vInfo := &entity.VInfo{Title: "Test Video", Pic: "https://example.com/cover.jpg"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test_output", Config: config.NewConfig()}

	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			if url == "https://example.com/cover.jpg" {
				return &http.Response{
					Body:       io.NopCloser(strings.NewReader("fake-image-data")),
					StatusCode: 200,
				}, nil
			}
			return &http.Response{Body: io.NopCloser(strings.NewReader("{}")), StatusCode: 200}, nil
		},
	}

	deps := DownloadDeps{HTTPClient: client, Logger: discardLogger()}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	finalCoverPath := filepath.Join(tmpDir, "test_output.jpg")
	data, err := os.ReadFile(finalCoverPath)
	if err != nil {
		t.Fatalf("final cover not found: %v", err)
	}
	if string(data) != "fake-image-data" {
		t.Errorf("unexpected cover data: %s", string(data))
	}
}

func TestDownloadPage_OnlyShowInfo(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	opt.OnlyShowInfo = true
	vInfo := &entity.VInfo{Title: "Test Video", Pic: "https://example.com/cover.jpg"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	coverRequested := false
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			if url == "https://example.com/cover.jpg" {
				coverRequested = true
			}
			return &http.Response{Body: io.NopCloser(strings.NewReader("{}")), StatusCode: 200}, nil
		},
	}

	deps := DownloadDeps{HTTPClient: client, Logger: discardLogger()}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if coverRequested {
		t.Error("expected no cover download when OnlyShowInfo is true")
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "123")); !os.IsNotExist(err) {
		t.Error("expected no directory creation when OnlyShowInfo is true")
	}
}

func TestFilterAiSubtitles(t *testing.T) {
	subs := []entity.Subtitle{
		{Lan: "zh-CN", Url: "http://example.com/1"},
		{Lan: "ai-zh", Url: "http://example.com/2"},
		{Lan: "en-US", Url: "http://example.com/3"},
		{Lan: "ai-en", Url: "http://example.com/4"},
	}

	filtered := filterAiSubtitles(subs)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 subtitles after filtering, got %d", len(filtered))
	}
	for _, s := range filtered {
		if strings.HasPrefix(s.Lan, "ai-") {
			t.Errorf("expected ai subtitle to be filtered out, got %s", s.Lan)
		}
	}
}
