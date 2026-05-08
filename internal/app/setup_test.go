package app

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sailist/BBDown-go/internal/cli"
	"github.com/sailist/BBDown-go/internal/config"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

func TestSetupWork_Basic(t *testing.T) {
	opt := cli.NewOption()
	opt.URL = "https://www.bilibili.com/video/BV1xx411c7mD"
	opt.SkipMux = true

	wc, err := SetupWork(opt, discardLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if wc.Input != opt.URL {
		t.Errorf("expected input %q, got %q", opt.URL, wc.Input)
	}
	if wc.SavePathFormat != cli.SinglePageDefaultSavePath {
		t.Errorf("expected save path %q, got %q", cli.SinglePageDefaultSavePath, wc.SavePathFormat)
	}
	if wc.Delay != 0 {
		t.Errorf("expected delay 0, got %d", wc.Delay)
	}
	if wc.Config == nil {
		t.Fatal("expected non-nil Config")
	}
	if wc.Config.Host != "api.bilibili.com" {
		t.Errorf("expected host api.bilibili.com, got %q", wc.Config.Host)
	}
}

func TestSetupWork_DelayParsing(t *testing.T) {
	opt := cli.NewOption()
	opt.URL = "https://example.com"
	opt.DelayPerPage = "5"
	opt.SkipMux = true

	wc, err := SetupWork(opt, discardLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wc.Delay != 5 {
		t.Errorf("expected delay 5, got %d", wc.Delay)
	}
}

func TestSetupWork_AccessToken(t *testing.T) {
	opt := cli.NewOption()
	opt.URL = "https://example.com"
	opt.AccessToken = "access_token=abc123"
	opt.SkipMux = true

	wc, err := SetupWork(opt, discardLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wc.Config.Token != "abc123" {
		t.Errorf("expected token abc123, got %q", wc.Config.Token)
	}
}

// Deprecated options tests

func TestHandleDeprecatedOptions_AddDfnSubfix(t *testing.T) {
	opt := cli.NewOption()
	opt.AddDfnSubfix = true

	handleDeprecatedOptions(opt, discardLogger())

	expected := cli.SinglePageDefaultSavePath + "[<dfn>]"
	if opt.FilePattern != expected {
		t.Errorf("expected FilePattern %q, got %q", expected, opt.FilePattern)
	}
	expectedMulti := cli.MultiPageDefaultSavePath + "[<dfn>]"
	if opt.MultiFilePattern != expectedMulti {
		t.Errorf("expected MultiFilePattern %q, got %q", expectedMulti, opt.MultiFilePattern)
	}
}

func TestHandleDeprecatedOptions_AddDfnSubfix_WithExistingPattern(t *testing.T) {
	opt := cli.NewOption()
	opt.AddDfnSubfix = true
	opt.FilePattern = "custom"

	handleDeprecatedOptions(opt, discardLogger())

	if opt.FilePattern != "custom" {
		t.Errorf("expected FilePattern unchanged, got %q", opt.FilePattern)
	}
}

func TestHandleDeprecatedOptions_Aria2cProxy(t *testing.T) {
	opt := cli.NewOption()
	opt.Aria2cProxy = "http://proxy:8080"
	opt.Aria2cArgs = "--max-concurrent-downloads=5"

	handleDeprecatedOptions(opt, discardLogger())

	if !strings.Contains(opt.Aria2cArgs, "--all-proxy=\"http://proxy:8080\"") {
		t.Errorf("expected aria2c args to contain proxy, got %q", opt.Aria2cArgs)
	}
}

func TestHandleDeprecatedOptions_OnlyHevc(t *testing.T) {
	opt := cli.NewOption()
	opt.OnlyHevc = true

	handleDeprecatedOptions(opt, discardLogger())

	if opt.EncodingPriority != "hevc" {
		t.Errorf("expected EncodingPriority hevc, got %q", opt.EncodingPriority)
	}
}

func TestHandleDeprecatedOptions_OnlyAvc(t *testing.T) {
	opt := cli.NewOption()
	opt.OnlyAvc = true

	handleDeprecatedOptions(opt, discardLogger())

	if opt.EncodingPriority != "avc" {
		t.Errorf("expected EncodingPriority avc, got %q", opt.EncodingPriority)
	}
}

func TestHandleDeprecatedOptions_OnlyAv1(t *testing.T) {
	opt := cli.NewOption()
	opt.OnlyAv1 = true

	handleDeprecatedOptions(opt, discardLogger())

	if opt.EncodingPriority != "av1" {
		t.Errorf("expected EncodingPriority av1, got %q", opt.EncodingPriority)
	}
}

func TestHandleDeprecatedOptions_NoPaddingPageNum(t *testing.T) {
	opt := cli.NewOption()
	opt.NoPaddingPageNum = true

	handleDeprecatedOptions(opt, discardLogger())

	expected := strings.ReplaceAll(cli.MultiPageDefaultSavePath, "<pageNumberWithZero>", "<pageNumber>")
	if opt.MultiFilePattern != expected {
		t.Errorf("expected MultiFilePattern %q, got %q", expected, opt.MultiFilePattern)
	}
}

func TestHandleDeprecatedOptions_BandwithAscending(t *testing.T) {
	opt := cli.NewOption()
	opt.BandwithAscending = true

	handleDeprecatedOptions(opt, discardLogger())

	if !opt.VideoAscending {
		t.Error("expected VideoAscending to be true")
	}
	if !opt.AudioAscending {
		t.Error("expected AudioAscending to be true")
	}
}

// Conflicting options tests

func TestHandleConflictingOptions_InteractiveHideStreams(t *testing.T) {
	opt := cli.NewOption()
	opt.Interactive = true
	opt.HideStreams = true

	handleConflictingOptions(opt)

	if opt.HideStreams {
		t.Error("expected HideStreams to be false when Interactive is true")
	}
}

func TestHandleConflictingOptions_AudioOnlyVideoOnly(t *testing.T) {
	opt := cli.NewOption()
	opt.AudioOnly = true
	opt.VideoOnly = true

	handleConflictingOptions(opt)

	if opt.AudioOnly {
		t.Error("expected AudioOnly to be false")
	}
	if opt.VideoOnly {
		t.Error("expected VideoOnly to be false")
	}
}

func TestHandleConflictingOptions_SkipSubtitleSubOnly(t *testing.T) {
	opt := cli.NewOption()
	opt.SkipSubtitle = true
	opt.SubOnly = true

	handleConflictingOptions(opt)

	if opt.SubOnly {
		t.Error("expected SubOnly to be false when SkipSubtitle is true")
	}
}

// Encoding / dfn priority parsing tests

func TestParseEncodingPriority_Empty(t *testing.T) {
	m, first := ParseEncodingPriority("")
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
	if first != "" {
		t.Errorf("expected empty first encoding, got %q", first)
	}
}

func TestParseEncodingPriority_Basic(t *testing.T) {
	m, first := ParseEncodingPriority("hevc,avc,av1")
	if len(m) != 3 {
		t.Errorf("expected 3 entries, got %d", len(m))
	}
	if first != "HEVC" {
		t.Errorf("expected first HEVC, got %q", first)
	}
	if m["HEVC"] != 0 {
		t.Errorf("expected HEVC priority 0, got %d", m["HEVC"])
	}
	if m["AVC"] != 1 {
		t.Errorf("expected AVC priority 1, got %d", m["AVC"])
	}
	if m["AV1"] != 2 {
		t.Errorf("expected AV1 priority 2, got %d", m["AV1"])
	}
}

func TestParseEncodingPriority_WithHyphens(t *testing.T) {
	m, first := ParseEncodingPriority("hevc-10bit,avc")
	if first != "HEVC10BIT" {
		t.Errorf("expected first HEVC10BIT, got %q", first)
	}
	if _, ok := m["HEVC10BIT"]; !ok {
		t.Errorf("expected HEVC10BIT in map")
	}
}

func TestParseEncodingPriority_Dedup(t *testing.T) {
	m, _ := ParseEncodingPriority("hevc,hevc,avc")
	if len(m) != 2 {
		t.Errorf("expected 2 entries after dedup, got %d", len(m))
	}
}

func TestParseEncodingPriority_ChineseComma(t *testing.T) {
	m, _ := ParseEncodingPriority("hevc，avc")
	if len(m) != 2 {
		t.Errorf("expected 2 entries, got %d", len(m))
	}
}

func TestParseDfnPriority_Empty(t *testing.T) {
	m := ParseDfnPriority("")
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestParseDfnPriority_Basic(t *testing.T) {
	m := ParseDfnPriority("1080P,720P,360P")
	if len(m) != 3 {
		t.Errorf("expected 3 entries, got %d", len(m))
	}
	if m["1080P"] != 0 {
		t.Errorf("expected 1080P priority 0, got %d", m["1080P"])
	}
	if m["720P"] != 1 {
		t.Errorf("expected 720P priority 1, got %d", m["720P"])
	}
	if m["360P"] != 2 {
		t.Errorf("expected 360P priority 2, got %d", m["360P"])
	}
}

func TestParseDfnPriority_Dedup(t *testing.T) {
	m := ParseDfnPriority("1080P,1080P,720P")
	if len(m) != 2 {
		t.Errorf("expected 2 entries after dedup, got %d", len(m))
	}
}

func TestParseDfnPriority_ChineseComma(t *testing.T) {
	m := ParseDfnPriority("1080P，720P")
	if len(m) != 2 {
		t.Errorf("expected 2 entries, got %d", len(m))
	}
}

func TestParseDfnPriority_MixedCase(t *testing.T) {
	m := ParseDfnPriority("1080p,720P")
	if _, ok := m["1080P"]; !ok {
		t.Errorf("expected 1080P (uppercase) in map")
	}
}

// Working dir change tests

func TestChangeWorkingDir(t *testing.T) {
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer os.Chdir(origWd)

	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub", "dir")

	opt := &cli.Option{WorkDir: subDir}
	if err := changeWorkingDir(opt, discardLogger()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd after chdir: %v", err)
	}
	// On macOS, /var is a symlink to /private/var; normalize both sides.
	resolvedWd, _ := filepath.EvalSymlinks(wd)
	resolvedSubDir, _ := filepath.EvalSymlinks(subDir)
	if resolvedWd != resolvedSubDir {
		t.Errorf("expected wd %q, got %q", resolvedSubDir, resolvedWd)
	}
}

func TestChangeWorkingDir_ExpandEnv(t *testing.T) {
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer os.Chdir(origWd)

	tmpDir := t.TempDir()
	t.Setenv("BBDOWN_TEST_DIR", tmpDir)

	opt := &cli.Option{WorkDir: "$BBDOWN_TEST_DIR/sub"}
	if err := changeWorkingDir(opt, discardLogger()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd after chdir: %v", err)
	}
	expected := filepath.Join(tmpDir, "sub")
	resolvedWd, _ := filepath.EvalSymlinks(wd)
	resolvedExpected, _ := filepath.EvalSymlinks(expected)
	if resolvedWd != resolvedExpected {
		t.Errorf("expected wd %q, got %q", resolvedExpected, resolvedWd)
	}
}

func TestChangeWorkingDir_Empty(t *testing.T) {
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer os.Chdir(origWd)

	opt := &cli.Option{WorkDir: ""}
	if err := changeWorkingDir(opt, discardLogger()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if wd != origWd {
		t.Errorf("expected wd unchanged")
	}
}

// Credentials loading tests

func TestLoadCredentials_Cookie(t *testing.T) {
	tmpDir := t.TempDir()
	origAppDir := appDir
	appDir = tmpDir
	defer func() { appDir = origAppDir }()

	cookiePath := filepath.Join(tmpDir, "BBDown.data")
	if err := os.WriteFile(cookiePath, []byte("testcookie"), 0o644); err != nil {
		t.Fatalf("write cookie file: %v", err)
	}

	opt := cli.NewOption()
	cfg := config.NewConfig()
	loadCredentials(opt, cfg, discardLogger())

	if cfg.Cookie != "testcookie" {
		t.Errorf("expected cookie testcookie, got %q", cfg.Cookie)
	}
}

func TestLoadCredentials_TvToken(t *testing.T) {
	tmpDir := t.TempDir()
	origAppDir := appDir
	appDir = tmpDir
	defer func() { appDir = origAppDir }()

	tokenPath := filepath.Join(tmpDir, "BBDownTV.data")
	if err := os.WriteFile(tokenPath, []byte("access_token=tvtoken"), 0o644); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	opt := cli.NewOption()
	opt.UseTvApi = true
	cfg := config.NewConfig()
	loadCredentials(opt, cfg, discardLogger())

	if cfg.Token != "tvtoken" {
		t.Errorf("expected token tvtoken, got %q", cfg.Token)
	}
}

func TestLoadCredentials_AppToken(t *testing.T) {
	tmpDir := t.TempDir()
	origAppDir := appDir
	appDir = tmpDir
	defer func() { appDir = origAppDir }()

	tokenPath := filepath.Join(tmpDir, "BBDownApp.data")
	if err := os.WriteFile(tokenPath, []byte("access_token=apptoken"), 0o644); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	opt := cli.NewOption()
	opt.UseAppApi = true
	cfg := config.NewConfig()
	loadCredentials(opt, cfg, discardLogger())

	if cfg.Token != "apptoken" {
		t.Errorf("expected token apptoken, got %q", cfg.Token)
	}
}

func TestLoadCredentials_Priority(t *testing.T) {
	// When cookie is already set in config, file should not override
	tmpDir := t.TempDir()
	origAppDir := appDir
	appDir = tmpDir
	defer func() { appDir = origAppDir }()

	cookiePath := filepath.Join(tmpDir, "BBDown.data")
	if err := os.WriteFile(cookiePath, []byte("filecookie"), 0o644); err != nil {
		t.Fatalf("write cookie file: %v", err)
	}

	opt := cli.NewOption()
	cfg := config.NewConfig()
	cfg.Cookie = "existingcookie"
	loadCredentials(opt, cfg, discardLogger())

	if cfg.Cookie != "existingcookie" {
		t.Errorf("expected cookie to remain existingcookie, got %q", cfg.Cookie)
	}
}

func TestLoadCredentials_TvBeforeApp(t *testing.T) {
	// When both TV and App token files exist, TV should take precedence
	// because TV is checked first when UseTvApi is true
	tmpDir := t.TempDir()
	origAppDir := appDir
	appDir = tmpDir
	defer func() { appDir = origAppDir }()

	tvPath := filepath.Join(tmpDir, "BBDownTV.data")
	if err := os.WriteFile(tvPath, []byte("access_token=tvtoken"), 0o644); err != nil {
		t.Fatalf("write tv token file: %v", err)
	}

	opt := cli.NewOption()
	opt.UseTvApi = true
	cfg := config.NewConfig()
	loadCredentials(opt, cfg, discardLogger())

	if cfg.Token != "tvtoken" {
		t.Errorf("expected token tvtoken, got %q", cfg.Token)
	}
}

// Danmaku formats tests

func TestParseDownloadDanmakuFormats_Empty(t *testing.T) {
	f := parseDownloadDanmakuFormats("")
	if len(f) != 2 || f[0] != "ass" || f[1] != "xml" {
		t.Errorf("expected default [ass xml], got %v", f)
	}
}

func TestParseDownloadDanmakuFormats_Custom(t *testing.T) {
	f := parseDownloadDanmakuFormats("ass")
	if len(f) != 1 || f[0] != "ass" {
		t.Errorf("expected [ass], got %v", f)
	}
}

func TestParseDownloadDanmakuFormats_Multiple(t *testing.T) {
	f := parseDownloadDanmakuFormats("ass,xml,protobuf")
	if len(f) != 3 {
		t.Errorf("expected 3 formats, got %d", len(f))
	}
}

func TestParseDownloadDanmakuFormats_ChineseComma(t *testing.T) {
	f := parseDownloadDanmakuFormats("ass，xml")
	if len(f) != 2 || f[0] != "ass" || f[1] != "xml" {
		t.Errorf("expected [ass xml], got %v", f)
	}
}

func TestSetupWork_DownloadDanmaku(t *testing.T) {
	opt := cli.NewOption()
	opt.URL = "https://example.com"
	opt.DownloadDanmaku = true
	opt.DownloadDanmakuFormats = "ass"
	opt.SkipMux = true

	wc, err := SetupWork(opt, discardLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !wc.DownloadDanmaku {
		t.Error("expected DownloadDanmaku to be true")
	}
	if len(wc.DownloadDanmakuFormats) != 1 || wc.DownloadDanmakuFormats[0] != "ass" {
		t.Errorf("expected [ass], got %v", wc.DownloadDanmakuFormats)
	}
}

func TestSetupWork_DanmakuOnly(t *testing.T) {
	opt := cli.NewOption()
	opt.URL = "https://example.com"
	opt.DanmakuOnly = true
	opt.SkipMux = true

	wc, err := SetupWork(opt, discardLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !wc.DownloadDanmaku {
		t.Error("expected DownloadDanmaku to be true when DanmakuOnly is set")
	}
}
