package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"github.com/sailist/BBDown-go/internal/config"
	"github.com/sailist/BBDown-go/internal/core/entity"
	"github.com/sailist/BBDown-go/pkg/httpclient"
)

// MediaListFetcher fetches video metadata for media list (listBizId:) IDs.
type MediaListFetcher struct {
	client httpclient.Client
	cfg    *config.Config
	logger *slog.Logger
}

// NewMediaListFetcher creates a new MediaListFetcher.
func NewMediaListFetcher(client httpclient.Client, cfg *config.Config, logger *slog.Logger) Fetcher {
	return &MediaListFetcher{
		client: client,
		cfg:    cfg,
		logger: logger,
	}
}

// Fetch retrieves media list metadata for the given listBizId.
func (f *MediaListFetcher) Fetch(ctx context.Context, id string) (*entity.VInfo, error) {
	id = id[10:] // strip "listBizId:" prefix
	api := fmt.Sprintf("https://api.bilibili.com/x/v1/medialist/info?type=8&biz_id=%s&tid=0", id)
	f.logger.Debug("fetching media list info", slog.String("url", api))

	resp, err := f.client.Get(ctx, api)
	if err != nil {
		return nil, fmt.Errorf("fetch media list info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read media list info response: %w", err)
	}

	var infoJson struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    *struct {
			Title  string `json:"title"`
			Intro  string `json:"intro"`
			CTime  int64  `json:"ctime"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &infoJson); err != nil {
		return nil, fmt.Errorf("parse media list info json: %w", err)
	}

	if infoJson.Data == nil {
		seriesFetcher := NewSeriesListFetcher(f.client, f.cfg, f.logger)
		vInfo, err := seriesFetcher.Fetch(ctx, fmt.Sprintf("seriesBizId:%s", id))
		if err != nil {
			code := infoJson.Code
			message := infoJson.Message
			if message == "" {
				message = "未知错误"
			}
			return nil, fmt.Errorf("获取合集信息失败(code=%d): %s", code, message)
		}
		return vInfo, nil
	}

	listTitle := infoJson.Data.Title
	intro := infoJson.Data.Intro
	pubTime := infoJson.Data.CTime

	pagesInfo := make([]entity.Page, 0)
	hasMore := true
	oid := ""
	index := 1

	for hasMore {
		listApi := fmt.Sprintf("https://api.bilibili.com/x/v2/medialist/resource/list?type=8&oid=%s&otype=2&biz_id=%s&with_current=true&mobi_app=web&ps=20&direction=false&sort_field=1&tid=0&desc=false", oid, id)
		f.logger.Debug("fetching media list page", slog.String("url", listApi))

		resp, err := f.client.Get(ctx, listApi)
		if err != nil {
			return nil, fmt.Errorf("fetch media list page: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read media list page response: %w", err)
		}

		var listJson struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    *struct {
				HasMore   bool `json:"has_more"`
				MediaList []struct {
					Attr   int    `json:"attr"`
					ID     int64  `json:"id"`
					Title  string `json:"title"`
					Intro  string `json:"intro"`
					Page   int    `json:"page"`
					PubTime int64 `json:"pubtime"`
					Cover  string `json:"cover"`
					Upper  struct {
						Name string `json:"name"`
						Mid  int64  `json:"mid"`
					} `json:"upper"`
					Pages []struct {
						ID        int64  `json:"id"`
						Page      int    `json:"page"`
						Title     string `json:"title"`
						Duration  int    `json:"duration"`
						Dimension struct {
							Width  int `json:"width"`
							Height int `json:"height"`
						} `json:"dimension"`
					} `json:"pages"`
				} `json:"media_list"`
			} `json:"data"`
		}

		if err := json.Unmarshal(body, &listJson); err != nil {
			return nil, fmt.Errorf("parse media list page json: %w", err)
		}

		if listJson.Data == nil {
			code := listJson.Code
			message := listJson.Message
			if message == "" {
				message = "未知错误"
			}
			return nil, fmt.Errorf("获取合集视频列表失败(code=%d): %s", code, message)
		}

		hasMore = listJson.Data.HasMore
		for _, m := range listJson.Data.MediaList {
			if m.Attr != 0 {
				continue
			}
			pageCount := m.Page
			desc := m.Intro
			ownerName := m.Upper.Name
			ownerMid := strconv.FormatInt(m.Upper.Mid, 10)
			for _, page := range m.Pages {
				title := m.Title
				if pageCount != 1 {
					title = fmt.Sprintf("%s_P%d_%s", m.Title, page.Page, page.Title)
				}
				p := entity.Page{
					Index:     index,
					Aid:       strconv.FormatInt(m.ID, 10),
					Cid:       strconv.FormatInt(page.ID, 10),
					Title:     title,
					Dur:       page.Duration,
					Res:       fmt.Sprintf("%dx%d", page.Dimension.Width, page.Dimension.Height),
					PubTime:   m.PubTime,
					Cover:     m.Cover,
					Desc:      desc,
					OwnerName: ownerName,
					OwnerMid:  ownerMid,
				}
				if !containsPage(pagesInfo, p) {
					pagesInfo = append(pagesInfo, p)
					index++
				}
			}
			oid = strconv.FormatInt(m.ID, 10)
		}
	}

	return &entity.VInfo{
		Title:     strings.TrimSpace(listTitle),
		Desc:      strings.TrimSpace(intro),
		Pic:       "",
		PubTime:   pubTime,
		PagesInfo: pagesInfo,
		IsBangumi: false,
	}, nil
}
