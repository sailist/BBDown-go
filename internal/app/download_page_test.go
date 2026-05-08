package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sailist/BBDown-go/internal/cli"
	"github.com/sailist/BBDown-go/internal/config"
	"github.com/sailist/BBDown-go/internal/core/entity"
	"github.com/sailist/BBDown-go/internal/core/parser"
	"github.com/sailist/BBDown-go/internal/download"
	"github.com/sailist/BBDown-go/internal/muxer"
	"github.com/sailist/BBDown-go/pkg/httpclient"
)

type mockDownloader struct {
	downloadFunc   func(ctx context.Context, url, path string, opts download.Options) error
	downloadMTFunc func(ctx context.Context, url, path string, opts download.Options) error
}

func (m *mockDownloader) Download(ctx context.Context, url, path string, opts download.Options) error {
	if m.downloadFunc != nil {
		return m.downloadFunc(ctx, url, path, opts)
	}
	return nil
}

func (m *mockDownloader) DownloadMultiThread(ctx context.Context, url, path string, opts download.Options) error {
	if m.downloadMTFunc != nil {
		return m.downloadMTFunc(ctx, url, path, opts)
	}
	return nil
}

type mockMuxer struct {
	muxFunc      func(ctx context.Context, cfg muxer.MuxConfig) error
	mergeFLVFunc func(ctx context.Context, files []string, outPath string) error
}

func (m *mockMuxer) Mux(ctx context.Context, cfg muxer.MuxConfig) error {
	if m.muxFunc != nil {
		return m.muxFunc(ctx, cfg)
	}
	return nil
}

func (m *mockMuxer) MergeFLV(ctx context.Context, files []string, outPath string) error {
	if m.mergeFLVFunc != nil {
		return m.mergeFLVFunc(ctx, files, outPath)
	}
	return nil
}

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

// Part 2 tests

func TestDownloadPage_ExtractTracksCalled(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	extractCalled := false
	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		extractCalled = true
		if aid != "123" || cid != "456" {
			t.Errorf("unexpected aid/cid: %s/%s", aid, cid)
		}
		return &entity.ParsedResult{}, nil
	}

	deps := DownloadDeps{HTTPClient: &mockClient{}, Logger: discardLogger(), ExtractTracks: extractTracks}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !extractCalled {
		t.Error("expected ExtractTracks to be called")
	}
}

func TestDownloadPage_MergeExtraPoints(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		return &entity.ParsedResult{
			ExtraPoints: []entity.ViewPoint{{Title: "Extra", Start: 10, End: 20}},
		}, nil
	}

	deps := DownloadDeps{HTTPClient: &mockClient{}, Logger: discardLogger(), ExtractTracks: extractTracks}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Points) != 1 || p.Points[0].Title != "Extra" {
		t.Errorf("expected points to be merged, got %+v", p.Points)
	}
}

func TestDownloadPage_PointsNotOverwritten(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1", Points: []entity.ViewPoint{{Title: "Existing", Start: 0, End: 5}}}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		return &entity.ParsedResult{
			ExtraPoints: []entity.ViewPoint{{Title: "Extra", Start: 10, End: 20}},
		}, nil
	}

	deps := DownloadDeps{HTTPClient: &mockClient{}, Logger: discardLogger(), ExtractTracks: extractTracks}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Points) != 1 || p.Points[0].Title != "Existing" {
		t.Errorf("expected existing points to be preserved, got %+v", p.Points)
	}
}

func TestDownloadPage_DebugJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	opt.Debug = true
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		return &entity.ParsedResult{WebJsonString: `{"debug":true}`}, nil
	}

	deps := DownloadDeps{HTTPClient: &mockClient{}, Logger: discardLogger(), ExtractTracks: extractTracks}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	debugPath := filepath.Join(tmpDir, "123", "123.debug.json")
	data, err := os.ReadFile(debugPath)
	if err != nil {
		t.Fatalf("debug json not found: %v", err)
	}
	if string(data) != `{"debug":true}` {
		t.Errorf("unexpected debug json: %s", string(data))
	}
}

