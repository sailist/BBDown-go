package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/sailist/BBDown-go/internal/config"
	"github.com/sailist/BBDown-go/internal/core/entity"
	"github.com/sailist/BBDown-go/pkg/httpclient"
)

var playInfoRegex = regexp.MustCompile(`window\.__playinfo__=([\s\S]*?)<\/script>`)

// ExtractOptions controls which API and parameters to use when extracting tracks.
type ExtractOptions struct {
	TvApi    bool
	IntlApi  bool
	AppApi   bool
	Encoding string
	Qn       string
}

// ExtractTracks fetches the playurl JSON, parses video/audio tracks, and handles
// intl re-fetch, max-QN re-fetch for DASH/FLV, VIP fallback, and bangumi clip info.
func ExtractTracks(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts ExtractOptions) (*entity.ParsedResult, error) {
	bangumi := strings.HasPrefix(aidOri, "ep:")

	webJson, err := getPlayJson(ctx, client, cfg, logger, aidOri, aid, cid, epid, opts)
	if err != nil {
		return nil, fmt.Errorf("get play json: %w", err)
	}

	if cfg.DebugLog {
		logger.Debug("play json response", slog.String("json", webJson))
	}

	// intl API: stream_list path
	if opts.IntlApi && strings.Contains(webJson, `"stream_list"`) {
		if _, err := parseIntlTracks(webJson); err != nil {
			return nil, fmt.Errorf("parse intl tracks: %w", err)
		}

		// Re-fetch with code="1" and overwrite
		apiURL := BuildIntlPlayurlAPI(aid, cid, epid, opts.Qn, "1", cfg)
		resp, err := client.Get(ctx, apiURL)
		if err != nil {
			return nil, fmt.Errorf("intl re-fetch code=1: %w", err)
		}
		webJson, err = readBody(resp)
		if err != nil {
			return nil, fmt.Errorf("read intl re-fetch body: %w", err)
		}
		result, err := parseIntlTracks(webJson)
		if err != nil {
			return nil, fmt.Errorf("parse intl tracks code=1: %w", err)
		}
		result.WebJsonString = webJson
		return result, nil
	}

	result := &entity.ParsedResult{WebJsonString: webJson}

	switch {
	case strings.Contains(webJson, `"dash":{`):
		videos, audios, bgAudios, roleAudios, err := ParseDashTracks(webJson, opts.TvApi, opts.AppApi, bangumi)
		if err != nil {
			return nil, fmt.Errorf("parse dash tracks: %w", err)
		}
		result.VideoTracks = videos
		result.AudioTracks = audios
		result.BackgroundAudioTracks = bgAudios
		result.RoleAudioList = roleAudios

		// 免二压视频: non-appApi DASH needs a second request with max QN
		if !opts.AppApi {
			maxOpts := opts
			maxOpts.Qn = getMaxQn()
			webJson, err = getPlayJson(ctx, client, cfg, logger, aidOri, aid, cid, epid, maxOpts)
			if err != nil {
				return nil, fmt.Errorf("dash re-fetch max qn: %w", err)
			}
			videos, audios, bgAudios, roleAudios, err = ParseDashTracks(webJson, opts.TvApi, opts.AppApi, bangumi)
			if err != nil {
				return nil, fmt.Errorf("parse dash tracks max qn: %w", err)
			}
			result = &entity.ParsedResult{
				WebJsonString:         webJson,
				VideoTracks:           videos,
				AudioTracks:           audios,
				BackgroundAudioTracks: bgAudios,
				RoleAudioList:         roleAudios,
			}
		}

	case strings.Contains(webJson, `"durl":[`):
		// FLV always uses max QN
		maxOpts := opts
		maxOpts.Qn = getMaxQn()
		webJson, err = getPlayJson(ctx, client, cfg, logger, aidOri, aid, cid, epid, maxOpts)
		if err != nil {
			return nil, fmt.Errorf("flv fetch max qn: %w", err)
		}
		result, err = ParseFlvTracks(webJson)
		if err != nil {
			return nil, fmt.Errorf("parse flv tracks: %w", err)
		}
		result.WebJsonString = webJson
	}

	// Bangumi clip info
	if bangumi {
		rootJSON, err := extractRootJSON(webJson)
		if err == nil && rootJSON != "" {
			clipJSON := extractClipInfoListJSON(rootJSON)
			if clipJSON != "" {
				points, err := ParseClipInfoList(clipJSON)
				if err == nil {
					result.ExtraPoints = points
				}
			}
		}
	}

	return result, nil
}

// getPlayJson builds the correct API URL and fetches the playurl JSON string.
func getPlayJson(ctx context.Context, client httpclient.Client, cfg *config.Config, logger *slog.Logger, aidOri, aid, cid, epid string, opts ExtractOptions) (string, error) {
	// AppApi falls back to web API until gRPC helper is implemented in commit 19.

	bangumi := strings.HasPrefix(aidOri, "ep:")
	cheese := strings.HasPrefix(aidOri, "cheese:")

	var apiURL string
	switch {
	case opts.IntlApi:
		apiURL = BuildIntlPlayurlAPI(aid, cid, epid, opts.Qn, "0", cfg)
	case opts.TvApi:
		apiURL = BuildTVPlayurlAPI(aid, cid, epid, opts.Qn, cfg, bangumi || cheese)
	default:
		apiURL = BuildWebPlayurlAPI(aid, cid, epid, opts.Qn, cfg, cfg.WBI, bangumi || cheese)
	}

	if cheese {
		apiURL = strings.Replace(apiURL, "/pgc/", "/pugv/", 1)
	}

	resp, err := client.Get(ctx, apiURL)
	if err != nil {
		return "", fmt.Errorf("http get: %w", err)
	}
	webJson, err := readBody(resp)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	// VIP-only fallback: fetch from web page source
	if strings.Contains(webJson, "大会员专享限制") {
		logger.Info("此视频需要大会员，尝试从网页源码解析")
		webURL := "https://www.bilibili.com/bangumi/play/ep" + epid
		resp, err = client.Get(ctx, webURL)
		if err != nil {
			return "", fmt.Errorf("fetch bangumi page: %w", err)
		}
		webSource, err := readBody(resp)
		if err != nil {
			return "", fmt.Errorf("read bangumi page: %w", err)
		}
		matches := playInfoRegex.FindStringSubmatch(webSource)
		if len(matches) > 1 {
			webJson = matches[1]
		}
	}

	return webJson, nil
}

