package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/nilaonai/bbdown-go/internal/config"
	"github.com/nilaonai/bbdown-go/internal/core/entity"
)

var baseUrlRegex = regexp.MustCompile(`http.*:\d+`)

func getVideoCodec(code string) string {
	switch code {
	case "13":
		return "AV1"
	case "12":
		return "HEVC"
	case "7":
		return "AVC"
	default:
		return "UNKNOWN"
	}
}

func getAudioCodec(codecs string) string {
	switch codecs {
	case "mp4a.40.2", "mp4a.40.5":
		return "M4A"
	case "ec-3":
		return "E-AC-3"
	case "fLaC":
		return "FLAC"
	default:
		return codecs
	}
}

func rawToString(r json.RawMessage) string {
	if len(r) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(r, &s); err == nil {
		return s
	}
	var n json.Number
	if err := json.Unmarshal(r, &n); err == nil {
		return string(n)
	}
	return ""
}

func rawToInt(r json.RawMessage) int {
	if len(r) == 0 {
		return 0
	}
	var n int
	if err := json.Unmarshal(r, &n); err == nil {
		return n
	}
	var num json.Number
	if err := json.Unmarshal(r, &num); err == nil {
		if v, err := num.Int64(); err == nil {
			return int(v)
		}
	}
	var s string
	if err := json.Unmarshal(r, &s); err == nil {
		if v, err := strconv.Atoi(s); err == nil {
			return v
		}
	}
	return 0
}

func rawToInt64(r json.RawMessage) int64 {
	if len(r) == 0 {
		return 0
	}
	var n int64
	if err := json.Unmarshal(r, &n); err == nil {
		return n
	}
	var num json.Number
	if err := json.Unmarshal(r, &num); err == nil {
		if v, err := num.Int64(); err == nil {
			return v
		}
	}
	var s string
	if err := json.Unmarshal(r, &s); err == nil {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			return v
		}
	}
	return 0
}

func rawToFloat64(r json.RawMessage) float64 {
	if len(r) == 0 {
		return 0
	}
	var f float64
	if err := json.Unmarshal(r, &f); err == nil {
		return f
	}
	var num json.Number
	if err := json.Unmarshal(r, &num); err == nil {
		if v, err := num.Float64(); err == nil {
			return v
		}
	}
	var s string
	if err := json.Unmarshal(r, &s); err == nil {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			return v
		}
	}
	return 0
}

func getUrlList(node map[string]json.RawMessage) []string {
	var urls []string
	if v, ok := node["base_url"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err == nil && s != "" {
			urls = append(urls, s)
		}
	}
	if v, ok := node["backup_url"]; ok {
		var arr []string
		if err := json.Unmarshal(v, &arr); err == nil {
			urls = append(urls, arr...)
		}
	}
	return urls
}

func selectBaseUrl(urls []string) string {
	for _, u := range urls {
		if !baseUrlRegex.MatchString(u) {
			return u
		}
	}
	if len(urls) > 0 {
		return urls[0]
	}
	return ""
}

// findAidCid tries to extract aid and cid from the JSON response.
func findAidCid(root map[string]json.RawMessage) (string, string) {
	if aid := rawToString(root["aid"]); aid != "" {
		return aid, rawToString(root["cid"])
	}
	if avid := rawToString(root["avid"]); avid != "" {
		return avid, rawToString(root["cid"])
	}
	if dataRaw, ok := root["data"]; ok {
		var data map[string]json.RawMessage
		if err := json.Unmarshal(dataRaw, &data); err == nil {
			if aid := rawToString(data["aid"]); aid != "" {
				return aid, rawToString(data["cid"])
			}
			if avid := rawToString(data["avid"]); avid != "" {
				return avid, rawToString(data["cid"])
			}
		}
	}
	if resultRaw, ok := root["result"]; ok {
		var result map[string]json.RawMessage
		if err := json.Unmarshal(resultRaw, &result); err == nil {
			if aid := rawToString(result["aid"]); aid != "" {
				return aid, rawToString(result["cid"])
			}
			if avid := rawToString(result["avid"]); avid != "" {
				return avid, rawToString(result["cid"])
			}
		}
	}
	return "", ""
}