func TestDownloadPage_DASHAudioOnly(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	opt.AudioOnly = true
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	var capturedResult *entity.ParsedResult
	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		return &entity.ParsedResult{
			VideoTracks: []entity.Video{{ID: "1", Dfn: "1080P"}},
			AudioTracks: []entity.Audio{{ID: "1", Dfn: "320K"}},
		}, nil
	}

	printCalled := false
	origPrint := printStreamsImpl
	printStreamsImpl = func(logger *slog.Logger, videos []entity.Video, audios []entity.Audio, bgAudios []entity.Audio, roleAudioList []entity.AudioMaterialInfo) {
		printCalled = true
		capturedResult = &entity.ParsedResult{VideoTracks: videos, AudioTracks: audios}
	}
	defer func() { printStreamsImpl = origPrint }()

	deps := DownloadDeps{HTTPClient: &mockClient{}, Logger: discardLogger(), ExtractTracks: extractTracks, Downloader: &mockDownloader{}, Muxer: &mockMuxer{}}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !printCalled {
		t.Fatal("expected printStreams to be called")
	}
	if len(capturedResult.VideoTracks) != 0 {
		t.Errorf("expected video tracks to be cleared, got %d", len(capturedResult.VideoTracks))
	}
	if len(capturedResult.AudioTracks) != 1 {
		t.Errorf("expected 1 audio track, got %d", len(capturedResult.AudioTracks))
	}
}

func TestDownloadPage_DASHVideoOnly(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	opt.VideoOnly = true
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	var capturedResult *entity.ParsedResult
	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		return &entity.ParsedResult{
			VideoTracks:           []entity.Video{{ID: "1", Dfn: "1080P"}},
			AudioTracks:           []entity.Audio{{ID: "1", Dfn: "320K"}},
			BackgroundAudioTracks: []entity.Audio{{ID: "2", Dfn: "64K"}},
			RoleAudioList:         []entity.AudioMaterialInfo{{Title: "Role1"}},
		}, nil
	}

	printCalled := false
	origPrint := printStreamsImpl
	printStreamsImpl = func(logger *slog.Logger, videos []entity.Video, audios []entity.Audio, bgAudios []entity.Audio, roleAudioList []entity.AudioMaterialInfo) {
		printCalled = true
		capturedResult = &entity.ParsedResult{
			VideoTracks:           videos,
			AudioTracks:           audios,
			BackgroundAudioTracks: bgAudios,
			RoleAudioList:         roleAudioList,
		}
	}
	defer func() { printStreamsImpl = origPrint }()

	deps := DownloadDeps{HTTPClient: &mockClient{}, Logger: discardLogger(), ExtractTracks: extractTracks, Downloader: &mockDownloader{}, Muxer: &mockMuxer{}}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !printCalled {
		t.Fatal("expected printStreams to be called")
	}
	if len(capturedResult.VideoTracks) != 1 {
		t.Errorf("expected 1 video track, got %d", len(capturedResult.VideoTracks))
	}
	if len(capturedResult.AudioTracks) != 0 {
		t.Errorf("expected audio tracks to be cleared, got %d", len(capturedResult.AudioTracks))
	}
	if len(capturedResult.BackgroundAudioTracks) != 0 {
		t.Errorf("expected background audio tracks to be cleared, got %d", len(capturedResult.BackgroundAudioTracks))
	}
	if len(capturedResult.RoleAudioList) != 0 {
		t.Errorf("expected role audio list to be cleared, got %d", len(capturedResult.RoleAudioList))
	}
}

func TestDownloadPage_DASHInteractiveSelection(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	opt.Interactive = true
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		return &entity.ParsedResult{
			VideoTracks: []entity.Video{{ID: "1", Dfn: "1080P"}, {ID: "2", Dfn: "720P"}},
			AudioTracks: []entity.Audio{{ID: "1", Dfn: "320K"}, {ID: "2", Dfn: "64K"}},
		}, nil
	}

	selectCalls := 0
	selectInteractive := func(prompt string, max int) (int, error) {
		selectCalls++
		if strings.Contains(prompt, "video") {
			return 1, nil
		}
		return 0, nil
	}

	deps := DownloadDeps{HTTPClient: &mockClient{}, Logger: discardLogger(), ExtractTracks: extractTracks, SelectTrackInteractive: selectInteractive, Downloader: &mockDownloader{}, Muxer: &mockMuxer{}}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selectCalls != 2 {
		t.Errorf("expected 2 select calls, got %d", selectCalls)
	}
}