func readBody(resp *http.Response) (string, error) {
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}
	return string(b), nil
}

func getMaxQn() string {
	max := 0
	for k := range config.Qualities {
		if v, err := strconv.Atoi(k); err == nil && v > max {
			max = v
		}
	}
	if max == 0 {
		return "127"
	}
	return strconv.Itoa(max)
}

func parseIntlTracks(jsonStr string) (*entity.ParsedResult, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &root); err != nil {
		return nil, fmt.Errorf("parse intl tracks root: %w", err)
	}

	var dataObj map[string]json.RawMessage
	if dataRaw, ok := root["data"]; ok {
		_ = json.Unmarshal(dataRaw, &dataObj)
	}
	if dataObj == nil {
		return &entity.ParsedResult{}, nil
	}

	var videoInfo map[string]json.RawMessage
	if viRaw, ok := dataObj["video_info"]; ok {
		_ = json.Unmarshal(viRaw, &videoInfo)
	}
	if videoInfo == nil {
		return &entity.ParsedResult{}, nil
	}

	pDur := rawToInt(videoInfo["timelength"]) / 1000
	result := &entity.ParsedResult{}

	// stream_list -> video tracks
	if streamListRaw, ok := videoInfo["stream_list"]; ok {
		var streams []map[string]json.RawMessage
		if err := json.Unmarshal(streamListRaw, &streams); err == nil {
			for _, stream := range streams {
				dashVideoRaw, ok := stream["dash_video"]
				if !ok {
					continue
				}
				var dashVideo map[string]json.RawMessage
				if err := json.Unmarshal(dashVideoRaw, &dashVideo); err != nil {
					continue
				}
				if rawToString(dashVideo["base_url"]) == "" {
					continue
				}
				urls := getUrlList(dashVideo)

				videoID := ""
				if streamInfoRaw, ok := stream["stream_info"]; ok {
					var streamInfo map[string]json.RawMessage
					if err := json.Unmarshal(streamInfoRaw, &streamInfo); err == nil {
						videoID = rawToString(streamInfo["quality"])
					}
				}

				v := entity.Video{
					Dur:      pDur,
					ID:       videoID,
					Dfn:      config.Qualities[videoID],
					Bandwith: rawToInt64(dashVideo["bandwidth"]) / 1000,
					BaseUrl:  selectBaseUrl(urls),
					Codecs:   getVideoCodec(rawToString(dashVideo["codecid"])),
					Size:     rawToFloat64(dashVideo["size"]),
				}

				exists := false
				for _, existing := range result.VideoTracks {
					if existing.Equal(v) {
						exists = true
						break
					}
				}
				if !exists {
					result.VideoTracks = append(result.VideoTracks, v)
				}
			}
		}
	}

	// dash_audio -> audio tracks
	if dashAudioRaw, ok := videoInfo["dash_audio"]; ok {
		var audioArr []map[string]json.RawMessage
		if err := json.Unmarshal(dashAudioRaw, &audioArr); err == nil {
			for _, node := range audioArr {
				urls := getUrlList(node)
				audioID := rawToString(node["id"])
				a := entity.Audio{
					ID:       audioID,
					Dfn:      audioID,
					Dur:      pDur,
					Bandwith: rawToInt64(node["bandwidth"]) / 1000,
					BaseUrl:  selectBaseUrl(urls),
					Codecs:   "M4A",
				}

				exists := false
				for _, existing := range result.AudioTracks {
					if existing.Equal(a) {
						exists = true
						break
					}
				}
				if !exists {
					result.AudioTracks = append(result.AudioTracks, a)
				}
			}
		}
	}

	return result, nil
}

func extractRootJSON(jsonStr string) (string, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &root); err != nil {
		return "", fmt.Errorf("extract root json: %w", err)
	}

	hasResult := strings.Contains(jsonStr, `"result":{`)
	hasVideoInfo := strings.Contains(jsonStr, `"video_info":{`)
	hasData := strings.Contains(jsonStr, `"data":{`)

	if hasResult {
		resultRaw, ok := root["result"]
		if !ok {
			return jsonStr, nil
		}
		if hasVideoInfo {
			var resultObj map[string]json.RawMessage
			if err := json.Unmarshal(resultRaw, &resultObj); err == nil {
				if viRaw, ok := resultObj["video_info"]; ok {
					return string(viRaw), nil
				}
			}
		}
		return string(resultRaw), nil
	}

	if hasData {
		if dataRaw, ok := root["data"]; ok {
			return string(dataRaw), nil
		}
	}

	return jsonStr, nil
}

func extractClipInfoListJSON(rootJSON string) string {
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(rootJSON), &root); err != nil {
		return ""
	}
	if clipsRaw, ok := root["clip_info_list"]; ok {
		wrapped, _ := json.Marshal(map[string]json.RawMessage{"clip_info_list": clipsRaw})
		return string(wrapped)
	}
	return ""
}
