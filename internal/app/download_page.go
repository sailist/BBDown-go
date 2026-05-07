package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nilaonai/bbdown-go/internal/cli"
	"github.com/nilaonai/bbdown-go/internal/config"
	"github.com/nilaonai/bbdown-go/internal/core/entity"
	"github.com/nilaonai/bbdown-go/internal/core/parser"
	"github.com/nilaonai/bbdown-go/internal/core/util"
	"github.com/nilaonai/bbdown-go/internal/danmaku"
	"github.com/nilaonai/bbdown-go/internal/download"
	"github.com/nilaonai/bbdown-go/internal/muxer"
	"github.com/nilaonai/bbdown-go/pkg/httpclient"
)

// DownloadDeps holds dependencies for DownloadPage, enabling test mocking.
type DownloadDeps struct {
	HTTPClient             httpclient.Client
	Logger                 *slog.Logger
	Downloader             download.Downloader
	Muxer                  muxer.Muxer
	Config                 *config.Config
	ExtractTracks          func(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts parser.ExtractOptions) (*entity.ParsedResult, error)
	SelectTrackInteractive func(prompt string, max int) (int, error)
	DownloadPage           func(ctx context.Context, p *entity.Page, opt *cli.Option, vInfo *entity.VInfo, selectedPages []entity.Page, workCfg *WorkConfig, deps DownloadDeps) error
}

// selectTrackInteractive reads user input from stdin to select a track index.
// It is a package-level variable so tests can override it.
var selectTrackInteractive = func(prompt string, max int) (int, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	text, err := reader.ReadString('\n')
	if err != nil {
		return 0, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, nil
	}
	idx, err := strconv.Atoi(text)
	if err != nil || idx < 0 || idx >= max {
		return 0, nil
	}
	return idx, nil
}