func TestDownloadPage_DASHOnlyShowInfoAfterExtract(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	opt.OnlyShowInfo = true
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		return &entity.ParsedResult{
			VideoTracks: []entity.Video{{ID: "1", Dfn: "1080P"}},
			AudioTracks: []entity.Audio{{ID: "1", Dfn: "320K"}},
		}, nil
	}

	deps := DownloadDeps{HTTPClient: &mockClient{}, Logger: discardLogger(), ExtractTracks: extractTracks}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return nil after extracting and printing tracks
}

func TestDownloadPage_FLVBasic(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		return &entity.ParsedResult{
			Clips: []string{"http://example.com/clip1.flv"},
			Dfns:  []string{"80"},
			VideoTracks: []entity.Video{{ID: "80", Dfn: "1080P"}},
		}, nil
	}

	deps := DownloadDeps{HTTPClient: &mockClient{}, Logger: discardLogger(), ExtractTracks: extractTracks, Downloader: &mockDownloader{}, Muxer: &mockMuxer{}}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDownloadPage_NoTracksLogsError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		return &entity.ParsedResult{}, nil
	}

	deps := DownloadDeps{HTTPClient: &mockClient{}, Logger: discardLogger(), ExtractTracks: extractTracks}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDownloadPage_DASHSortTracks(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{
		SavePathFormat:   "test",
		Config:           config.NewConfig(),
		DfnPriority:      map[string]int{"1080P": 0, "720P": 1},
		EncodingPriority: map[string]byte{"AVC": 0, "HEVC": 1},
	}

	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		return &entity.ParsedResult{
			VideoTracks: []entity.Video{
				{ID: "1", Dfn: "720P", Codecs: "HEVC", Bandwith: 2000},
				{ID: "2", Dfn: "1080P", Codecs: "AVC", Bandwith: 4000},
			},
			AudioTracks: []entity.Audio{
				{ID: "1", Dfn: "64K", Codecs: "M4A", Bandwith: 64},
				{ID: "2", Dfn: "320K", Codecs: "M4A", Bandwith: 320},
			},
		}, nil
	}

	var capturedVideos []entity.Video
	var capturedAudios []entity.Audio
	origPrint := printStreamsImpl
	printStreamsImpl = func(logger *slog.Logger, videos []entity.Video, audios []entity.Audio, bgAudios []entity.Audio, roleAudioList []entity.AudioMaterialInfo) {
		capturedVideos = videos
		capturedAudios = audios
	}
	defer func() { printStreamsImpl = origPrint }()

	deps := DownloadDeps{HTTPClient: &mockClient{}, Logger: discardLogger(), ExtractTracks: extractTracks, Downloader: &mockDownloader{}, Muxer: &mockMuxer{}}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(capturedVideos) != 2 {
		t.Fatalf("expected 2 video tracks, got %d", len(capturedVideos))
	}
	if capturedVideos[0].Dfn != "1080P" {
		t.Errorf("expected first video to be 1080P, got %s", capturedVideos[0].Dfn)
	}
	if len(capturedAudios) != 2 {
		t.Fatalf("expected 2 audio tracks, got %d", len(capturedAudios))
	}
	if capturedAudios[0].Bandwith != 320 {
		t.Errorf("expected first audio to have highest bandwidth, got %d", capturedAudios[0].Bandwith)
	}
}

