package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/nilaonai/bbdown-go/internal/config"
	"github.com/nilaonai/bbdown-go/internal/core/entity"
	"github.com/nilaonai/bbdown-go/pkg/httpclient"
)

// BangumiFetcher fetches video metadata for bangumi (ep:) IDs.
type BangumiFetcher struct {
	client httpclient.Client
	cfg    *config.Config
	logger *slog.Logger
}

// NewBangumiFetcher creates a new BangumiFetcher.
func NewBangumiFetcher(client httpclient.Client, cfg *config.Config, logger *slog.Logger) Fetcher {
	return &BangumiFetcher{
		client: client,
		cfg:    cfg,
		logger: logger,
	}
}

// Fetch retrieves bangumi metadata for the given ep_id.
func (f *BangumiFetcher) Fetch(ctx context.Context, id string) (*entity.VInfo, error) {
	id = id[3:] // strip "ep:" prefix
	index := ""
	api := fmt.Sprintf("https://%s/pgc/view/web/season?ep_id=%s", f.cfg.EpHost, id)
	f.logger.Debug("fetching bangumi info", slog.String("url", api))

	resp, err := f.client.Get(ctx, api)
	if err != nil {
		return nil, fmt.Errorf("fetch bangumi info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read bangumi info response: %w", err)
	}

	var infoJson struct {
		Result struct {
			Cover     string          `json:"cover"`
			Title     string          `json:"title"`
			Evaluate  string          `json:"evaluate"`
			Publish   struct {
				PubTime string `json:"pub_time"`
			} `json:"publish"`
			Episodes json.RawMessage `json:"episodes"`
			Section  []struct {
				Title    string          `json:"title"`
				Episodes json.RawMessage `json:"episodes"`
			} `json:"section"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &infoJson); err != nil {
		return nil, fmt.Errorf("parse bangumi info json: %w", err)
	}

	result := infoJson.Result
	cover := result.Cover
	title := result.Title
	desc := result.Evaluate

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
	if err := json.Unmarshal(result.Episodes, &pages); err != nil {
		return nil, fmt.Errorf("parse episodes: %w", err)
	}

	hasEp := strings.Contains(string(result.Episodes), "/ep"+id)
	if !(len(pages) > 0 && hasEp) {
		for _, section := range result.Section {
			if strings.Contains(string(section.Episodes), "/ep"+id) {
				title += "[" + section.Title + "]"
				if err := json.Unmarshal(section.Episodes, &pages); err != nil {
					return nil, fmt.Errorf("parse section episodes: %w", err)
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
