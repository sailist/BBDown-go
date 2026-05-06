package fetcher

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/nilaonai/bbdown-go/internal/config"
	"github.com/nilaonai/bbdown-go/internal/core/entity"
	"github.com/nilaonai/bbdown-go/pkg/httpclient"
)

var epRegex = regexp.MustCompile(`ep(\d+)`)

// NormalFetcher fetches video metadata for normal AV/BV videos.
type NormalFetcher struct {
	client httpclient.Client
	cfg    *config.Config
	logger *slog.Logger
}

// NewNormalFetcher creates a new NormalFetcher.
func NewNormalFetcher(client httpclient.Client, cfg *config.Config, logger *slog.Logger) Fetcher {
	return &NormalFetcher{
		client: client,
		cfg:    cfg,
		logger: logger,
	}
}

// Fetch retrieves video metadata for the given aid.
func (f *NormalFetcher) Fetch(ctx context.Context, id string) (*entity.VInfo, error) {
	api := fmt.Sprintf("https://api.bilibili.com/x/web-interface/view?aid=%s", id)
	f.logger.Debug("fetching normal video info", slog.String("url", api))

	resp, err := f.client.Get(ctx, api)
	if err != nil {
		return nil, fmt.Errorf("fetch video info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read video info response: %w", err)
	}

	var viewResp struct {
		Data struct {
			Title       string `json:"title"`
			Desc        string `json:"desc"`
			Pic         string `json:"pic"`
			Pubdate     int64  `json:"pubdate"`
			Bvid        string `json:"bvid"`
			Cid         int64  `json:"cid"`
			RedirectURL string `json:"redirect_url"`
			Owner       struct {
				Mid  int64  `json:"mid"`
				Name string `json:"name"`
			} `json:"owner"`
			Rights struct {
				IsSteinGate int `json:"is_stein_gate"`
			} `json:"rights"`
			Pages []struct {
				Page      int    `json:"page"`
				Cid       int64  `json:"cid"`
				Part      string `json:"part"`
				Duration  int    `json:"duration"`
				Dimension struct {
					Width  int `json:"width"`
					Height int `json:"height"`
				} `json:"dimension"`
			} `json:"pages"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &viewResp); err != nil {
		return nil, fmt.Errorf("parse video info json: %w", err)
	}

	data := viewResp.Data
	title := data.Title
	desc := data.Desc
	pic := data.Pic
	ownerMid := strconv.FormatInt(data.Owner.Mid, 10)
	ownerName := data.Owner.Name
	pubTime := data.Pubdate
	bvid := data.Bvid
	cid := data.Cid
	isSteinGate := data.Rights.IsSteinGate
	bangumi := false

	pagesInfo := make([]entity.Page, 0, len(data.Pages))
	for _, page := range data.Pages {
		p := entity.Page{
			Index:     page.Page,
			Aid:       id,
			Cid:       strconv.FormatInt(page.Cid, 10),
			Title:     strings.TrimSpace(page.Part),
			Dur:       page.Duration,
			Res:       fmt.Sprintf("%dx%d", page.Dimension.Width, page.Dimension.Height),
			PubTime:   pubTime,
			OwnerName: ownerName,
			OwnerMid:  ownerMid,
		}
		pagesInfo = append(pagesInfo, p)
	}

	if isSteinGate == 1 {
		if err := f.fetchSteinGatePages(ctx, id, bvid, cid, pubTime, ownerName, ownerMid, &pagesInfo); err != nil {
			return nil, fmt.Errorf("fetch stein gate pages: %w", err)
		}
	}

	if strings.Contains(data.RedirectURL, "bangumi") {
		bangumi = true
		matches := epRegex.FindStringSubmatch(data.RedirectURL)
		if len(matches) > 1 {
			epID := matches[1]
			if len(data.Pages) == 1 {
				for i := range pagesInfo {
					pagesInfo[i].Epid = epID
				}
			}
		}
	}

	return &entity.VInfo{
		Title:       strings.TrimSpace(title),
		Desc:        strings.TrimSpace(desc),
		Pic:         pic,
		PubTime:     pubTime,
		PagesInfo:   pagesInfo,
		IsBangumi:   bangumi,
		IsSteinGate: isSteinGate == 1,
	}, nil
}

func (f *NormalFetcher) fetchSteinGatePages(ctx context.Context, id, bvid string, cid, pubTime int64, ownerName, ownerMid string, pagesInfo *[]entity.Page) error {
	playerSoAPI := fmt.Sprintf("https://api.bilibili.com/x/player.so?bvid=%s&id=cid:%d", bvid, cid)
	f.logger.Debug("fetching player.so", slog.String("url", playerSoAPI))

	resp, err := f.client.Get(ctx, playerSoAPI)
	if err != nil {
		return fmt.Errorf("fetch player.so: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read player.so response: %w", err)
	}

	wrapped := fmt.Sprintf("<root>%s</root>", string(body))
	var root struct {
		Interaction string `xml:"interaction"`
	}
	if err := xml.Unmarshal([]byte(wrapped), &root); err != nil {
		return fmt.Errorf("parse player.so xml: %w", err)
	}

	if len(root.Interaction) == 0 {
		return fmt.Errorf("互动视频获取分P信息失败")
	}

	var interaction struct {
		GraphVersion int64 `json:"graph_version"`
	}
	if err := json.Unmarshal([]byte(root.Interaction), &interaction); err != nil {
		return fmt.Errorf("parse interaction json: %w", err)
	}

	edgeInfoAPI := fmt.Sprintf("https://api.bilibili.com/x/stein/edgeinfo_v2?graph_version=%d&bvid=%s", interaction.GraphVersion, bvid)
	f.logger.Debug("fetching edge info", slog.String("url", edgeInfoAPI))

	resp2, err := f.client.Get(ctx, edgeInfoAPI)
	if err != nil {
		return fmt.Errorf("fetch edge info: %w", err)
	}
	defer resp2.Body.Close()

	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		return fmt.Errorf("read edge info response: %w", err)
	}

	var edgeResp struct {
		Data struct {
			Edges struct {
				Questions []struct {
					Choices []struct {
						Cid    int64  `json:"cid"`
						Option string `json:"option"`
					} `json:"choices"`
				} `json:"questions"`
			} `json:"edges"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body2, &edgeResp); err != nil {
		return fmt.Errorf("parse edge info json: %w", err)
	}

	index := 2
	for _, question := range edgeResp.Data.Edges.Questions {
		for _, choice := range question.Choices {
			p := entity.Page{
				Index:     index,
				Aid:       id,
				Cid:       strconv.FormatInt(choice.Cid, 10),
				Title:     strings.TrimSpace(choice.Option),
				Dur:       0,
				PubTime:   pubTime,
				OwnerName: ownerName,
				OwnerMid:  ownerMid,
			}
			*pagesInfo = append(*pagesInfo, p)
			index++
		}
	}

	return nil
}