func TestDownloadPage_DASHHideStreams(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	opt.HideStreams = true
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		return &entity.ParsedResult{
			VideoTracks: []entity.Video{{ID: "1", Dfn: "1080P"}},
			AudioTracks: []entity.Audio{{ID: "1", Dfn: "320K"}},
		}, nil
	}

	printCalled := false
	origPrint := printStreamsImpl
	printStreamsImpl = func(logger *slog.Logger, videos []entity.Video, audios []entity.Audio, bgAudios []entity.Audio, roleAudioList []entity.AudioMaterialInfo) {
		printCalled = true
	}
	defer func() { printStreamsImpl = origPrint }()

	deps := DownloadDeps{HTTPClient: &mockClient{}, Logger: discardLogger(), ExtractTracks: extractTracks, Downloader: &mockDownloader{}, Muxer: &mockMuxer{}}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if printCalled {
		t.Error("expected printStreams to be skipped when HideStreams is true")
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



// Part 3 tests

func TestDownloadPage_DASHDownloadAndMux(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		return &entity.ParsedResult{
			VideoTracks: []entity.Video{{ID: "1", Dfn: "1080P", BaseUrl: "https://example.com/video.m4v"}},
			AudioTracks: []entity.Audio{{ID: "1", Dfn: "320K", BaseUrl: "https://example.com/audio.m4a"}},
		}, nil
	}

	var downloaded []string
	downloader := &mockDownloader{
		downloadMTFunc: func(ctx context.Context, url, path string, opts download.Options) error {
			downloaded = append(downloaded, url)
			return os.WriteFile(path, []byte("data"), 0o644)
		},
	}

	var muxCfg *muxer.MuxConfig
	muxerMock := &mockMuxer{
		muxFunc: func(ctx context.Context, cfg muxer.MuxConfig) error {
			muxCfg = &cfg
			return nil
		},
	}

	deps := DownloadDeps{HTTPClient: &mockClient{}, Logger: discardLogger(), ExtractTracks: extractTracks, Downloader: downloader, Muxer: muxerMock}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(downloaded) != 2 {
		t.Errorf("expected 2 downloads, got %d", len(downloaded))
	}
	if muxCfg == nil {
		t.Fatal("expected mux to be called")
	}
	if muxCfg.VideoPath == "" {
		t.Error("expected video path in mux config")
	}
	if muxCfg.AudioPath == "" {
		t.Error("expected audio path in mux config")
	}
}

func TestDownloadPage_DASHSkipMux(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	opt.SkipMux = true
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		return &entity.ParsedResult{
			VideoTracks: []entity.Video{{ID: "1", Dfn: "1080P", BaseUrl: "https://example.com/video.m4v"}},
			AudioTracks: []entity.Audio{{ID: "1", Dfn: "320K", BaseUrl: "https://example.com/audio.m4a"}},
		}, nil
	}

	downloader := &mockDownloader{
		downloadMTFunc: func(ctx context.Context, url, path string, opts download.Options) error {
			return os.WriteFile(path, []byte("data"), 0o644)
		},
	}

	muxCalled := false
	muxerMock := &mockMuxer{
		muxFunc: func(ctx context.Context, cfg muxer.MuxConfig) error {
			muxCalled = true
			return nil
		},
	}

	deps := DownloadDeps{HTTPClient: &mockClient{}, Logger: discardLogger(), ExtractTracks: extractTracks, Downloader: downloader, Muxer: muxerMock}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if muxCalled {
		t.Error("expected mux to be skipped")
	}
}

func TestDownloadPage_FLVDownloadAndMerge(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		return &entity.ParsedResult{
			Clips:       []string{"https://example.com/clip1.flv", "https://example.com/clip2.flv"},
			Dfns:        []string{"80"},
			VideoTracks: []entity.Video{{ID: "80", Dfn: "1080P"}},
		}, nil
	}

	var downloaded []string
	downloader := &mockDownloader{
		downloadMTFunc: func(ctx context.Context, url, path string, opts download.Options) error {
			downloaded = append(downloaded, url)
			return os.WriteFile(path, []byte("data"), 0o644)
		},
	}

	var mergedFiles []string
	var muxCfg *muxer.MuxConfig
	muxerMock := &mockMuxer{
		mergeFLVFunc: func(ctx context.Context, files []string, outPath string) error {
			mergedFiles = files
			return os.WriteFile(outPath, []byte("merged"), 0o644)
		},
		muxFunc: func(ctx context.Context, cfg muxer.MuxConfig) error {
			muxCfg = &cfg
			return nil
		},
	}

	deps := DownloadDeps{HTTPClient: &mockClient{}, Logger: discardLogger(), ExtractTracks: extractTracks, Downloader: downloader, Muxer: muxerMock}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(downloaded) != 2 {
		t.Errorf("expected 2 clip downloads, got %d", len(downloaded))
	}
	if len(mergedFiles) != 2 {
		t.Errorf("expected 2 files merged, got %d", len(mergedFiles))
	}
	if muxCfg == nil {
		t.Fatal("expected mux to be called")
	}
	if muxCfg.VideoPath == "" {
		t.Error("expected merged video path in mux config")
	}
}

