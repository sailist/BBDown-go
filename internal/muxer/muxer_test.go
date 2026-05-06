package muxer

import (
	"context"
	"testing"

	"github.com/nilaonai/bbdown-go/internal/core/entity"
)

// mockMuxer is a test implementation of the Muxer interface.
type mockMuxer struct {
	muxCalled      bool
	lastMuxConfig  MuxConfig
	mergeCalled    bool
	lastFiles      []string
	lastOutPath    string
}

func (m *mockMuxer) Mux(ctx context.Context, cfg MuxConfig) error {
	m.muxCalled = true
	m.lastMuxConfig = cfg
	return nil
}

func (m *mockMuxer) MergeFLV(ctx context.Context, files []string, outPath string) error {
	m.mergeCalled = true
	m.lastFiles = files
	m.lastOutPath = outPath
	return nil
}

func TestMuxer_InterfaceContract(t *testing.T) {
	var _ Muxer = (*mockMuxer)(nil)
}

func TestMuxConfig_ConstructionAndFieldAccess(t *testing.T) {
	cfg := MuxConfig{
		BVid:      "BV1xx411c7mD",
		VideoPath: "/tmp/video.m4s",
		AudioPath: "/tmp/audio.m4s",
		AudioMaterial: []entity.AudioMaterial{
			{Title: "track1", PersonName: "Alice", Path: "/tmp/a1.m4s"},
		},
		OutPath:   "/tmp/out.mp4",
		Desc:      "desc",
		Title:     "title",
		Author:    "author",
		EpisodeID: "ep1",
		Pic:       "/tmp/cover.jpg",
		Lang:      "zh-CN",
		Subtitles: []entity.Subtitle{
			{Lan: "zh-CN", Url: "http://example.com/s1", Path: "/tmp/s1.json"},
		},
		AudioOnly: true,
		VideoOnly: false,
		Points: []entity.ViewPoint{
			{Title: "intro", Start: 0, End: 10},
		},
		PubTime:   1234567890,
		SimplyMux: true,
		IsHevc:    false,
	}

	if cfg.BVid != "BV1xx411c7mD" {
		t.Errorf("BVid = %q, want %q", cfg.BVid, "BV1xx411c7mD")
	}
	if cfg.VideoPath != "/tmp/video.m4s" {
		t.Errorf("VideoPath = %q, want %q", cfg.VideoPath, "/tmp/video.m4s")
	}
	if cfg.AudioPath != "/tmp/audio.m4s" {
		t.Errorf("AudioPath = %q, want %q", cfg.AudioPath, "/tmp/audio.m4s")
	}
	if len(cfg.AudioMaterial) != 1 || cfg.AudioMaterial[0].Title != "track1" {
		t.Errorf("AudioMaterial mismatch")
	}
	if cfg.OutPath != "/tmp/out.mp4" {
		t.Errorf("OutPath = %q, want %q", cfg.OutPath, "/tmp/out.mp4")
	}
	if cfg.Desc != "desc" {
		t.Errorf("Desc = %q, want %q", cfg.Desc, "desc")
	}
	if cfg.Title != "title" {
		t.Errorf("Title = %q, want %q", cfg.Title, "title")
	}
	if cfg.Author != "author" {
		t.Errorf("Author = %q, want %q", cfg.Author, "author")
	}
	if cfg.EpisodeID != "ep1" {
		t.Errorf("EpisodeID = %q, want %q", cfg.EpisodeID, "ep1")
	}
	if cfg.Pic != "/tmp/cover.jpg" {
		t.Errorf("Pic = %q, want %q", cfg.Pic, "/tmp/cover.jpg")
	}
	if cfg.Lang != "zh-CN" {
		t.Errorf("Lang = %q, want %q", cfg.Lang, "zh-CN")
	}
	if len(cfg.Subtitles) != 1 || cfg.Subtitles[0].Lan != "zh-CN" {
		t.Errorf("Subtitles mismatch")
	}
	if !cfg.AudioOnly {
		t.Errorf("AudioOnly = %v, want true", cfg.AudioOnly)
	}
	if cfg.VideoOnly {
		t.Errorf("VideoOnly = %v, want false", cfg.VideoOnly)
	}
	if len(cfg.Points) != 1 || cfg.Points[0].Title != "intro" {
		t.Errorf("Points mismatch")
	}
	if cfg.PubTime != 1234567890 {
		t.Errorf("PubTime = %d, want %d", cfg.PubTime, 1234567890)
	}
	if !cfg.SimplyMux {
		t.Errorf("SimplyMux = %v, want true", cfg.SimplyMux)
	}
	if cfg.IsHevc {
		t.Errorf("IsHevc = %v, want false", cfg.IsHevc)
	}
}

func TestMockMuxer_Mux(t *testing.T) {
	m := &mockMuxer{}
	ctx := context.Background()
	cfg := MuxConfig{BVid: "BV1xx411c7mD"}

	if err := m.Mux(ctx, cfg); err != nil {
		t.Fatalf("Mux error: %v", err)
	}
	if !m.muxCalled {
		t.Error("expected muxCalled to be true")
	}
	if m.lastMuxConfig.BVid != "BV1xx411c7mD" {
		t.Errorf("lastMuxConfig.BVid = %q, want %q", m.lastMuxConfig.BVid, "BV1xx411c7mD")
	}
}

func TestMockMuxer_MergeFLV(t *testing.T) {
	m := &mockMuxer{}
	ctx := context.Background()
	files := []string{"/tmp/1.flv", "/tmp/2.flv"}
	out := "/tmp/merged.flv"

	if err := m.MergeFLV(ctx, files, out); err != nil {
		t.Fatalf("MergeFLV error: %v", err)
	}
	if !m.mergeCalled {
		t.Error("expected mergeCalled to be true")
	}
	if len(m.lastFiles) != 2 || m.lastFiles[0] != "/tmp/1.flv" {
		t.Errorf("lastFiles mismatch: %v", m.lastFiles)
	}
	if m.lastOutPath != "/tmp/merged.flv" {
		t.Errorf("lastOutPath = %q, want %q", m.lastOutPath, "/tmp/merged.flv")
	}
}