// ParseDashTracks parses DASH video and audio tracks from a Bilibili playurl JSON response.
// It returns videoTracks, audioTracks, backgroundAudioTracks, roleAudioList, and any error.
func ParseDashTracks(jsonStr string, tvApi, appApi, bangumi bool) ([]entity.Video, []entity.Audio, []entity.Audio, []entity.AudioMaterialInfo, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &root); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parse dash root: %w", err)
	}

	var dashObj map[string]json.RawMessage
	var dataObj map[string]json.RawMessage
	var pDur int

	// Try "data" first
	if dataRaw, ok := root["data"]; ok {
		if err := json.Unmarshal(dataRaw, &dataObj); err == nil {
			if dashRaw, ok := dataObj["dash"]; ok {
				_ = json.Unmarshal(dashRaw, &dashObj)
			}
			if pDur == 0 {
				pDur = rawToInt(dataObj["duration"])
			}
			if pDur == 0 {
				pDur = rawToInt(dataObj["timelength"]) / 1000
			}
		}
	}

	// Try "result"
	if dashObj == nil {
		if resultRaw, ok := root["result"]; ok {
			var resultObj map[string]json.RawMessage
			if err := json.Unmarshal(resultRaw, &resultObj); err == nil {
				if dashRaw, ok := resultObj["dash"]; ok {
					_ = json.Unmarshal(dashRaw, &dashObj)
				}
				// Try result.video_info
				if dashObj == nil {
					if viRaw, ok := resultObj["video_info"]; ok {
						var viObj map[string]json.RawMessage
						if err := json.Unmarshal(viRaw, &viObj); err == nil {
							if dashRaw, ok := viObj["dash"]; ok {
								_ = json.Unmarshal(dashRaw, &dashObj)
							}
						}
					}
				}
				if pDur == 0 {
					pDur = rawToInt(resultObj["duration"])
				}
				if pDur == 0 {
					pDur = rawToInt(resultObj["timelength"]) / 1000
				}
			}
		}
	}

	// Try root-level dash
	if dashObj == nil {
		if dashRaw, ok := root["dash"]; ok {
			_ = json.Unmarshal(dashRaw, &dashObj)
		}
	}

	// Duration from dash itself (lowest priority)
	if pDur == 0 && dashObj != nil {
		pDur = rawToInt(dashObj["duration"])
	}

	// Try root-level timelength
	if pDur == 0 {
		pDur = rawToInt(root["timelength"]) / 1000
	}

	aid, cid := findAidCid(root)

	var videoTracks []entity.Video
	var audioTracks []entity.Audio
	var backgroundAudioTracks []entity.Audio
	var roleAudioList []entity.AudioMaterialInfo

	if dashObj == nil {
		return videoTracks, audioTracks, backgroundAudioTracks, roleAudioList, nil
	}

	// Parse video tracks
	if videoRaw, ok := dashObj["video"]; ok {
		var videoArr []map[string]json.RawMessage
		if err := json.Unmarshal(videoRaw, &videoArr); err == nil && videoArr != nil {
			for _, node := range videoArr {
				urls := getUrlList(node)
				videoID := rawToString(node["id"])
				v := entity.Video{
					Dur:      pDur,
					ID:       videoID,
					Dfn:      config.Qualities[videoID],
					Bandwith: rawToInt64(node["bandwidth"]) / 1000,
					BaseUrl:  selectBaseUrl(urls),
					Codecs:   getVideoCodec(rawToString(node["codecid"])),
					Size:     rawToFloat64(node["size"]),
				}
				if !tvApi && !appApi {
					v.Res = rawToString(node["width"]) + "x" + rawToString(node["height"])
					v.Fps = rawToString(node["frame_rate"])
				}

				exists := false
				for _, existing := range videoTracks {
					if existing.Equal(v) {
						exists = true
						break
					}
				}
				if !exists {
					videoTracks = append(videoTracks, v)
				}
			}
		}
	}

	// Parse standard audio tracks
	var hasAudio bool
	if audioRaw, ok := dashObj["audio"]; ok {
		var audioArr []map[string]json.RawMessage
		if err := json.Unmarshal(audioRaw, &audioArr); err == nil && audioArr != nil {
			hasAudio = true
			for _, node := range audioArr {
				urls := getUrlList(node)
				audioID := rawToString(node["id"])
				a := entity.Audio{
					ID:       audioID,
					Dfn:      audioID,
					Dur:      pDur,
					Bandwith: rawToInt64(node["bandwidth"]) / 1000,
					BaseUrl:  selectBaseUrl(urls),
					Codecs:   getAudioCodec(rawToString(node["codecs"])),
				}

				exists := false
				for _, existing := range audioTracks {
					if existing.Equal(a) {
						exists = true
						break
					}
				}
				if !exists {
					audioTracks = append(audioTracks, a)
				}
			}
		}
	}

	// Dolby audio
	if hasAudio && !tvApi {
		if dolbyRaw, ok := dashObj["dolby"]; ok {
			var dolbyObj map[string]json.RawMessage
			if err := json.Unmarshal(dolbyRaw, &dolbyObj); err == nil {
				if dbRaw, ok := dolbyObj["audio"]; ok {
					var dbArr []map[string]json.RawMessage
					if err := json.Unmarshal(dbRaw, &dbArr); err == nil {
						for _, node := range dbArr {
							urls := getUrlList(node)
							audioID := rawToString(node["id"])
							a := entity.Audio{
								ID:       audioID,
								Dfn:      audioID,
								Dur:      pDur,
								Bandwith: rawToInt64(node["bandwidth"]) / 1000,
								BaseUrl:  selectBaseUrl(urls),
								Codecs:   getAudioCodec(rawToString(node["codecs"])),
							}

							exists := false
							for _, existing := range audioTracks {
								if existing.Equal(a) {
									exists = true
									break
								}
							}
							if !exists {
								audioTracks = append(audioTracks, a)
							}
						}
					}
				}
			}
		}
	}

	// Hi-Res FLAC
	if hasAudio && !tvApi {
		if flacRaw, ok := dashObj["flac"]; ok {
			var flacObj map[string]json.RawMessage
			if err := json.Unmarshal(flacRaw, &flacObj); err == nil {
				if hiResRaw, ok := flacObj["audio"]; ok {
					var hiResNode map[string]json.RawMessage
					if err := json.Unmarshal(hiResRaw, &hiResNode); err == nil && hiResNode != nil {
						urls := getUrlList(hiResNode)
						audioID := rawToString(hiResNode["id"])
						a := entity.Audio{
							ID:       audioID,
							Dfn:      audioID,
							Dur:      pDur,
							Bandwith: rawToInt64(hiResNode["bandwidth"]) / 1000,
							BaseUrl:  selectBaseUrl(urls),
							Codecs:   getAudioCodec(rawToString(hiResNode["codecs"])),
						}

						exists := false
						for _, existing := range audioTracks {
							if existing.Equal(a) {
								exists = true
								break
							}
						}
						if !exists {
							audioTracks = append(audioTracks, a)
						}
					}
				}
			}
		}
	}

	// Background audio and role audio (dubbing)
	if appApi && bangumi && dataObj != nil {
		if dubbingRaw, ok := dataObj["dubbing_info"]; ok {
			var dubbingObj map[string]json.RawMessage
			if err := json.Unmarshal(dubbingRaw, &dubbingObj); err == nil {
				// Background audio
				if bgRaw, ok := dubbingObj["background_audio"]; ok {
					var bgArr []map[string]json.RawMessage
					if err := json.Unmarshal(bgRaw, &bgArr); err == nil {
						for _, node := range bgArr {
							urls := getUrlList(node)
							audioID := rawToString(node["id"])
							a := entity.Audio{
								ID:       audioID,
								Dfn:      audioID,
								Dur:      pDur,
								Bandwith: rawToInt64(node["bandwidth"]) / 1000,
								BaseUrl:  selectBaseUrl(urls),
								Codecs:   rawToString(node["codecs"]),
							}
							backgroundAudioTracks = append(backgroundAudioTracks, a)
						}
					}
				}

				// Role audio
				if roleRaw, ok := dubbingObj["role_audio_list"]; ok {
					var roleArr []map[string]json.RawMessage
					if err := json.Unmarshal(roleRaw, &roleArr); err == nil {
						for _, role := range roleArr {
							var roleAudioTracks []entity.Audio
							if audioRaw2, ok := role["audio"]; ok {
								var audioArr2 []map[string]json.RawMessage
								if err := json.Unmarshal(audioRaw2, &audioArr2); err == nil {
									for _, node := range audioArr2 {
										urls := getUrlList(node)
										audioID := rawToString(node["id"])
										a := entity.Audio{
											ID:       audioID,
											Dfn:      audioID,
											Dur:      pDur,
											Bandwith: rawToInt64(node["bandwidth"]) / 1000,
											BaseUrl:  selectBaseUrl(urls),
											Codecs:   rawToString(node["codecs"]),
										}
										roleAudioTracks = append(roleAudioTracks, a)
									}
								}
							}

							audioID := rawToString(role["audio_id"])
							path := ""
							if aid != "" && cid != "" {
								path = aid + "/" + aid + "." + cid + "." + audioID + ".m4a"
							}

							roleAudioList = append(roleAudioList, entity.AudioMaterialInfo{
								Title:      rawToString(role["title"]),
								PersonName: rawToString(role["person_name"]),
								Path:       path,
								Audio:      roleAudioTracks,
							})
						}
					}
				}
			}
		}
	}

	return videoTracks, audioTracks, backgroundAudioTracks, roleAudioList, nil
}