func TestDownloadPage_FileExistsSkipsDownload(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	// Create the output file so it already exists
	if err := os.WriteFile("test.mp4", []byte("exists"), 0o644); err != nil {
		t.Fatalf("create existing file: %v", err)
	}

	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		return &entity.ParsedResult{
			VideoTracks: []entity.Video{{ID: "1", Dfn: "1080P", BaseUrl: "https://example.com/video.m4v"}},
		}, nil
	}

	downloadCalled := false
	downloader := &mockDownloader{
		downloadMTFunc: func(ctx context.Context, url, path string, opts download.Options) error {
			downloadCalled = true
			return nil
		},
	}

	deps := DownloadDeps{HTTPClient: &mockClient{}, Logger: discardLogger(), ExtractTracks: extractTracks, Downloader: downloader, Muxer: &mockMuxer{}}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if downloadCalled {
		t.Error("expected download to be skipped when file exists")
	}
}

func TestDownloadPage_RetryLogic(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	callCount := 0
	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		callCount++
		if callCount < 2 {
			return nil, errors.New("network error")
		}
		return &entity.ParsedResult{
			VideoTracks: []entity.Video{{ID: "1", Dfn: "1080P", BaseUrl: "https://example.com/video.m4v"}},
		}, nil
	}

	downloader := &mockDownloader{
		downloadMTFunc: func(ctx context.Context, url, path string, opts download.Options) error {
			return os.WriteFile(path, []byte("data"), 0o644)
		},
	}

	deps := DownloadDeps{HTTPClient: &mockClient{}, Logger: discardLogger(), ExtractTracks: extractTracks, Downloader: downloader, Muxer: &mockMuxer{}}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 extract calls (1 retry), got %d", callCount)
	}
}

func TestDownloadPage_RetryExhausted(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		return nil, errors.New("persistent network error")
	}

	deps := DownloadDeps{HTTPClient: &mockClient{}, Logger: discardLogger(), ExtractTracks: extractTracks, Downloader: &mockDownloader{}, Muxer: &mockMuxer{}}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}
	if !strings.Contains(err.Error(), "download page failed after 3 retries") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestDownloadPage_DASHDanmakuDownload(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig(), DownloadDanmaku: true, DownloadDanmakuFormats: []string{"ass", "xml"}}

	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		return &entity.ParsedResult{
			VideoTracks: []entity.Video{{ID: "1", Dfn: "1080P", BaseUrl: "https://example.com/video.m4v"}},
		}, nil
	}

	danmakuXML := `<?xml version="1.0" encoding="UTF-8"?><i><d p="0,1,25,16777215,0,0,0,0">Hello</d></i>`
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			if strings.Contains(url, "comment.bilibili.com") {
				return &http.Response{Body: io.NopCloser(strings.NewReader(danmakuXML)), StatusCode: 200}, nil
			}
			return &http.Response{Body: io.NopCloser(strings.NewReader("{}")), StatusCode: 200}, nil
		},
	}

	downloader := &mockDownloader{
		downloadMTFunc: func(ctx context.Context, url, path string, opts download.Options) error {
			return os.WriteFile(path, []byte("data"), 0o644)
		},
	}

	deps := DownloadDeps{HTTPClient: client, Logger: discardLogger(), ExtractTracks: extractTracks, Downloader: downloader, Muxer: &mockMuxer{}}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// XML should be kept
	xmlPath := filepath.Join(tmpDir, "123", "123.xml")
	if _, err := os.Stat(xmlPath); os.IsNotExist(err) {
		t.Errorf("expected danmaku xml to exist at %s", xmlPath)
	}

	// ASS should be generated
	assPath := filepath.Join(tmpDir, "123", "123.ass")
	if _, err := os.Stat(assPath); os.IsNotExist(err) {
		t.Errorf("expected danmaku ass to exist at %s", assPath)
	}
}

