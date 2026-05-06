package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"strings"

	"github.com/nilaonai/bbdown-go/internal/config"
	"github.com/nilaonai/bbdown-go/internal/core/entity"
	"github.com/nilaonai/bbdown-go/internal/core/util"
	"github.com/nilaonai/bbdown-go/pkg/httpclient"
)

// SpaceVideoFetcher fetches video metadata for user space (mid:) IDs.
type SpaceVideoFetcher struct {
	client httpclient.Client
	cfg    *config.Config
	logger *slog.Logger
}

// NewSpaceVideoFetcher creates a new SpaceVideoFetcher.
func NewSpaceVideoFetcher(client httpclient.Client, cfg *config.Config, logger *slog.Logger) Fetcher {
	return &SpaceVideoFetcher{
		client: client,
		cfg:    cfg,
		logger: logger,
	}
}

// Fetch retrieves user space video metadata for the given mid.
func (f *SpaceVideoFetcher) Fetch(ctx context.Context, id string) (*entity.VInfo, error) {
	id = id[4:] // strip "mid:" prefix

	userInfoApi := fmt.Sprintf("https://api.live.bilibili.com/live_user/v1/Master/info?uid=%s", id)
	f.logger.Debug("fetching user info", slog.String("url", userInfoApi))

	resp, err := f.client.Get(ctx, userInfoApi)
	if err != nil {
		return nil, fmt.Errorf("fetch user info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read user info response: %w", err)
	}

	var userInfoJson struct {
		Data struct {
			Info struct {
				Uname string `json:"uname"`
			} `json:"info"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &userInfoJson); err != nil {
		return nil, fmt.Errorf("parse user info json: %w", err)
	}

	userName := util.GetValidFileName(userInfoJson.Data.Info.Uname, ".", true)

	pageSize := 50
	pageNumber := 1

	query := fmt.Sprintf("mid=%s&order=pubdate&pn=%d&ps=%d&tid=0&wts=%s", id, pageNumber, pageSize, util.GetTimestamp(true))
	signed := util.WbiSign(query, f.cfg.WBI)
	api := fmt.Sprintf("https://api.bilibili.com/x/space/wbi/arc/search?%s", signed)
	f.logger.Debug("fetching space video list", slog.String("url", api))

	resp, err = f.client.Get(ctx, api)
	if err != nil {
		return nil, fmt.Errorf("fetch space video list: %w", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read space video list response: %w", err)
	}

	var infoJson struct {
		Data struct {
			List struct {
				Vlist []struct {
					Aid int64 `json:"aid"`
				} `json:"vlist"`
			} `json:"list"`
			Page struct {
				Count int `json:"count"`
			} `json:"page"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &infoJson); err != nil {
		return nil, fmt.Errorf("parse space video list json: %w", err)
	}

	urls := make([]string, 0)
	for _, page := range infoJson.Data.List.Vlist {
		urls = append(urls, fmt.Sprintf("https://www.bilibili.com/video/av%d", page.Aid))
	}

	totalCount := infoJson.Data.Page.Count
	totalPage := int(math.Ceil(float64(totalCount) / float64(pageSize)))

	for pageNumber < totalPage {
		pageNumber++
		pageUrls, err := f.getVideosByPage(ctx, pageNumber, pageSize, id)
		if err != nil {
			return nil, fmt.Errorf("fetch space video page %d: %w", pageNumber, err)
		}
		urls = append(urls, pageUrls...)
	}

	fileName := fmt.Sprintf("%s的投稿视频.txt", userName)
	if err := os.WriteFile(fileName, []byte(strings.Join(urls, "\n")), 0644); err != nil {
		return nil, fmt.Errorf("write space video urls file: %w", err)
	}

	f.logger.Info("目前下载器不支持下载用户的全部投稿视频...")
	return nil, fmt.Errorf("暂不支持该功能")
}

func (f *SpaceVideoFetcher) getVideosByPage(ctx context.Context, pageNumber, pageSize int, id string) ([]string, error) {
	query := fmt.Sprintf("mid=%s&order=pubdate&pn=%d&ps=%d&tid=0&wts=%s", id, pageNumber, pageSize, util.GetTimestamp(true))
	signed := util.WbiSign(query, f.cfg.WBI)
	api := fmt.Sprintf("https://api.bilibili.com/x/space/wbi/arc/search?%s", signed)
	f.logger.Debug("fetching space video page", slog.String("url", api))

	resp, err := f.client.Get(ctx, api)
	if err != nil {
		return nil, fmt.Errorf("fetch space video page: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read space video page response: %w", err)
	}

	var pageJson struct {
		Data struct {
			List struct {
				Vlist []struct {
					Aid int64 `json:"aid"`
				} `json:"vlist"`
			} `json:"list"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &pageJson); err != nil {
		return nil, fmt.Errorf("parse space video page json: %w", err)
	}

	urls := make([]string, 0, len(pageJson.Data.List.Vlist))
	for _, page := range pageJson.Data.List.Vlist {
		urls = append(urls, fmt.Sprintf("https://www.bilibili.com/video/av%d", page.Aid))
	}

	return urls, nil
}
