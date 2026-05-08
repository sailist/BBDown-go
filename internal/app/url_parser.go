package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/sailist/BBDown-go/pkg/bvconv"
	"github.com/sailist/BBDown-go/pkg/httpclient"
)

const defaultEpHost = "api.bilibili.com"

var (
	avRegex        = regexp.MustCompile(`av(\d+)`)
	bvRegex        = regexp.MustCompile(`([Bb][Vv]1\w+)`)
	epRegex        = regexp.MustCompile(`/ep(\d+)`)
	ssRegex        = regexp.MustCompile(`/ss(\d+)`)
	uidRegex       = regexp.MustCompile(`space\.bilibili\.com/(\d+)`)
	globalEpRegex  = regexp.MustCompile(`\.bilibili\.tv/\w+/play/\d+/(\d+)`)
	bangumiMdRegex = regexp.MustCompile(`bangumi/media/(md\d+)`)
	mdRegex        = regexp.MustCompile(`md(\d+)`)
	stateRegex     = regexp.MustCompile(`window\.__INITIAL_STATE__=([\s\S]*?);\(function\(\)`)
)

// ParseInput parses various Bilibili input formats into a normalized ID string.
func ParseInput(ctx context.Context, client httpclient.Client, input string) (string, error) {
	// Pass through already-normalized inputs
	if strings.HasPrefix(input, "listBizId:") ||
		strings.HasPrefix(input, "seriesBizId:") ||
		strings.HasPrefix(input, "favId:") ||
		strings.HasPrefix(input, "mid:") ||
		strings.HasPrefix(input, "cheese:") ||
		strings.HasPrefix(input, "ep:") {
		return input, nil
	}

	avid := input

	if strings.HasPrefix(input, "http") {
		// Resolve b23.tv short links
		if strings.Contains(input, "b23.tv") {
			location, err := client.GetRedirectLocation(ctx, input)
			if err != nil {
				return "", fmt.Errorf("failed to resolve b23.tv link: %w", err)
			}
			if location == input {
				return "", fmt.Errorf("infinite redirect for b23.tv link")
			}
			input = location
		}

		switch {
		case strings.Contains(input, "video/av"):
			match := avRegex.FindStringSubmatch(input)
			if len(match) > 1 {
				avid = match[1]
			}

		case strings.Contains(strings.ToLower(input), "video/bv"):
			match := bvRegex.FindStringSubmatch(input)
			if len(match) > 1 {
				aid, err := bvconv.Decode(match[1])
				if err != nil {
					return "", fmt.Errorf("failed to decode BV: %w", err)
				}
				avid = strconv.FormatInt(aid, 10)
			}

		case strings.Contains(input, "/cheese/"):
			epID := ""
			switch {
			case strings.Contains(input, "/ep"):
				match := epRegex.FindStringSubmatch(input)
				if len(match) > 1 {
					epID = match[1]
				}
			case strings.Contains(input, "/ss"):
				match := ssRegex.FindStringSubmatch(input)
				if len(match) > 1 {
					var err error
					epID, err = getEpidByCheeseSSID(ctx, client, match[1])
					if err != nil {
						return "", fmt.Errorf("get cheese ssid: %w", err)
					}
				}
			}
			avid = fmt.Sprintf("cheese:%s", epID)

		case strings.Contains(input, "/ep"):
			match := epRegex.FindStringSubmatch(input)
			if len(match) > 1 {
				avid = fmt.Sprintf("ep:%s", match[1])
			}

		case strings.Contains(input, "/ss"):
			match := ssRegex.FindStringSubmatch(input)
			if len(match) > 1 {
				epID, err := getEpidByBangumiSSID(ctx, client, match[1])
				if err != nil {
					return "", fmt.Errorf("get bangumi ssid: %w", err)
				}
				avid = fmt.Sprintf("ep:%s", epID)
			}

		case strings.Contains(input, "/medialist/") && strings.Contains(input, "business_id=") && strings.Contains(input, "business=space_collection"):
			bizID := getQueryString("business_id", input)
			avid = fmt.Sprintf("listBizId:%s", bizID)

		case strings.Contains(input, "/medialist/") && strings.Contains(input, "business_id=") && strings.Contains(input, "business=space_series"):
			bizID := getQueryString("business_id", input)
			avid = fmt.Sprintf("seriesBizId:%s", bizID)

		case strings.Contains(input, "/channel/collectiondetail?sid="):
			bizID := getQueryString("sid", input)
			avid = fmt.Sprintf("listBizId:%s", bizID)

		case strings.Contains(input, "/channel/seriesdetail?sid="):
			bizID := getQueryString("sid", input)
			avid = fmt.Sprintf("seriesBizId:%s", bizID)

		case strings.Contains(input, "/space.bilibili.com/") && strings.Contains(input, "/lists/"):
			typ := strings.ToLower(getQueryString("type", input))
			path := input
			if idx := strings.Index(path, "?"); idx != -1 {
				path = path[:idx]
			}
			if idx := strings.Index(path, "#"); idx != -1 {
				path = path[:idx]
			}
			sidPart := path[strings.LastIndex(path, "/")+1:]
			if typ == "series" {
				avid = fmt.Sprintf("seriesBizId:%s", sidPart)
			} else {
				avid = fmt.Sprintf("listBizId:%s", sidPart)
			}

		case strings.Contains(input, "/space.bilibili.com/") && strings.Contains(input, "/favlist"):
			match := uidRegex.FindStringSubmatch(input)
			mid := ""
			if len(match) > 1 {
				mid = match[1]
			}
			fid := getQueryString("fid", input)
			avid = fmt.Sprintf("favId:%s:%s", fid, mid)

		case strings.Contains(input, "/space.bilibili.com/"):
			match := uidRegex.FindStringSubmatch(input)
			if len(match) > 1 {
				avid = fmt.Sprintf("mid:%s", match[1])
			}

		case strings.Contains(input, "ep_id="):
			epID := getQueryString("ep_id", input)
			avid = fmt.Sprintf("ep:%s", epID)

		case globalEpRegex.MatchString(input):
			match := globalEpRegex.FindStringSubmatch(input)
			if len(match) > 1 {
				avid = fmt.Sprintf("ep:%s", match[1])
			}

		case bangumiMdRegex.MatchString(input):
			match := bangumiMdRegex.FindStringSubmatch(input)
			if len(match) > 1 {
				mdID := match[1]
				epID, err := getEpidByMD(ctx, client, mdID)
				if err != nil {
					return "", fmt.Errorf("get md review: %w", err)
				}
				avid = fmt.Sprintf("ep:%s", epID)
			}

		default:
			web, err := getWebSource(ctx, client, input)
			if err != nil {
				return "", fmt.Errorf("fetch web source: %w", err)
			}
			match := stateRegex.FindStringSubmatch(web)
			if len(match) > 1 {
				jsonStr := match[1]
				var state struct {
					EpList []struct {
						ID int `json:"id"`
					} `json:"epList"`
				}
				if err := json.Unmarshal([]byte(jsonStr), &state); err != nil {
					return "", fmt.Errorf("failed to parse page state: %w", err)
				}
				if len(state.EpList) > 0 {
					avid = fmt.Sprintf("ep:%d", state.EpList[0].ID)
				} else {
					return "", fmt.Errorf("no ep found in page state")
				}
			} else {
				return "", fmt.Errorf("unrecognized URL format")
			}
		}

	} else if strings.HasPrefix(strings.ToLower(input), "bv") {
		aid, err := bvconv.Decode(input)
		if err != nil {
			return "", fmt.Errorf("failed to decode BV: %w", err)
		}
		avid = strconv.FormatInt(aid, 10)

	} else if strings.HasPrefix(strings.ToLower(input), "av") {
		avid = strings.ToLower(input)[2:]

	} else if strings.HasPrefix(input, "cheese/") {
		epID := ""
		switch {
		case strings.Contains(input, "/ep"):
			match := epRegex.FindStringSubmatch(input)
			if len(match) > 1 {
				epID = match[1]
			}
		case strings.Contains(input, "/ss"):
			match := ssRegex.FindStringSubmatch(input)
			if len(match) > 1 {
				var err error
				epID, err = getEpidByCheeseSSID(ctx, client, match[1])
				if err != nil {
					return "", fmt.Errorf("get cheese ssid: %w", err)
				}
			}
		}
		avid = fmt.Sprintf("cheese:%s", epID)

	} else if strings.HasPrefix(input, "ep") {
		avid = fmt.Sprintf("ep:%s", input[2:])

	} else if strings.HasPrefix(input, "ss") {
		epID, err := getEpidByBangumiSSID(ctx, client, input[2:])
		if err != nil {
			return "", fmt.Errorf("get bangumi ssid: %w", err)
		}
		avid = fmt.Sprintf("ep:%s", epID)

	} else if strings.HasPrefix(input, "md") {
		match := mdRegex.FindStringSubmatch(input)
		if len(match) > 1 {
			epID, err := getEpidByMD(ctx, client, match[1])
			if err != nil {
				return "", fmt.Errorf("get md review: %w", err)
			}
			avid = fmt.Sprintf("ep:%s", epID)
		}

	} else {
		return "", fmt.Errorf("invalid input")
	}

	return fixAvid(ctx, client, avid)
}