func TestDownloadPage_DASHDanmakuAssOnly(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig(), DownloadDanmaku: true, DownloadDanmakuFormats: []string{"ass"}}

	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		return &entity.ParsedResult{
			VideoTracks: []entity.Video{{ID: "1", Dfn: "1080P", BaseUrl: "https://example.com/video.m4v"}},
		}, nil
	}

	danmakuXML := `<?xml version="1.0" encoding="UTF-8"?><i><d p="0,1,25,16777215,0,0,0,0">Hello</d></i>`
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			if strings.Contains(url, "comment.bilibili.com") {
				return &http.Response{Body: io.NopCloser(strings.NewReader(danmakuXML)), StatusCode: 200}, nil
			}
			return &http.Response{Body: io.NopCloser(strings.NewReader("{}")), StatusCode: 200}, nil
		},
	}

	downloader := &mockDownloader{
		downloadMTFunc: func(ctx context.Context, url, path string, opts download.Options) error {
			return os.WriteFile(path, []byte("data"), 0o644)
		},
	}

	deps := DownloadDeps{HTTPClient: client, Logger: discardLogger(), ExtractTracks: extractTracks, Downloader: downloader, Muxer: &mockMuxer{}}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// XML should be deleted
	xmlPath := filepath.Join(tmpDir, "123", "123.xml")
	if _, err := os.Stat(xmlPath); !os.IsNotExist(err) {
		t.Errorf("expected danmaku xml to be deleted")
	}

	// ASS should be generated
	assPath := filepath.Join(tmpDir, "123", "123.ass")
	if _, err := os.Stat(assPath); os.IsNotExist(err) {
		t.Errorf("expected danmaku ass to exist at %s", assPath)
	}
}

func TestDownloadPage_DASHCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		return &entity.ParsedResult{
			VideoTracks: []entity.Video{{ID: "1", Dfn: "1080P", BaseUrl: "https://example.com/video.m4v"}},
			AudioTracks: []entity.Audio{{ID: "1", Dfn: "320K", BaseUrl: "https://example.com/audio.m4a"}},
		}, nil
	}

	downloader := &mockDownloader{
		downloadMTFunc: func(ctx context.Context, url, path string, opts download.Options) error {
			return os.WriteFile(path, []byte("data"), 0o644)
		},
	}

	deps := DownloadDeps{HTTPClient: &mockClient{}, Logger: discardLogger(), ExtractTracks: extractTracks, Downloader: downloader, Muxer: &mockMuxer{}}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Temp files should be cleaned up
	videoPath := filepath.Join(tmpDir, "123", "123.m4v")
	if _, err := os.Stat(videoPath); !os.IsNotExist(err) {
		t.Errorf("expected video temp file to be cleaned up")
	}
	audioPath := filepath.Join(tmpDir, "123", "123.m4a")
	if _, err := os.Stat(audioPath); !os.IsNotExist(err) {
		t.Errorf("expected audio temp file to be cleaned up")
	}
}

func TestDownloadPage_FLVCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	opt := cli.NewOption()
	vInfo := &entity.VInfo{Title: "Test Video"}
	p := &entity.Page{Index: 1, Aid: "123", Cid: "456", Title: "P1"}
	workCfg := &WorkConfig{SavePathFormat: "test", Config: config.NewConfig()}

	extractTracks := func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error) {
		return &entity.ParsedResult{
			Clips:       []string{"https://example.com/clip1.flv"},
			Dfns:        []string{"80"},
			VideoTracks: []entity.Video{{ID: "80", Dfn: "1080P"}},
		}, nil
	}

	downloader := &mockDownloader{
		downloadMTFunc: func(ctx context.Context, url, path string, opts download.Options) error {
			return os.WriteFile(path, []byte("data"), 0o644)
		},
	}

	muxerMock := &mockMuxer{
		mergeFLVFunc: func(ctx context.Context, files []string, outPath string) error {
			return os.WriteFile(outPath, []byte("merged"), 0o644)
		},
	}

	deps := DownloadDeps{HTTPClient: &mockClient{}, Logger: discardLogger(), ExtractTracks: extractTracks, Downloader: downloader, Muxer: muxerMock}
	err := DownloadPage(context.Background(), p, opt, vInfo, []entity.Page{*p}, workCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	clipPath := filepath.Join(tmpDir, "123", "123_clip0.flv")
	if _, err := os.Stat(clipPath); !os.IsNotExist(err) {
		t.Errorf("expected clip temp file to be cleaned up")
	}
	mergedPath := filepath.Join(tmpDir, "123", "123.merged.flv")
	if _, err := os.Stat(mergedPath); !os.IsNotExist(err) {
		t.Errorf("expected merged temp file to be cleaned up")
	}
}
