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

// CheeseFetcher fetches video metadata for cheese (cheese:) IDs.
type CheeseFetcher struct {
	client httpclient.Client
	cfg    *config.Config
	logger *slog.Logger
}

// NewCheeseFetcher creates a new CheeseFetcher.
func NewCheeseFetcher(client httpclient.Client, cfg *config.Config, logger *slog.Logger) Fetcher {
	return &CheeseFetcher{
		client: client,
		cfg:    cfg,
		logger: logger,
	}
}

// Fetch retrieves cheese course metadata for the given ep_id.
func (f *CheeseFetcher) Fetch(ctx context.Context, id string) (*entity.VInfo, error) {
	id = id[7:] // strip "cheese:" prefix
	index := ""
	api := fmt.Sprintf("https://api.bilibili.com/pugv/view/web/season?ep_id=%s", id)
	f.logger.Debug("fetching cheese info", slog.String("url", api))

	resp, err := f.client.Get(ctx, api)
	if err != nil {
		return nil, fmt.Errorf("fetch cheese info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read cheese info response: %w", err)
	}

	var infoJson struct {
		Data struct {
			Cover    string `json:"cover"`
			Title    string `json:"title"`
			Subtitle string `json:"subtitle"`
			UpInfo   struct {
				Uname string `json:"uname"`
				Mid   int64  `json:"mid"`
			} `json:"up_info"`
			Episodes []struct {
				Index       int    `json:"index"`
				Aid         int64  `json:"aid"`
				Cid         int64  `json:"cid"`
				ID          int64  `json:"id"`
				Title       string `json:"title"`
				Duration    int    `json:"duration"`
				ReleaseDate int64  `json:"release_date"`
			} `json:"episodes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &infoJson); err != nil {
		return nil, fmt.Errorf("parse cheese info json: %w", err)
	}

	data := infoJson.Data
	cover := data.Cover
	title := data.Title
	desc := data.Subtitle
	ownerName := data.UpInfo.Uname
	ownerMid := strconv.FormatInt(data.UpInfo.Mid, 10)

	pagesInfo := make([]entity.Page, 0, len(data.Episodes))
	for _, page := range data.Episodes {
		p := entity.Page{
			Index:     page.Index,
			Aid:       strconv.FormatInt(page.Aid, 10),
			Cid:       strconv.FormatInt(page.Cid, 10),
			Epid:      strconv.FormatInt(page.ID, 10),
			Title:     strings.TrimSpace(page.Title),
			Dur:       page.Duration,
			Res:       "",
			PubTime:   page.ReleaseDate,
			OwnerName: ownerName,
			OwnerMid:  ownerMid,
		}
		if p.Epid == id {
			index = strconv.Itoa(p.Index)
		}
		pagesInfo = append(pagesInfo, p)
	}

	var pubTime int64
	if len(pagesInfo) > 0 {
		pubTime = pagesInfo[0].PubTime
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
