package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nilaonai/bbdown-go/internal/config"
	"github.com/nilaonai/bbdown-go/internal/core/entity"
	"github.com/nilaonai/bbdown-go/pkg/httpclient"
)

// IntlBangumiFetcher fetches video metadata for international bangumi (ep:) IDs.
type IntlBangumiFetcher struct {
	client httpclient.Client
	cfg    *config.Config
	logger *slog.Logger
}

// NewIntlBangumiFetcher creates a new IntlBangumiFetcher.
func NewIntlBangumiFetcher(client httpclient.Client, cfg *config.Config, logger *slog.Logger) Fetcher {
	return &IntlBangumiFetcher{
		client: client,
		cfg:    cfg,
		logger: logger,
	}
}

var stateRegex = regexp.MustCompile(`window\.__INITIAL_STATE__=([\s\S]*?);\(function\(\)`)

// Fetch retrieves international bangumi metadata for the given ep_id.
func (f *IntlBangumiFetcher) Fetch(ctx context.Context, id string) (*entity.VInfo, error) {
	id = id[3:] // strip "ep:" prefix
	index := ""

	host := f.cfg.Host
	if host == "api.bilibili.com" {
		host = "api.bilibili.tv"
	}

	api := fmt.Sprintf("https://%s/intl/gateway/v2/ogv/view/app/season?ep_id=%s&platform=android&s_locale=zh_SG&mobi_app=bstar_a", host, id)
	if f.cfg.Token != "" {
		api += "&access_key=" + f.cfg.Token
	}
	f.logger.Debug("fetching intl bangumi info", slog.String("url", api))

	resp, err := f.client.Get(ctx, api)
	if err != nil {
		return nil, fmt.Errorf("fetch intl bangumi info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read intl bangumi info response: %w", err)
	}

	var infoJson struct {
		Result struct {
			SeasonID string `json:"season_id"`
			Cover    string `json:"cover"`
			Title    string `json:"title"`
			Evaluate string `json:"evaluate"`
			Publish  struct {
				PubTime string `json:"pub_time"`
			} `json:"publish"`
			Episodes json.RawMessage `json:"episodes"`
			Modules  []struct {
				Data struct {
					Episodes json.RawMessage `json:"episodes"`
				} `json:"data"`
			} `json:"modules"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &infoJson); err != nil {
		return nil, fmt.Errorf("parse intl bangumi info json: %w", err)
	}

	result := infoJson.Result
	cover := result.Cover
	title := result.Title
	desc := result.Evaluate

	if cover == "" {
		animeURL := fmt.Sprintf("https://bangumi.bilibili.com/anime/%s", result.SeasonID)
		f.logger.Debug("intl bangumi cover empty, fetching fallback", slog.String("url", animeURL))

		webResp, err := f.client.Get(ctx, animeURL)
		if err == nil {
			webBody, _ := io.ReadAll(webResp.Body)
			webResp.Body.Close()
			web := string(webBody)
			if web != "" {
				matches := stateRegex.FindStringSubmatch(web)
				if len(matches) > 1 {
					var tempJson struct {
						MediaInfo struct {
							Cover    string `json:"cover"`
							Title    string `json:"title"`
							Evaluate string `json:"evaluate"`
						} `json:"mediaInfo"`
					}
					if err := json.Unmarshal([]byte(matches[1]), &tempJson); err == nil {
						cover = tempJson.MediaInfo.Cover
						title = tempJson.MediaInfo.Title
						desc = tempJson.MediaInfo.Evaluate
					}
				}
			}
		}
	}

	var pubTime int64
	if result.Publish.PubTime != "" {
		t, err := time.Parse("2006-01-02 15:04:05", result.Publish.PubTime)
		if err != nil {
			return nil, fmt.Errorf("parse pub_time: %w", err)
		}
		pubTime = t.Unix()
	}

	type pageStruct struct {
		Badge     string `json:"badge"`
		Dimension struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"dimension"`
		Title     string `json:"title"`
		LongTitle string `json:"long_title"`
		Aid       int64  `json:"aid"`
		Cid       int64  `json:"cid"`
		ID        int64  `json:"id"`
		PubTime   int64  `json:"pub_time"`
	}

	var pages []pageStruct
	if result.Episodes != nil {
		if err := json.Unmarshal(result.Episodes, &pages); err != nil {
			return nil, fmt.Errorf("parse episodes: %w", err)
		}
	}

	if len(result.Modules) > 0 {
		for _, module := range result.Modules {
			if strings.Contains(string(module.Data.Episodes), "/"+id) {
				if err := json.Unmarshal(module.Data.Episodes, &pages); err != nil {
					return nil, fmt.Errorf("parse module episodes: %w", err)
				}
				break
			}
		}
	}

	pagesInfo := make([]entity.Page, 0, len(pages))
	i := 1
	for _, page := range pages {
		if page.Badge == "预告" {
			continue
		}

		res := ""
		if page.Dimension.Width != 0 || page.Dimension.Height != 0 {
			res = fmt.Sprintf("%dx%d", page.Dimension.Width, page.Dimension.Height)
		}

		pageTitle := strings.TrimSpace(page.Title + " " + page.LongTitle)

		p := entity.Page{
			Index:   i,
			Aid:     strconv.FormatInt(page.Aid, 10),
			Cid:     strconv.FormatInt(page.Cid, 10),
			Epid:    strconv.FormatInt(page.ID, 10),
			Title:   pageTitle,
			Dur:     0,
			Res:     res,
			PubTime: page.PubTime,
		}
		if p.Epid == id {
			index = strconv.Itoa(p.Index)
		}
		pagesInfo = append(pagesInfo, p)
		i++
	}

	return &entity.VInfo{
		Title:     strings.TrimSpace(title),
		Desc:      strings.TrimSpace(desc),
		Pic:       cover,
		PubTime:   pubTime,
		PagesInfo: pagesInfo,
		IsBangumi: true,
		IsCheese:  true,
		Index:     index,
	}, nil
}
