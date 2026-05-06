package muxer

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/nilaonai/bbdown-go/internal/core/entity"
	"github.com/nilaonai/bbdown-go/internal/core/util"
)

// execCommandContext is a package-level variable to allow mocking in tests.
var execCommandContext = exec.CommandContext

// FFmpegMuxer implements Muxer using ffmpeg.
type FFmpegMuxer struct {
	logger *slog.Logger
}

// NewFFmpegMuxer creates a new ffmpeg-based muxer.
func NewFFmpegMuxer(logger *slog.Logger) Muxer {
	return &FFmpegMuxer{logger: logger}
}

// Mux multiplexes video, audio, subtitles and metadata into an MP4 file using ffmpeg.
func (m *FFmpegMuxer) Mux(ctx context.Context, cfg MuxConfig) error {
	ffmpegPath, err := FindExecutable("ffmpeg")
	if err != nil {
		return fmt.Errorf("find ffmpeg: %w", err)
	}

	args, err := m.buildFFmpegArgs(cfg)
	if err != nil {
		return fmt.Errorf("build ffmpeg args: %w", err)
	}

	dir := filepath.Dir(cfg.OutPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}
	}

	cmd := execCommandContext(ctx, ffmpegPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg: %w", err)
	}
	return nil
}

// MergeFLV merges multiple FLV files into a single output file.
// If only one file is provided, it is renamed directly.
// Otherwise, each FLV is converted to TS via ffmpeg and then concatenated.
func (m *FFmpegMuxer) MergeFLV(ctx context.Context, files []string, outPath string) error {
	return mergeFLVWithFFmpeg(ctx, files, outPath)
}

func mergeFLVWithFFmpeg(ctx context.Context, files []string, outPath string) error {
	if len(files) == 0 {
		return fmt.Errorf("no files to merge")
	}

	if len(files) == 1 {
		if err := os.Rename(files[0], outPath); err != nil {
			return fmt.Errorf("rename single file: %w", err)
		}
		return nil
	}

	ffmpegPath, err := FindExecutable("ffmpeg")
	if err != nil {
		return fmt.Errorf("find ffmpeg: %w", err)
	}

	var tsFiles []string
	for _, file := range files {
		tmpFile := filepath.Join(filepath.Dir(file), filepath.Base(file[:len(file)-len(filepath.Ext(file))])+".ts")
		args := []string{
			"-loglevel", "warning", "-y", "-i", file,
			"-map", "0", "-c", "copy", "-f", "mpegts",
			"-bsf:v", "h264_mp4toannexb", tmpFile,
		}
		cmd := execCommandContext(ctx, ffmpegPath, args...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ffmpeg flv->ts: %w", err)
		}
		if err := os.Remove(file); err != nil {
			return fmt.Errorf("remove flv: %w", err)
		}
		tsFiles = append(tsFiles, tmpFile)
	}

	if err := combineFiles(tsFiles, outPath); err != nil {
		return fmt.Errorf("combine ts files: %w", err)
	}

	for _, f := range tsFiles {
		_ = os.Remove(f)
	}

	return nil
}

func combineFiles(files []string, outPath string) error {
	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer out.Close()

	for _, f := range files {
		in, err := os.Open(f)
		if err != nil {
			return fmt.Errorf("open input file: %w", err)
		}
		_, err = io.Copy(out, in)
		in.Close()
		if err != nil {
			return fmt.Errorf("copy file: %w", err)
		}
	}
	return nil
}