// DownloadPage handles downloading a single page (part/episode).
func DownloadPage(
	ctx context.Context,
	p *entity.Page,
	opt *cli.Option,
	vInfo *entity.VInfo,
	selectedPages []entity.Page,
	workCfg *WorkConfig,
	deps DownloadDeps,
) error {
	// 1. Get chapter/viewpoint info
	points, err := FetchPoints(ctx, deps.HTTPClient, p.Cid, p.Aid)
	if err != nil {
		deps.Logger.Warn("fetch points failed", "error", err)
	} else {
		p.Points = points
	}

	title := vInfo.Title
	if strings.HasSuffix(title, ".") {
		title += "_fix"
	}
	if strings.HasPrefix(title, ".") {
		title = "_" + title
	}

	coverPath := fmt.Sprintf("%s/%s.jpg", p.Aid, p.Aid)
	pagesCount := len(selectedPages)

	apiType := "WEB"
	switch {
	case opt.UseTvApi:
		apiType = "TV"
	case opt.UseAppApi:
		apiType = "APP"
	case opt.UseIntlApi:
		apiType = "INTL"
	}

	// Early return: OnlyShowInfo
	if opt.OnlyShowInfo {
		return nil
	}

	// Ensure directory exists
	if err := os.MkdirAll(p.Aid, 0o755); err != nil {
		return fmt.Errorf("create aid directory: %w", err)
	}

	// 2. Download cover
	if !opt.SkipCover && !opt.SubOnly && !opt.DanmakuOnly && !opt.CoverOnly {
		picURL := vInfo.Pic
		if picURL == "" {
			picURL = p.Cover
		}
		if picURL != "" {
			if err := downloadCover(ctx, deps.HTTPClient, picURL, coverPath); err != nil {
				deps.Logger.Warn("download cover failed", "error", err)
			}
		}
	}

	// 3. Get and download subtitles
	if !opt.SkipSubtitle && !opt.DanmakuOnly && !opt.CoverOnly {
		subtitles, err := util.GetSubtitles(ctx, deps.HTTPClient, workCfg.Config, p.Aid, p.Cid, p.Epid, p.Index, opt.UseIntlApi)
		if err != nil {
			deps.Logger.Warn("get subtitles failed", "error", err)
		} else {
			if opt.SkipAi {
				subtitles = filterAiSubtitles(subtitles)
			}
			for _, sub := range subtitles {
				if err := util.SaveSubtitle(ctx, deps.HTTPClient, sub); err != nil {
					deps.Logger.Warn("save subtitle failed", "lan", sub.Lan, "error", err)
					continue
				}
				if opt.SubOnly {
					savePath := FormatSavePath(workCfg.SavePathFormat, title, nil, nil, *p, pagesCount, apiType, vInfo.PubTime)
					ext := filepath.Ext(sub.Path)
					if ext == "" {
						ext = ".srt"
					}
					outSubPath := strings.TrimSuffix(savePath, filepath.Ext(savePath)) + "." + sub.Lan + ext
					if err := os.MkdirAll(filepath.Dir(outSubPath), 0o755); err != nil {
						deps.Logger.Warn("create subtitle output dir failed", "error", err)
						continue
					}
					if err := os.Rename(sub.Path, outSubPath); err != nil {
						deps.Logger.Warn("move subtitle to final path failed", "error", err)
					}
				}
			}
		}
	}

	// Early return: SubOnly
	if opt.SubOnly {
		if err := cleanEmptyDir(p.Aid); err != nil {
			deps.Logger.Debug("clean empty dir failed", "error", err)
		}
		return nil
	}

	// Early return: CoverOnly
	if opt.CoverOnly {
		picURL := vInfo.Pic
		if picURL == "" {
			picURL = p.Cover
		}
		if picURL != "" {
			savePath := FormatSavePath(workCfg.SavePathFormat, title, nil, nil, *p, pagesCount, apiType, vInfo.PubTime)
			ext := filepath.Ext(picURL)
			if ext == "" {
				ext = ".jpg"
			}
			finalCoverPath := strings.TrimSuffix(savePath, filepath.Ext(savePath)) + ext
			if err := os.MkdirAll(filepath.Dir(finalCoverPath), 0o755); err != nil {
				return fmt.Errorf("create cover output dir: %w", err)
			}
			if err := downloadCover(ctx, deps.HTTPClient, picURL, finalCoverPath); err != nil {
				return fmt.Errorf("download cover to final path: %w", err)
			}
		}
		if err := cleanEmptyDir(p.Aid); err != nil {
			deps.Logger.Debug("clean empty dir failed", "error", err)
		}
		return nil
	}

	// Parts 2 & 3: stream selection, download, and mux with retry logic
	var lastErr error
	for retry := 0; retry < 3; retry++ {
		if retry > 0 {
			deps.Logger.Warn("retrying download page", "retry", retry)
		}
		lastErr = downloadPageBody(ctx, p, opt, vInfo, selectedPages, workCfg, deps, coverPath, title, pagesCount, apiType)
		if lastErr == nil {
			return nil
		}
	}
	return fmt.Errorf("download page failed after 3 retries: %w", lastErr)
}

