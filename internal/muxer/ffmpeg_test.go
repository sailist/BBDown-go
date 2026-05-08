package muxer

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sailist/BBDown-go/internal/core/entity"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

func debugLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestFFmpegMuxer_BuildArgs_BasicVideoAudio(t *testing.T) {
	m := &FFmpegMuxer{logger: discardLogger()}
	cfg := MuxConfig{
		BVid:      "BV1xx411c7mD",
		VideoPath: "/tmp/video.m4s",
		AudioPath: "/tmp/audio.m4s",
		OutPath:   "/tmp/out.mp4",
		Title:     "Test Title",
		Desc:      "Test Desc",
		Author:    "Author",
		Lang:      "chi",
	}

	args, err := m.buildFFmpegArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "-i /tmp/video.m4s") {
		t.Errorf("expected video input, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-i /tmp/audio.m4s") {
		t.Errorf("expected audio input, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-metadata title=Test Title") {
		t.Errorf("expected title metadata, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-metadata comment=https://www.bilibili.com/video/BV1xx411c7mD/") {
		t.Errorf("expected comment metadata, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-metadata:s:a:0 language=chi") {
		t.Errorf("expected audio language metadata, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-metadata artist=Author") {
		t.Errorf("expected artist metadata, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-c:v copy") {
		t.Errorf("expected video copy, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-c:a copy") {
		t.Errorf("expected audio copy, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-- /tmp/out.mp4") {
		t.Errorf("expected output path, got: %s", argsStr)
	}
}

func TestFFmpegMuxer_BuildArgs_AudioOnly(t *testing.T) {
	m := &FFmpegMuxer{logger: discardLogger()}
	cfg := MuxConfig{
		BVid:      "BV1xx411c7mD",
		VideoPath: "/tmp/video.m4s",
		AudioPath: "/tmp/audio.m4s",
		OutPath:   "/tmp/out.mp4",
		Title:     "Title",
		AudioOnly: true,
	}

	args, err := m.buildFFmpegArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if strings.Contains(argsStr, "-i /tmp/video.m4s") {
		t.Errorf("expected no video input for audioOnly, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-i /tmp/audio.m4s") {
		t.Errorf("expected audio input, got: %s", argsStr)
	}
}

func TestFFmpegMuxer_BuildArgs_VideoOnly(t *testing.T) {
	m := &FFmpegMuxer{logger: discardLogger()}
	cfg := MuxConfig{
		BVid:      "BV1xx411c7mD",
		VideoPath: "/tmp/video.m4s",
		AudioPath: "/tmp/audio.m4s",
		OutPath:   "/tmp/out.mp4",
		Title:     "Title",
		VideoOnly: true,
	}

	args, err := m.buildFFmpegArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "-i /tmp/video.m4s") {
		t.Errorf("expected video input, got: %s", argsStr)
	}
	if strings.Contains(argsStr, "-i /tmp/audio.m4s") {
		t.Errorf("expected no audio input for videoOnly, got: %s", argsStr)
	}
}

func TestFFmpegMuxer_BuildArgs_AudioMaterial(t *testing.T) {
	m := &FFmpegMuxer{logger: discardLogger()}
	cfg := MuxConfig{
		BVid:      "BV1xx411c7mD",
		VideoPath: "/tmp/video.m4s",
		AudioPath: "/tmp/audio.m4s",
		AudioMaterial: []entity.AudioMaterial{
			{Title: "Commentary", PersonName: "Commentator", Path: "/tmp/commentary.m4s"},
		},
		OutPath: "/tmp/out.mp4",
		Title:   "Title",
	}

	args, err := m.buildFFmpegArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "-metadata:s:a:0 title=原音频") {
		t.Errorf("expected original audio title, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-metadata:s:a:1 title=Commentary") {
		t.Errorf("expected commentary title, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-metadata:s:a:1 artist=Commentator") {
		t.Errorf("expected commentary artist, got: %s", argsStr)
	}
}

func TestFFmpegMuxer_BuildArgs_WithPic(t *testing.T) {
	m := &FFmpegMuxer{logger: discardLogger()}
	cfg := MuxConfig{
		BVid:      "BV1xx411c7mD",
		VideoPath: "/tmp/video.m4s",
		AudioPath: "/tmp/audio.m4s",
		Pic:       "/tmp/cover.jpg",
		OutPath:   "/tmp/out.mp4",
		Title:     "Title",
	}

	args, err := m.buildFFmpegArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "-i /tmp/cover.jpg") {
		t.Errorf("expected pic input, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-disposition:v:1 attached_pic") {
		t.Errorf("expected attached_pic disposition, got: %s", argsStr)
	}
}

func TestFFmpegMuxer_BuildArgs_WithPicAudioOnly(t *testing.T) {
	m := &FFmpegMuxer{logger: discardLogger()}
	cfg := MuxConfig{
		BVid:      "BV1xx411c7mD",
		VideoPath: "/tmp/video.m4s",
		AudioPath: "/tmp/audio.m4s",
		Pic:       "/tmp/cover.jpg",
		OutPath:   "/tmp/out.mp4",
		Title:     "Title",
		AudioOnly: true,
	}

	args, err := m.buildFFmpegArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "-disposition:v:0 attached_pic") {
		t.Errorf("expected attached_pic disposition for audioOnly, got: %s", argsStr)
	}
}

func TestFFmpegMuxer_BuildArgs_WithSubtitles(t *testing.T) {
	dir := t.TempDir()
	sub1 := filepath.Join(dir, "sub1.srt")
	sub2 := filepath.Join(dir, "sub2.srt")
	os.WriteFile(sub1, []byte("1\n00:00:00,000 --> 00:00:01,000\nHello\n"), 0o644)
	os.WriteFile(sub2, []byte(""), 0o644) // empty, should be skipped

	m := &FFmpegMuxer{logger: discardLogger()}
	cfg := MuxConfig{
		BVid:      "BV1xx411c7mD",
		VideoPath: "/tmp/video.m4s",
		AudioPath: "/tmp/audio.m4s",
		OutPath:   "/tmp/out.mp4",
		Title:     "Title",
		Subtitles: []entity.Subtitle{
			{Lan: "zh-CN", Path: sub1},
			{Lan: "en-US", Path: sub2},
		},
	}

	args, err := m.buildFFmpegArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "-i "+sub1) {
		t.Errorf("expected subtitle input, got: %s", argsStr)
	}
	if strings.Contains(argsStr, "-i "+sub2) {
		t.Errorf("expected empty subtitle to be skipped, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-metadata:s:s:0 title=Chinese (Simplified)") {
		t.Errorf("expected subtitle title metadata, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-metadata:s:s:0 language=chi") {
		t.Errorf("expected subtitle language metadata, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-c:s mov_text") {
		t.Errorf("expected mov_text codec, got: %s", argsStr)
	}
}

func TestFFmpegMuxer_BuildArgs_WithPoints(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "video.m4s")
	audioPath := filepath.Join(dir, "audio.m4s")
	os.WriteFile(videoPath, []byte("video"), 0o644)
	os.WriteFile(audioPath, []byte("audio"), 0o644)

	m := &FFmpegMuxer{logger: discardLogger()}
	cfg := MuxConfig{
		BVid:      "BV1xx411c7mD",
		VideoPath: videoPath,
		AudioPath: audioPath,
		OutPath:   filepath.Join(dir, "out.mp4"),
		Title:     "Title",
		Points: []entity.ViewPoint{
			{Title: "Intro", Start: 0, End: 10},
			{Title: "Main", Start: 10, End: 100},
		},
	}

	args, err := m.buildFFmpegArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "-map_chapters 2") {
		t.Errorf("expected map_chapters, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-map 0") {
		t.Errorf("expected map 0, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-map 1") {
		t.Errorf("expected map 1, got: %s", argsStr)
	}
	if strings.Contains(argsStr, "-map 2") {
		// chapters file should not be mapped as streams
		t.Errorf("expected no map for chapter file, got: %s", argsStr)
	}

	// Verify chapter file was written
	chapterFile := filepath.Join(dir, "chapters")
	data, err := os.ReadFile(chapterFile)
	if err != nil {
		t.Fatalf("expected chapter file to exist: %v", err)
	}
	if !strings.Contains(string(data), "[CHAPTER]") {
		t.Errorf("expected FFMETADATA format, got: %s", string(data))
	}
}

func TestFFmpegMuxer_BuildArgs_SimplyMux(t *testing.T) {
	m := &FFmpegMuxer{logger: discardLogger()}
	cfg := MuxConfig{
		BVid:      "BV1xx411c7mD",
		VideoPath: "/tmp/video.m4s",
		AudioPath: "/tmp/audio.m4s",
		OutPath:   "/tmp/out.mp4",
		Title:     "Title",
		Desc:      "Desc",
		Author:    "Author",
		SimplyMux: true,
	}

	args, err := m.buildFFmpegArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if strings.Contains(argsStr, "-metadata title=") {
		t.Errorf("expected no title metadata for simplyMux, got: %s", argsStr)
	}
	if strings.Contains(argsStr, "-metadata comment=") {
		t.Errorf("expected no comment metadata for simplyMux, got: %s", argsStr)
	}
	if strings.Contains(argsStr, "-metadata artist=") {
		t.Errorf("expected no artist metadata for simplyMux, got: %s", argsStr)
	}
}

func TestFFmpegMuxer_BuildArgs_EpisodeID(t *testing.T) {
	m := &FFmpegMuxer{logger: discardLogger()}
	cfg := MuxConfig{
		BVid:      "BV1xx411c7mD",
		VideoPath: "/tmp/video.m4s",
		AudioPath: "/tmp/audio.m4s",
		OutPath:   "/tmp/out.mp4",
		Title:     "Series Title",
		EpisodeID: "EP01",
	}

	args, err := m.buildFFmpegArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "-metadata title=EP01") {
		t.Errorf("expected episodeID as title, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-metadata album=Series Title") {
		t.Errorf("expected album metadata, got: %s", argsStr)
	}
}

func TestFFmpegMuxer_BuildArgs_PubTime(t *testing.T) {
	m := &FFmpegMuxer{logger: discardLogger()}
	cfg := MuxConfig{
		BVid:      "BV1xx411c7mD",
		VideoPath: "/tmp/video.m4s",
		AudioPath: "/tmp/audio.m4s",
		OutPath:   "/tmp/out.mp4",
		Title:     "Title",
		PubTime:   1609459200, // 2021-01-01T00:00:00Z
	}

	args, err := m.buildFFmpegArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "-metadata creation_time=2021-01-01T00:00:00.000000Z") {
		t.Errorf("expected creation_time metadata, got: %s", argsStr)
	}
}

func TestFFmpegMuxer_BuildArgs_HEVC_macOS(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping macOS-specific test")
	}

	m := &FFmpegMuxer{logger: discardLogger()}
	cfg := MuxConfig{
		BVid:      "BV1xx411c7mD",
		VideoPath: "/tmp/video.m4s",
		AudioPath: "/tmp/audio.m4s",
		OutPath:   "/tmp/out.mp4",
		Title:     "Title",
		IsHevc:    true,
	}

	args, err := m.buildFFmpegArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "-tag:v:0 hvc1") {
		t.Errorf("expected hvc1 tag, got: %s", argsStr)
	}
}

func TestFFmpegMuxer_BuildArgs_DebugLog(t *testing.T) {
	m := &FFmpegMuxer{logger: debugLogger()}
	cfg := MuxConfig{
		BVid:      "BV1xx411c7mD",
		VideoPath: "/tmp/video.m4s",
		AudioPath: "/tmp/audio.m4s",
		OutPath:   "/tmp/out.mp4",
		Title:     "Title",
	}

	args, err := m.buildFFmpegArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "-loglevel verbose") {
		t.Errorf("expected verbose loglevel, got: %s", argsStr)
	}
}

func TestFFmpegMuxer_MergeFLV_SingleFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "1.flv")
	dst := filepath.Join(dir, "out.mp4")
	os.WriteFile(src, []byte("flvdata"), 0o644)

	m := NewFFmpegMuxer(discardLogger())
	err := m.MergeFLV(context.Background(), []string{src}, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	if string(data) != "flvdata" {
		t.Errorf("expected flvdata, got %q", string(data))
	}
	if _, err := os.Stat(src); err == nil {
		t.Error("expected source file to be moved")
	}
}

func TestFFmpegMuxer_MergeFLV_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	flv1 := filepath.Join(dir, "1.flv")
	flv2 := filepath.Join(dir, "2.flv")
	os.WriteFile(flv1, []byte("aaa"), 0o644)
	os.WriteFile(flv2, []byte("bbb"), 0o644)
	outPath := filepath.Join(dir, "out.mp4")

	origExec := execCommandContext
	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		// Mock ffmpeg: copy input (-i arg) to output (last arg)
		var input string
		for i := 0; i < len(arg); i++ {
			if arg[i] == "-i" && i+1 < len(arg) {
				input = arg[i+1]
			}
		}
		output := arg[len(arg)-1]
		return origExec(ctx, "cp", input, output)
	}
	defer func() { execCommandContext = origExec }()

	origFind := FindExecutable
	FindExecutable = func(name string) (string, error) { return "ffmpeg", nil }
	defer func() { FindExecutable = origFind }()

	m := NewFFmpegMuxer(discardLogger())
	err := m.MergeFLV(context.Background(), []string{flv1, flv2}, outPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	if string(data) != "aaabbb" {
		t.Errorf("expected concatenated content 'aaabbb', got %q", string(data))
	}
	if _, err := os.Stat(flv1); err == nil {
		t.Error("expected flv1 to be deleted")
	}
	if _, err := os.Stat(flv2); err == nil {
		t.Error("expected flv2 to be deleted")
	}
}
