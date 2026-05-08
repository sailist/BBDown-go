package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"strconv"
	"strings"

	"github.com/sailist/BBDown-go/internal/config"
	"github.com/sailist/BBDown-go/internal/core/entity"
	"github.com/sailist/BBDown-go/pkg/httpclient"
)

// FavListFetcher fetches video metadata for favorite list (favId:) IDs.
type FavListFetcher struct {
	client httpclient.Client
	cfg    *config.Config
	logger *slog.Logger
}

// NewFavListFetcher creates a new FavListFetcher.
func NewFavListFetcher(client httpclient.Client, cfg *config.Config, logger *slog.Logger) Fetcher {
	return &FavListFetcher{
		client: client,
		cfg:    cfg,
		logger: logger,
	}
}

// Fetch retrieves favorite list metadata for the given favId.
func (f *FavListFetcher) Fetch(ctx context.Context, id string) (*entity.VInfo, error) {
	id = id[6:] // strip "favId:" prefix
	parts := strings.SplitN(id, ":", 2)
	favId := parts[0]
	mid := ""
	if len(parts) > 1 {
		mid = parts[1]
	}

	if favId == "" {
		favListApi := fmt.Sprintf("https://api.bilibili.com/x/v3/fav/folder/created/list-all?up_mid=%s", mid)
		f.logger.Debug("fetching default fav list", slog.String("url", favListApi))

		resp, err := f.client.Get(ctx, favListApi)
		if err != nil {
			return nil, fmt.Errorf("fetch default fav list: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read default fav list response: %w", err)
		}

		var favListJson struct {
			Data struct {
				List []struct {
					ID int64 `json:"id"`
				} `json:"list"`
			} `json:"data"`
		}

		if err := json.Unmarshal(body, &favListJson); err != nil {
			return nil, fmt.Errorf("parse default fav list json: %w", err)
		}

		if len(favListJson.Data.List) > 0 {
			favId = strconv.FormatInt(favListJson.Data.List[0].ID, 10)
		}
	}

	pageSize := 20
	index := 1
	pagesInfo := make([]entity.Page, 0)

	api := fmt.Sprintf("https://api.bilibili.com/x/v3/fav/resource/list?media_id=%s&pn=1&ps=%d&order=mtime&type=2&tid=0&platform=web", favId, pageSize)
	f.logger.Debug("fetching fav list info", slog.String("url", api))

	resp, err := f.client.Get(ctx, api)
	if err != nil {
		return nil, fmt.Errorf("fetch fav list info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read fav list info response: %w", err)
	}

	var infoJson struct {
		Data struct {
			Info struct {
				Title       string `json:"title"`
				Intro       string `json:"intro"`
				CTime       int64  `json:"ctime"`
				MediaCount  int    `json:"media_count"`
				Upper       struct {
					Name string `json:"name"`
				} `json:"upper"`
			} `json:"info"`
			Medias []struct {
				Attr   int    `json:"attr"`
				ID     int64  `json:"id"`
				Title  string `json:"title"`
				Intro  string `json:"intro"`
				Page   int    `json:"page"`
				Duration int `json:"duration"`
				PubTime int64 `json:"pubtime"`
				Cover  string `json:"cover"`
				Ugc    struct {
					FirstCid int64 `json:"first_cid"`
				} `json:"ugc"`
				Upper struct {
					Name string `json:"name"`
					Mid  int64  `json:"mid"`
				} `json:"upper"`
			} `json:"medias"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &infoJson); err != nil {
		return nil, fmt.Errorf("parse fav list info json: %w", err)
	}

	totalCount := infoJson.Data.Info.MediaCount
	totalPage := int(math.Ceil(float64(totalCount) / float64(pageSize)))
	title := infoJson.Data.Info.Title
	intro := infoJson.Data.Info.Intro
	pubTime := infoJson.Data.Info.CTime
	userName := infoJson.Data.Info.Upper.Name

	medias := make([]struct {
		Attr   int    `json:"attr"`
		ID     int64  `json:"id"`
		Title  string `json:"title"`
		Intro  string `json:"intro"`
		Page   int    `json:"page"`
		Duration int `json:"duration"`
		PubTime int64 `json:"pubtime"`
		Cover  string `json:"cover"`
		Ugc    struct {
			FirstCid int64 `json:"first_cid"`
		} `json:"ugc"`
		Upper struct {
			Name string `json:"name"`
			Mid  int64  `json:"mid"`
		} `json:"upper"`
	}, 0, totalCount)
	medias = append(medias, infoJson.Data.Medias...)

	for page := 2; page <= totalPage; page++ {
		api := fmt.Sprintf("https://api.bilibili.com/x/v3/fav/resource/list?media_id=%s&pn=%d&ps=%d&order=mtime&type=2&tid=0&platform=web", favId, page, pageSize)
		f.logger.Debug("fetching fav list page", slog.String("url", api))

		resp, err := f.client.Get(ctx, api)
		if err != nil {
			return nil, fmt.Errorf("fetch fav list page: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read fav list page response: %w", err)
		}

		var pageJson struct {
			Data struct {
				Medias []struct {
					Attr   int    `json:"attr"`
					ID     int64  `json:"id"`
					Title  string `json:"title"`
					Intro  string `json:"intro"`
					Page   int    `json:"page"`
					Duration int `json:"duration"`
					PubTime int64 `json:"pubtime"`
					Cover  string `json:"cover"`
					Ugc    struct {
						FirstCid int64 `json:"first_cid"`
					} `json:"ugc"`
					Upper struct {
						Name string `json:"name"`
						Mid  int64  `json:"mid"`
					} `json:"upper"`
				} `json:"medias"`
			} `json:"data"`
		}

		if err := json.Unmarshal(body, &pageJson); err != nil {
			return nil, fmt.Errorf("parse fav list page json: %w", err)
		}

		medias = append(medias, pageJson.Data.Medias...)
	}

	for _, m := range medias {
		if m.Attr != 0 {
			continue
		}
		pageCount := m.Page
		if pageCount > 1 {
			normalFetcher := NewNormalFetcher(f.client, f.cfg, f.logger)
			tmpInfo, err := normalFetcher.Fetch(ctx, strconv.FormatInt(m.ID, 10))
			if err != nil {
				return nil, fmt.Errorf("fetch normal info for fav media %d: %w", m.ID, err)
			}
			for _, item := range tmpInfo.PagesInfo {
				p := entity.Page{
					Index:     index,
					Aid:       item.Aid,
					Cid:       item.Cid,
					Epid:      item.Epid,
					Title:     fmt.Sprintf("%s_P%d_%s", m.Title, item.Index, item.Title),
					Dur:       item.Dur,
					Res:       item.Res,
					PubTime:   item.PubTime,
					Cover:     tmpInfo.Pic,
					Desc:      m.Intro,
					OwnerName: item.OwnerName,
					OwnerMid:  item.OwnerMid,
				}
				if !containsPage(pagesInfo, p) {
					pagesInfo = append(pagesInfo, p)
					index++
				}
			}
		} else {
			p := entity.Page{
				Index:     index,
				Aid:       strconv.FormatInt(m.ID, 10),
				Cid:       strconv.FormatInt(m.Ugc.FirstCid, 10),
				Title:     m.Title,
				Dur:       m.Duration,
				Res:       "",
				PubTime:   m.PubTime,
				Cover:     m.Cover,
				Desc:      m.Intro,
				OwnerName: m.Upper.Name,
				OwnerMid:  strconv.FormatInt(m.Upper.Mid, 10),
			}
			if !containsPage(pagesInfo, p) {
				pagesInfo = append(pagesInfo, p)
				index++
			}
		}
	}

	_ = userName // userName is fetched but not used in VInfo directly

	return &entity.VInfo{
		Title:     strings.TrimSpace(title),
		Desc:      strings.TrimSpace(intro),
		Pic:       "",
		PubTime:   pubTime,
		PagesInfo: pagesInfo,
		IsBangumi: false,
	}, nil
}
