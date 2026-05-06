package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/nilaonai/bbdown-go/internal/cli"
	"github.com/nilaonai/bbdown-go/internal/core/entity"
	"github.com/nilaonai/bbdown-go/internal/core/util"
	"github.com/nilaonai/bbdown-go/internal/download"
	"github.com/nilaonai/bbdown-go/internal/muxer"
	"github.com/nilaonai/bbdown-go/pkg/httpclient"
)

// DownloadDeps holds dependencies for DownloadPage, enabling test mocking.
type DownloadDeps struct {
	HTTPClient httpclient.Client
	Logger     *slog.Logger
	Downloader download.Downloader
	Muxer      muxer.Muxer
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

	// TODO: part 2 - stream selection and download
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
