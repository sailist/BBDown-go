package muxer

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/nilaonai/bbdown-go/internal/core/entity"
	"github.com/nilaonai/bbdown-go/internal/core/util"
)

// MP4BoxMuxer implements Muxer using MP4Box.
type MP4BoxMuxer struct {
	logger *slog.Logger
}

// NewMP4BoxMuxer creates a new MP4Box-based muxer.
func NewMP4BoxMuxer(logger *slog.Logger) Muxer {
	return &MP4BoxMuxer{logger: logger}
}

// Mux multiplexes video, audio, subtitles and metadata into an MP4 file using MP4Box.
func (m *MP4BoxMuxer) Mux(ctx context.Context, cfg MuxConfig) error {
	mp4boxPath, err := FindExecutable("mp4box")
	if err != nil {
		return fmt.Errorf("find mp4box: %w", err)
	}

	args, err := m.buildMP4BoxArgs(cfg)
	if err != nil {
		return err
	}

	cmd := execCommandContext(ctx, mp4boxPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mp4box: %w", err)
	}
	return nil
}

// MergeFLV merges multiple FLV files into a single output file using ffmpeg.
func (m *MP4BoxMuxer) MergeFLV(ctx context.Context, files []string, outPath string) error {
	return mergeFLVWithFFmpeg(ctx, files, outPath)
}

func (m *MP4BoxMuxer) buildMP4BoxArgs(cfg MuxConfig) ([]string, error) {
	videoPath := cfg.VideoPath
	audioPath := cfg.AudioPath

	if cfg.AudioOnly && audioPath != "" {
		videoPath = ""
	}
	if cfg.VideoOnly {
		audioPath = ""
	}

	desc := cfg.Desc
	title := cfg.Title
	episodeID := cfg.EpisodeID
	url := fmt.Sprintf("https://www.bilibili.com/video/%s/", cfg.BVid)

	var args []string

	if m.logger.Enabled(context.Background(), slog.LevelDebug) {
		args = append(args, "-v")
	}
	args = append(args, "-inter", "500", "-noprog")

	nowID := 0

	if videoPath != "" {
		trackID := "1"
		if cfg.AudioOnly && audioPath == "" {
			trackID = "2"
		}
		args = append(args, "-add", fmt.Sprintf("%s#trackID=%s:name=", videoPath, trackID))
		nowID++
	}

	if audioPath != "" {
		lang := cfg.Lang
		if lang == "" {
			lang = "und"
		}
		args = append(args, "-add", fmt.Sprintf("%s:lang=%s", audioPath, lang))
		nowID++
	}

	if len(cfg.Points) > 0 {
		basePath := videoPath
		if basePath == "" {
			basePath = audioPath
		}
		dir := filepath.Dir(basePath)
		chapterFile := filepath.Join(dir, "chapters")
		if err := os.WriteFile(chapterFile, []byte(getMp4boxMetaString(cfg.Points)), 0o644); err != nil {
			return nil, fmt.Errorf("write chapter file: %w", err)
		}
		args = append(args, "-chap", chapterFile)
	}

	var metaArg strings.Builder
	if cfg.Pic != "" {
		metaArg.WriteString(fmt.Sprintf(":cover=%s", cfg.Pic))
	}
	if episodeID != "" {
		metaArg.WriteString(fmt.Sprintf(":album=%s:title=%s", title, episodeID))
	} else {
		metaArg.WriteString(fmt.Sprintf(":title=%s", title))
	}
	metaArg.WriteString(fmt.Sprintf(":sdesc=%s", desc))
	metaArg.WriteString(fmt.Sprintf(":comment=%s", url))
	metaArg.WriteString(fmt.Sprintf(":artist=%s", cfg.Author))

	if metaArg.Len() > 0 {
		args = append(args, "-itags", "tool="+metaArg.String())
	}

	for _, sub := range cfg.Subtitles {
		if sub.Path == "" {
			continue
		}
		data, err := os.ReadFile(sub.Path)
		if err != nil || len(data) == 0 {
			continue
		}
		nowID++
		code, name := util.GetSubtitleCode(sub.Lan)
		args = append(args, "-add", fmt.Sprintf("%s#trackID=1:name=:hdlr=sbtl:lang=%s", sub.Path, code))
		args = append(args, "-udta", fmt.Sprintf("%d:type=name:str=%s", nowID, name))
	}

	args = append(args, "-new", "--", cfg.OutPath)

	return args, nil
}

func getMp4boxMetaString(points []entity.ViewPoint) string {
	var sb strings.Builder
	for _, p := range points {
		sb.WriteString(fmt.Sprintf("%s %s\n", formatTime(p.Start), p.Title))
	}
	return sb.String()
}

func formatTime(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}