func (m *FFmpegMuxer) buildFFmpegArgs(cfg MuxConfig) ([]string, error) {
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
		args = append(args, "-loglevel", "verbose")
	} else {
		args = append(args, "-loglevel", "warning")
	}
	args = append(args, "-y")

	var realInputs []string
	if videoPath != "" {
		realInputs = append(realInputs, videoPath)
	}
	if audioPath != "" {
		realInputs = append(realInputs, audioPath)
	}
	for _, am := range cfg.AudioMaterial {
		realInputs = append(realInputs, am.Path)
	}
	if cfg.Pic != "" {
		realInputs = append(realInputs, cfg.Pic)
	}

	var validSubs []entity.Subtitle
	for _, sub := range cfg.Subtitles {
		if sub.Path == "" {
			continue
		}
		data, err := os.ReadFile(sub.Path)
		if err != nil || len(data) == 0 {
			continue
		}
		validSubs = append(validSubs, sub)
		realInputs = append(realInputs, sub.Path)
	}

	inputCount := len(realInputs)

	var chapterFile string
	if len(cfg.Points) > 0 {
		basePath := videoPath
		if basePath == "" {
			basePath = audioPath
		}
		dir := filepath.Dir(basePath)
		chapterFile = filepath.Join(dir, "chapters")
		if err := os.WriteFile(chapterFile, []byte(getFFmpegMetaString(cfg.Points)), 0o644); err != nil {
			return nil, fmt.Errorf("write chapter file: %w", err)
		}
	}

	for _, input := range realInputs {
		args = append(args, "-i", input)
	}

	if chapterFile != "" {
		args = append(args, "-i", chapterFile)
		args = append(args, "-map_chapters", strconv.Itoa(inputCount))
	}

	for i := 0; i < inputCount; i++ {
		args = append(args, "-map", strconv.Itoa(i))
	}

	if len(cfg.AudioMaterial) > 0 {
		args = append(args, "-metadata:s:a:0", "title=原音频")
		for i, am := range cfg.AudioMaterial {
			audioIndex := i + 1
			if strings.TrimSpace(am.Title) != "" {
				args = append(args, fmt.Sprintf("-metadata:s:a:%d", audioIndex), fmt.Sprintf("title=%s", am.Title))
			}
			if strings.TrimSpace(am.PersonName) != "" {
				args = append(args, fmt.Sprintf("-metadata:s:a:%d", audioIndex), fmt.Sprintf("artist=%s", am.PersonName))
			}
		}
	}

	if cfg.Pic != "" {
		videoStreamIndex := "1"
		if cfg.AudioOnly {
			videoStreamIndex = "0"
		}
		args = append(args, fmt.Sprintf("-disposition:v:%s", videoStreamIndex), "attached_pic")
	}

	for i, sub := range validSubs {
		code, name := util.GetSubtitleCode(sub.Lan)
		args = append(args, fmt.Sprintf("-metadata:s:s:%d", i), fmt.Sprintf("title=%s", name))
		args = append(args, fmt.Sprintf("-metadata:s:s:%d", i), fmt.Sprintf("language=%s", code))
	}

	if !cfg.SimplyMux {
		metaTitle := title
		if episodeID != "" {
			metaTitle = episodeID
		}
		args = append(args, "-metadata", fmt.Sprintf("title=%s", metaTitle))
		args = append(args, "-metadata", fmt.Sprintf("comment=%s", url))
		if cfg.Lang != "" {
			args = append(args, "-metadata:s:a:0", fmt.Sprintf("language=%s", cfg.Lang))
		}
		if strings.TrimSpace(desc) != "" {
			args = append(args, "-metadata", fmt.Sprintf("description=%s", desc))
		}
		if cfg.Author != "" {
			args = append(args, "-metadata", fmt.Sprintf("artist=%s", cfg.Author))
		}
		if episodeID != "" {
			args = append(args, "-metadata", fmt.Sprintf("album=%s", title))
		}
		if cfg.PubTime != 0 {
			tm := time.Unix(cfg.PubTime, 0).UTC().Format("2006-01-02T15:04:05.000000Z")
			args = append(args, "-metadata", fmt.Sprintf("creation_time=%s", tm))
		}
	}

	args = append(args, "-c:v", "copy", "-c:a", "copy")
	if cfg.AudioOnly && audioPath == "" {
		args = append(args, "-vn")
	}
	if len(validSubs) > 0 {
		args = append(args, "-c:s", "mov_text")
	}
	if runtime.GOOS == "darwin" && cfg.IsHevc {
		args = append(args, "-tag:v:0", "hvc1")
	}
	args = append(args, "-movflags", "faststart", "-strict", "unofficial", "-strict", "-2", "-f", "mp4", "--", cfg.OutPath)

	return args, nil
}

func getFFmpegMetaString(points []entity.ViewPoint) string {
	var sb strings.Builder
	sb.WriteString(";FFMETADATA\n")
	const timeBase = 1000
	for _, p := range points {
		sb.WriteString("[CHAPTER]\n")
		sb.WriteString(fmt.Sprintf("TIMEBASE=1/%d\n", timeBase))
		sb.WriteString(fmt.Sprintf("START=%d\n", p.Start*timeBase))
		sb.WriteString(fmt.Sprintf("END=%d\n", p.End*timeBase))
		sb.WriteString(fmt.Sprintf("title=%s\n", p.Title))
		sb.WriteString("\n")
	}
	return sb.String()
}