func fixAvid(ctx context.Context, client httpclient.Client, avid string) (string, error) {
	if !isAllDigits(avid) {
		return avid, nil
	}
	api := fmt.Sprintf("https://www.bilibili.com/video/av%s/", avid)
	location, err := client.GetRedirectLocation(ctx, api)
	if err != nil {
		return avid, nil // return original on error, matching C# behavior
	}
	if strings.Contains(location, "/ep") {
		match := epRegex.FindStringSubmatch(location)
		if len(match) > 1 {
			return fmt.Sprintf("ep:%s", match[1]), nil
		}
	}
	return avid, nil
}

func getEpidByCheeseSSID(ctx context.Context, client httpclient.Client, ssid string) (string, error) {
	api := fmt.Sprintf("https://api.bilibili.com/pugv/view/web/season?season_id=%s", ssid)
	body, err := fetchBody(ctx, client, api)
	if err != nil {
		return "", fmt.Errorf("fetch cheese season: %w", err)
	}
	var result struct {
		Data struct {
			Episodes []struct {
				ID int `json:"id"`
			} `json:"episodes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse cheese season: %w", err)
	}
	if len(result.Data.Episodes) == 0 {
		return "", fmt.Errorf("no episodes found for cheese ss%s", ssid)
	}
	return strconv.Itoa(result.Data.Episodes[0].ID), nil
}

func getEpidByBangumiSSID(ctx context.Context, client httpclient.Client, ssID string) (string, error) {
	api := fmt.Sprintf("https://%s/pgc/view/web/season?season_id=%s", defaultEpHost, ssID)
	body, err := fetchBody(ctx, client, api)
	if err != nil {
		return "", fmt.Errorf("fetch bangumi season: %w", err)
	}
	var result struct {
		Result struct {
			Episodes []struct {
				ID int `json:"id"`
			} `json:"episodes"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse bangumi season: %w", err)
	}
	if len(result.Result.Episodes) == 0 {
		return "", fmt.Errorf("no episodes found for bangumi ss%s", ssID)
	}
	return strconv.Itoa(result.Result.Episodes[0].ID), nil
}

func getEpidByMD(ctx context.Context, client httpclient.Client, mdID string) (string, error) {
	api := fmt.Sprintf("https://api.bilibili.com/pgc/review/user?media_id=%s", mdID)
	body, err := fetchBody(ctx, client, api)
	if err != nil {
		return "", fmt.Errorf("fetch md review: %w", err)
	}
	var result struct {
		Result struct {
			Media struct {
				NewEp struct {
					ID int `json:"id"`
				} `json:"new_ep"`
			} `json:"media"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse md review: %w", err)
	}
	return strconv.Itoa(result.Result.Media.NewEp.ID), nil
}

func fetchBody(ctx context.Context, client httpclient.Client, url string) ([]byte, error) {
	resp, err := client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}
	return body, nil
}

func getWebSource(ctx context.Context, client httpclient.Client, url string) (string, error) {
	body, err := fetchBody(ctx, client, url)
	if err != nil {
		return "", fmt.Errorf("fetch web source: %w", err)
	}
	return string(body), nil
}

func getQueryString(name, urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	return u.Query().Get(name)
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}
