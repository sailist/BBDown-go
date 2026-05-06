package muxer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nilaonai/bbdown-go/internal/core/entity"
)

func TestMP4BoxMuxer_BuildArgs_BasicVideoAudio(t *testing.T) {
	m := &MP4BoxMuxer{logger: discardLogger()}
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

	args, err := m.buildMP4BoxArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "-add /tmp/video.m4s#trackID=1:name=") {
		t.Errorf("expected video add, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-add /tmp/audio.m4s:lang=chi") {
		t.Errorf("expected audio add with lang, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-itags tool=") {
		t.Errorf("expected itags, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, ":title=Test Title") {
		t.Errorf("expected title in itags, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, ":sdesc=Test Desc") {
		t.Errorf("expected sdesc in itags, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, ":artist=Author") {
		t.Errorf("expected artist in itags, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-new") {
		t.Errorf("expected -new flag, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-- /tmp/out.mp4") {
		t.Errorf("expected output path, got: %s", argsStr)
	}
}

func TestMP4BoxMuxer_BuildArgs_AudioOnlyNoAudio(t *testing.T) {
	m := &MP4BoxMuxer{logger: discardLogger()}
	cfg := MuxConfig{
		BVid:      "BV1xx411c7mD",
		VideoPath: "/tmp/video.m4s",
		AudioPath: "",
		OutPath:   "/tmp/out.mp4",
		Title:     "Title",
		AudioOnly: true,
	}

	args, err := m.buildMP4BoxArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "#trackID=2:name=") {
		t.Errorf("expected trackID=2 for audioOnly with no audio, got: %s", argsStr)
	}
}

func TestMP4BoxMuxer_BuildArgs_DefaultLang(t *testing.T) {
	m := &MP4BoxMuxer{logger: discardLogger()}
	cfg := MuxConfig{
		BVid:      "BV1xx411c7mD",
		VideoPath: "/tmp/video.m4s",
		AudioPath: "/tmp/audio.m4s",
		OutPath:   "/tmp/out.mp4",
		Title:     "Title",
		Lang:      "",
	}

	args, err := m.buildMP4BoxArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "-add /tmp/audio.m4s:lang=und") {
		t.Errorf("expected default lang und, got: %s", argsStr)
	}
}

func TestMP4BoxMuxer_BuildArgs_WithPoints(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "video.m4s")
	audioPath := filepath.Join(dir, "audio.m4s")
	os.WriteFile(videoPath, []byte("video"), 0o644)
	os.WriteFile(audioPath, []byte("audio"), 0o644)

	m := &MP4BoxMuxer{logger: discardLogger()}
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

	args, err := m.buildMP4BoxArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "-chap") {
		t.Errorf("expected chap flag, got: %s", argsStr)
	}

	chapterFile := filepath.Join(dir, "chapters")
	data, err := os.ReadFile(chapterFile)
	if err != nil {
		t.Fatalf("expected chapter file to exist: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "00:00:00 Intro") {
		t.Errorf("expected chapter format, got: %s", content)
	}
	if !strings.Contains(content, "00:00:10 Main") {
		t.Errorf("expected chapter format, got: %s", content)
	}
}

func TestMP4BoxMuxer_BuildArgs_WithPic(t *testing.T) {
	m := &MP4BoxMuxer{logger: discardLogger()}
	cfg := MuxConfig{
		BVid:      "BV1xx411c7mD",
		VideoPath: "/tmp/video.m4s",
		AudioPath: "/tmp/audio.m4s",
		Pic:       "/tmp/cover.jpg",
		OutPath:   "/tmp/out.mp4",
		Title:     "Title",
	}

	args, err := m.buildMP4BoxArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, ":cover=/tmp/cover.jpg") {
		t.Errorf("expected cover in itags, got: %s", argsStr)
	}
}

func TestMP4BoxMuxer_BuildArgs_WithEpisodeID(t *testing.T) {
	m := &MP4BoxMuxer{logger: discardLogger()}
	cfg := MuxConfig{
		BVid:      "BV1xx411c7mD",
		VideoPath: "/tmp/video.m4s",
		AudioPath: "/tmp/audio.m4s",
		OutPath:   "/tmp/out.mp4",
		Title:     "Series Title",
		EpisodeID: "EP01",
	}

	args, err := m.buildMP4BoxArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, ":album=Series Title:title=EP01") {
		t.Errorf("expected album and episode title, got: %s", argsStr)
	}
}

func TestMP4BoxMuxer_BuildArgs_WithSubtitles(t *testing.T) {
	dir := t.TempDir()
	sub1 := filepath.Join(dir, "sub1.srt")
	sub2 := filepath.Join(dir, "sub2.srt")
	os.WriteFile(sub1, []byte("1\n00:00:00,000 --> 00:00:01,000\nHello\n"), 0o644)
	os.WriteFile(sub2, []byte(""), 0o644)

	m := &MP4BoxMuxer{logger: discardLogger()}
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

	args, err := m.buildMP4BoxArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "-add "+sub1+"#trackID=1:name=:hdlr=sbtl:lang=chi") {
		t.Errorf("expected subtitle add, got: %s", argsStr)
	}
	if strings.Contains(argsStr, "-add "+sub2) {
		t.Errorf("expected empty subtitle to be skipped, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-udta") {
		t.Errorf("expected udta for subtitle name, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "type=name:str=Chinese (Simplified)") {
		t.Errorf("expected subtitle name in udta, got: %s", argsStr)
	}
}

func TestMP4BoxMuxer_BuildArgs_DebugLog(t *testing.T) {
	m := &MP4BoxMuxer{logger: debugLogger()}
	cfg := MuxConfig{
		BVid:      "BV1xx411c7mD",
		VideoPath: "/tmp/video.m4s",
		AudioPath: "/tmp/audio.m4s",
		OutPath:   "/tmp/out.mp4",
		Title:     "Title",
	}

	args, err := m.buildMP4BoxArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "-v") {
		t.Errorf("expected -v flag for debug, got: %s", argsStr)
	}
}

func TestFormatTime(t *testing.T) {
	cases := []struct {
		seconds int
		want    string
	}{
		{0, "00:00:00"},
		{59, "00:00:59"},
		{60, "00:01:00"},
		{3661, "01:01:01"},
		{86399, "23:59:59"},
	}

	for _, c := range cases {
		got := formatTime(c.seconds)
		if got != c.want {
			t.Errorf("formatTime(%d) = %q, want %q", c.seconds, got, c.want)
		}
	}
}

func TestGetFFmpegMetaString(t *testing.T) {
	points := []entity.ViewPoint{
		{Title: "Intro", Start: 0, End: 10},
		{Title: "Main", Start: 10, End: 100},
	}
	meta := getFFmpegMetaString(points)
	if !strings.Contains(meta, ";FFMETADATA") {
		t.Errorf("expected header, got: %s", meta)
	}
	if !strings.Contains(meta, "TIMEBASE=1/1000") {
		t.Errorf("expected timebase, got: %s", meta)
	}
	if !strings.Contains(meta, "START=0") {
		t.Errorf("expected start 0, got: %s", meta)
	}
	if !strings.Contains(meta, "END=10000") {
		t.Errorf("expected end 10000, got: %s", meta)
	}
	if !strings.Contains(meta, "title=Intro") {
		t.Errorf("expected title Intro, got: %s", meta)
	}
}