func downloadPageBody(
	ctx context.Context,
	p *entity.Page,
	opt *cli.Option,
	vInfo *entity.VInfo,
	selectedPages []entity.Page,
	workCfg *WorkConfig,
	deps DownloadDeps,
	coverPath, title string,
	pagesCount int,
	apiType string,
) error {
	extractTracks := deps.ExtractTracks
	if extractTracks == nil {
		extractTracks = parser.ExtractTracks
	}

	selectInteractive := deps.SelectTrackInteractive
	if selectInteractive == nil {
		selectInteractive = selectTrackInteractive
	}

	opts := parser.ExtractOptions{
		TvApi:    opt.UseTvApi,
		IntlApi:  opt.UseIntlApi,
		AppApi:   opt.UseAppApi,
		Encoding: workCfg.FirstEncoding,
		Qn:       opt.DfnPriority,
	}

	parsedResult, err := extractTracks(ctx, deps.HTTPClient, workCfg.Config, deps.Logger, workCfg.AidOri, p.Aid, p.Cid, p.Epid, opts)
	if err != nil {
		return fmt.Errorf("extract tracks: %w", err)
	}

	// Merge extra points
	if len(p.Points) == 0 {
		p.Points = parsedResult.ExtraPoints
	}

	// Write debug JSON
	if opt.Debug {
		debugPath := fmt.Sprintf("%s/%s.debug.json", p.Aid, p.Aid)
		_ = os.WriteFile(debugPath, []byte(parsedResult.WebJsonString), 0o644)
	}

	// Handle DASH case (VideoTracks or AudioTracks exist, no Clips)
	if (len(parsedResult.VideoTracks) > 0 || len(parsedResult.AudioTracks) > 0) && len(parsedResult.Clips) == 0 {
		// Filter tracks based on options
		if opt.AudioOnly {
			parsedResult.VideoTracks = nil
		}
		if opt.VideoOnly {
			parsedResult.AudioTracks = nil
			parsedResult.BackgroundAudioTracks = nil
			parsedResult.RoleAudioList = nil
		}

		// Sort tracks
		parsedResult.VideoTracks = SortVideoTracks(parsedResult.VideoTracks, workCfg.DfnPriority, workCfg.EncodingPriority, opt.VideoAscending)
		parsedResult.AudioTracks = SortAudioTracks(parsedResult.AudioTracks, workCfg.EncodingPriority, opt.AudioAscending)
		parsedResult.BackgroundAudioTracks = SortAudioTracks(parsedResult.BackgroundAudioTracks, workCfg.EncodingPriority, opt.AudioAscending)

		// Print tracks info
		if !opt.HideStreams {
			printStreamsImpl(deps.Logger, parsedResult.VideoTracks, parsedResult.AudioTracks, parsedResult.BackgroundAudioTracks, parsedResult.RoleAudioList)
		}

		// Early return if OnlyShowInfo
		if opt.OnlyShowInfo {
			return nil
		}

		selected := false

		// Interactive selection
		var selectedVideo *entity.Video
		var selectedAudio *entity.Audio
		var selectedBackgroundAudio *entity.Audio

		if opt.Interactive && !selected {
			if len(parsedResult.VideoTracks) > 0 {
				idx, _ := selectInteractive("Select video track: ", len(parsedResult.VideoTracks))
				if idx >= 0 && idx < len(parsedResult.VideoTracks) {
					selectedVideo = &parsedResult.VideoTracks[idx]
				}
			}
			if len(parsedResult.AudioTracks) > 0 {
				idx, _ := selectInteractive("Select audio track: ", len(parsedResult.AudioTracks))
				if idx >= 0 && idx < len(parsedResult.AudioTracks) {
					selectedAudio = &parsedResult.AudioTracks[idx]
				}
			}
			selected = true
		} else {
			if len(parsedResult.VideoTracks) > 0 {
				selectedVideo = &parsedResult.VideoTracks[0]
			}
			if len(parsedResult.AudioTracks) > 0 {
				selectedAudio = &parsedResult.AudioTracks[0]
			}
		}
		if len(parsedResult.BackgroundAudioTracks) > 0 {
			selectedBackgroundAudio = &parsedResult.BackgroundAudioTracks[0]
		}

		// Download danmaku
		if workCfg.DownloadDanmaku {
			danmakuXMLPath := fmt.Sprintf("%s/%s.xml", p.Aid, p.Aid)
			danmakuURL := fmt.Sprintf("https://comment.bilibili.com/%s.xml", p.Cid)
			if err := downloadFile(ctx, deps.HTTPClient, danmakuURL, danmakuXMLPath); err != nil {
				deps.Logger.Warn("download danmaku failed", "error", err)
			} else {
				items, err := danmaku.ParseXML(danmakuXMLPath)
				if err != nil {
					deps.Logger.Warn("parse danmaku failed", "error", err)
				} else {
					hasASS := false
					hasXML := false
					for _, f := range workCfg.DownloadDanmakuFormats {
						if f == "ass" {
							hasASS = true
						}
						if f == "xml" {
							hasXML = true
						}
					}
					if hasASS {
						assPath := strings.TrimSuffix(danmakuXMLPath, filepath.Ext(danmakuXMLPath)) + ".ass"
						if err := danmaku.SaveAsAss(items, assPath); err != nil {
							deps.Logger.Warn("save danmaku ass failed", "error", err)
						}
					}
					if !hasXML {
						_ = os.Remove(danmakuXMLPath)
					}
				}
			}
		}

		// Early return if DanmakuOnly
		if opt.DanmakuOnly {
			return nil
		}

		// Compute save path and check existence
		savePath := FormatSavePath(workCfg.SavePathFormat, title, selectedVideo, selectedAudio, *p, pagesCount, apiType, vInfo.PubTime)
		if _, err := os.Stat(savePath); err == nil {
			deps.Logger.Info("file already exists, skipping download", "path", savePath)
			return nil
		}

		// Download tracks
		var videoPath, audioPath string
		if selectedVideo != nil {
			videoPath = fmt.Sprintf("%s/%s.m4v", p.Aid, p.Aid)
			if err := downloadTrack(ctx, deps, selectedVideo.BaseUrl, videoPath, opt); err != nil {
				return fmt.Errorf("download video: %w", err)
			}
		}
		if selectedAudio != nil {
			audioPath = fmt.Sprintf("%s/%s.m4a", p.Aid, p.Aid)
			if err := downloadTrack(ctx, deps, selectedAudio.BaseUrl, audioPath, opt); err != nil {
				return fmt.Errorf("download audio: %w", err)
			}
		}
		if selectedBackgroundAudio != nil {
			bgPath := fmt.Sprintf("%s/%s.bg.m4a", p.Aid, p.Aid)
			if err := downloadTrack(ctx, deps, selectedBackgroundAudio.BaseUrl, bgPath, opt); err != nil {
				return fmt.Errorf("download background audio: %w", err)
			}
		}

		// Download role audios
		var audioMaterials []entity.AudioMaterial
		for _, role := range parsedResult.RoleAudioList {
			if len(role.Audio) > 0 {
				raPath := fmt.Sprintf("%s/%s.role.%s.m4a", p.Aid, p.Aid, role.PersonName)
				if err := downloadTrack(ctx, deps, role.Audio[0].BaseUrl, raPath, opt); err != nil {
					deps.Logger.Warn("download role audio failed", "role", role.PersonName, "error", err)
					continue
				}
				audioMaterials = append(audioMaterials, entity.AudioMaterial{
					Title:      role.Title,
					PersonName: role.PersonName,
					Path:       raPath,
				})
			}
		}

		// Early return if SkipMux
		if opt.SkipMux {
			return nil
		}

		// Mux
		var subs []entity.Subtitle
		if !opt.SkipSubtitle {
			subs, _ = util.GetSubtitles(ctx, deps.HTTPClient, workCfg.Config, p.Aid, p.Cid, p.Epid, p.Index, opt.UseIntlApi)
			if opt.SkipAi {
				subs = filterAiSubtitles(subs)
			}
		}

		muxCfg := muxer.MuxConfig{
			BVid:          p.BVid(),
			VideoPath:     videoPath,
			AudioPath:     audioPath,
			AudioMaterial: audioMaterials,
			OutPath:       savePath,
			Desc:          p.Desc,
			Title:         title,
			Author:        p.OwnerName,
			EpisodeID:     p.Epid,
			Pic:           coverPath,
			Lang:          workCfg.Lang,
			Subtitles:     subs,
			AudioOnly:     opt.AudioOnly,
			VideoOnly:     opt.VideoOnly,
			SimplyMux:     opt.SimplyMux,
			IsHevc:        selectedVideo != nil && strings.Contains(selectedVideo.Codecs, "hevc"),
			Points:        p.Points,
			PubTime:       vInfo.PubTime,
		}
		if err := deps.Muxer.Mux(ctx, muxCfg); err != nil {
			return fmt.Errorf("mux failed: %w", err)
		}

		// Clean up temp files
		if videoPath != "" {
			_ = os.Remove(videoPath)
		}
		if audioPath != "" {
			_ = os.Remove(audioPath)
		}
		for _, sub := range subs {
			if sub.Path != "" {
				_ = os.Remove(sub.Path)
			}
		}
		for _, am := range audioMaterials {
			if am.Path != "" {
				_ = os.Remove(am.Path)
			}
		}
		if coverPath != "" {
			_ = os.Remove(coverPath)
		}

		return nil
	}

	// Handle FLV case (Clips and Dfns exist)
	if len(parsedResult.Clips) > 0 && len(parsedResult.Dfns) > 0 {
		parsedResult.VideoTracks = SortVideoTracks(parsedResult.VideoTracks, workCfg.DfnPriority, workCfg.EncodingPriority, opt.VideoAscending)

		// Interactive quality selection if needed (re-parse with selected dfn)
		selected := false
		if opt.Interactive && !selected && len(parsedResult.VideoTracks) > 0 {
			idx, _ := selectInteractive("Select video quality: ", len(parsedResult.VideoTracks))
			_ = idx
			// In C#, this would re-parse with selected dfn
			selected = true
		}

		if !opt.HideStreams {
			printStreamsImpl(deps.Logger, parsedResult.VideoTracks, nil, nil, nil)
		}

		if opt.OnlyShowInfo {
			return nil
		}

		// Compute save path and check existence
		savePath := FormatSavePath(workCfg.SavePathFormat, title, nil, nil, *p, pagesCount, apiType, vInfo.PubTime)
		if _, err := os.Stat(savePath); err == nil {
			deps.Logger.Info("file already exists, skipping download", "path", savePath)
			return nil
		}

		// Download each clip
		var clipFiles []string
		for i, clipURL := range parsedResult.Clips {
			clipPath := fmt.Sprintf("%s/%s_clip%d.flv", p.Aid, p.Aid, i)
			if err := downloadTrack(ctx, deps, clipURL, clipPath, opt); err != nil {
				return fmt.Errorf("download clip %d: %w", i, err)
			}
			clipFiles = append(clipFiles, clipPath)
		}

		// Merge clips
		mergedPath := fmt.Sprintf("%s/%s.merged.flv", p.Aid, p.Aid)
		if err := deps.Muxer.MergeFLV(ctx, clipFiles, mergedPath); err != nil {
			return fmt.Errorf("merge flv failed: %w", err)
		}

		// Early return if SkipMux
		if opt.SkipMux {
			return nil
		}

		// Mux
		var subs []entity.Subtitle
		if !opt.SkipSubtitle {
			subs, _ = util.GetSubtitles(ctx, deps.HTTPClient, workCfg.Config, p.Aid, p.Cid, p.Epid, p.Index, opt.UseIntlApi)
			if opt.SkipAi {
				subs = filterAiSubtitles(subs)
			}
		}

		muxCfg := muxer.MuxConfig{
			BVid:      p.BVid(),
			VideoPath: mergedPath,
			OutPath:   savePath,
			Desc:      p.Desc,
			Title:     title,
			Author:    p.OwnerName,
			EpisodeID: p.Epid,
			Pic:       coverPath,
			Lang:      workCfg.Lang,
			Subtitles: subs,
			VideoOnly: true,
			SimplyMux: opt.SimplyMux,
			Points:    p.Points,
			PubTime:   vInfo.PubTime,
		}
		if err := deps.Muxer.Mux(ctx, muxCfg); err != nil {
			return fmt.Errorf("mux failed: %w", err)
		}

		// Clean up temp files
		for _, cf := range clipFiles {
			_ = os.Remove(cf)
		}
		_ = os.Remove(mergedPath)
		for _, sub := range subs {
			if sub.Path != "" {
				_ = os.Remove(sub.Path)
			}
		}
		if coverPath != "" {
			_ = os.Remove(coverPath)
		}

		return nil
	}

	// Handle failure: if no tracks and no clips, log error
	if len(parsedResult.VideoTracks) == 0 && len(parsedResult.AudioTracks) == 0 && len(parsedResult.Clips) == 0 {
		deps.Logger.Error("no tracks or clips found")
	}

	return nil
}

var printStreamsImpl = func(logger *slog.Logger, videos []entity.Video, audios []entity.Audio, bgAudios []entity.Audio, roleAudioList []entity.AudioMaterialInfo) {
	defaultPrintStreams(logger, videos, audios, bgAudios, roleAudioList)
}

func defaultPrintStreams(logger *slog.Logger, videos []entity.Video, audios []entity.Audio, bgAudios []entity.Audio, roleAudioList []entity.AudioMaterialInfo) {
	if len(videos) > 0 {
		logger.Info("video tracks")
		for i, v := range videos {
			logger.Info(fmt.Sprintf("  [%d] %s | %s | %s | %s | %dKbps", i, v.Dfn, v.Res, v.Fps, v.Codecs, v.Bandwith))
		}
	}
	if len(audios) > 0 {
		logger.Info("audio tracks")
		for i, a := range audios {
			logger.Info(fmt.Sprintf("  [%d] %s | %s | %dKbps", i, a.Dfn, a.Codecs, a.Bandwith))
		}
	}
	if len(bgAudios) > 0 {
		logger.Info("background audio tracks")
		for i, a := range bgAudios {
			logger.Info(fmt.Sprintf("  [%d] %s | %s | %dKbps", i, a.Dfn, a.Codecs, a.Bandwith))
		}
	}
	if len(roleAudioList) > 0 {
		logger.Info("role audio tracks")
		for i, r := range roleAudioList {
			logger.Info(fmt.Sprintf("  [%d] %s - %s", i, r.Title, r.PersonName))
		}
	}
}

func downloadTrack(ctx context.Context, deps DownloadDeps, url, path string, opt *cli.Option) error {
	if opt.UseAria2c {
		if !download.IsAria2cAvailable() {
			return fmt.Errorf("aria2c requested but not found in PATH, current directory, or program directory")
		}
		cfg := deps.Config
		if cfg == nil {
			cfg = config.NewConfig()
		}
		return download.DownloadWithAria2c(ctx, url, path, opt.Aria2cArgs, cfg)
	}

	dlOpts := download.Options{
		UseAria2c:    false,
		Aria2cArgs:   opt.Aria2cArgs,
		ForceHTTP:    opt.ForceHttp,
		MultiThread:  opt.MultiThread,
		ShowProgress: true,
	}
	if opt.MultiThread {
		return deps.Downloader.DownloadMultiThread(ctx, url, path, dlOpts)
	}
	return deps.Downloader.Download(ctx, url, path, dlOpts)
}

func downloadFile(ctx context.Context, client httpclient.Client, url, path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	resp, err := client.Get(ctx, url)
	if err != nil {
		return fmt.Errorf("download file request failed: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read file response: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create file dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// FetchPoints fetches viewpoint/chapter data from Bilibili API.
func FetchPoints(ctx context.Context, client httpclient.Client, cid, aid string) ([]entity.ViewPoint, error) {
	api := fmt.Sprintf("https://api.bilibili.com/x/player/wbi/v2?cid=%s&aid=%s", cid, aid)
	resp, err := client.Get(ctx, api)
	if err != nil {
		return nil, fmt.Errorf("fetch points request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read fetch points response: %w", err)
	}

	var result struct {
		Data struct {
			ViewPoints []struct {
				Content string `json:"content"`
				From    int    `json:"from"`
				To      int    `json:"to"`
			} `json:"view_points"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse fetch points response: %w", err)
	}

	points := make([]entity.ViewPoint, 0, len(result.Data.ViewPoints))
	for _, vp := range result.Data.ViewPoints {
		points = append(points, entity.ViewPoint{
			Title: vp.Content,
			Start: vp.From,
			End:   vp.To,
		})
	}

	return points, nil
}

func downloadCover(ctx context.Context, client httpclient.Client, picURL, coverPath string) error {
	if _, err := os.Stat(coverPath); err == nil {
		return nil
	}

	resp, err := client.Get(ctx, picURL)
	if err != nil {
		return fmt.Errorf("download cover request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read cover response: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(coverPath), 0o755); err != nil {
		return fmt.Errorf("create cover dir: %w", err)
	}

	if err := os.WriteFile(coverPath, data, 0o644); err != nil {
		return fmt.Errorf("write cover file: %w", err)
	}

	return nil
}

func filterAiSubtitles(subs []entity.Subtitle) []entity.Subtitle {
	filtered := make([]entity.Subtitle, 0, len(subs))
	for _, s := range subs {
		if !strings.HasPrefix(s.Lan, "ai-") {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func cleanEmptyDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return os.RemoveAll(dir)
	}
	return nil
}
