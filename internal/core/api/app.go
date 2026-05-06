//go:generate protoc --go_out=. --go_opt=paths=source_relative proto/Payload/playviewreq.proto proto/Response/playviewreply.proto proto/Header/*.proto

package api

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"github.com/nilaonai/bbdown-go/internal/core/api/proto/Response"
)

// PackMessage gzip-compresses the input and prepends a 5-byte gRPC header:
// 1-byte flag (1 = compressed) + 4-byte big-endian length of compressed payload.
func PackMessage(input []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(input); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	comp := buf.Bytes()

	out := make([]byte, 5+len(comp))
	out[0] = 1
	binary.BigEndian.PutUint32(out[1:5], uint32(len(comp)))
	copy(out[5:], comp)
	return out, nil
}

// ReadMessage reads a 5-byte gRPC header from data, decompresses if the flag
// indicates gzip, and returns the payload.
func ReadMessage(data []byte) ([]byte, error) {
	if len(data) < 5 {
		return nil, fmt.Errorf("data too short: expected at least 5 bytes, got %d", len(data))
	}
	flag := data[0]
	size := int(binary.BigEndian.Uint32(data[1:5]))
	if len(data) < 5+size {
		return nil, fmt.Errorf("data too short: expected %d bytes, got %d", 5+size, len(data))
	}
	payload := data[5 : 5+size]
	if flag == 1 {
		zr, err := gzip.NewReader(bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		defer zr.Close()
		return io.ReadAll(zr)
	}
	return payload, nil
}

// BuildGRPCHeaders builds a simplified set of gRPC request headers.
func BuildGRPCHeaders(token string) map[string]string {
	return map[string]string{
		"Host":                  "grpc.biliapi.net",
		"user-agent":            "Dalvik/2.1.0 (Linux; U; Android 11; M2012K11AC Build/RKQ1.200826.002) 7.32.0 os/android model/M2012K11AC mobi_app/android build/7320200 channel/xiaomi_cn_tv.danmaku.bili_zm20200902 innerVer/7320200 osVer/11 network/2 grpc-java-cronet/1.36.1",
		"te":                    "trailers",
		"authorization":         "identify_v1 " + token,
		"x-bili-metadata-bin":   "",
		"x-bili-device-bin":     "",
		"grpc-encoding":         "gzip",
		"grpc-accept-encoding":  "identity,gzip",
	}
}

// dashVideoItem represents a video stream in the DASH JSON output.
type dashVideoItem struct {
	ID        uint32   `json:"id"`
	BaseURL   string   `json:"base_url"`
	BackupURL []string `json:"backup_url"`
	Bandwidth uint32   `json:"bandwidth"`
	CodecID   uint32   `json:"codecid"`
}

// dashAudioItem represents an audio stream in the DASH JSON output.
type dashAudioItem struct {
	ID        uint32   `json:"id"`
	BaseURL   string   `json:"base_url"`
	BackupURL []string `json:"backup_url"`
	Bandwidth uint32   `json:"bandwidth"`
	Codecs    string   `json:"codecs"`
}

// dashClip represents clip information in the DASH JSON output.
type dashClip struct {
	Start     int32  `json:"start"`
	End       int32  `json:"end"`
	ToastText string `json:"toastText"`
}

// dashInfo holds video and audio arrays.
type dashInfo struct {
	Video []interface{} `json:"video"`
	Audio []interface{} `json:"audio"`
}

// dashData is the nested "data" object.
type dashData struct {
	TimeLength   uint64        `json:"timelength"`
	Dash         dashInfo      `json:"dash"`
	ClipInfoList []interface{} `json:"clip_info_list"`
}

// audioMaterial represents dubbing audio material.
type audioMaterial struct {
	AudioID    string        `json:"audio_id"`
	Title      string        `json:"title"`
	PersonName string        `json:"person_name"`
	Audio      []interface{} `json:"audio"`
}

// dubbingInfo represents dubbing information.
type dubbingInfo struct {
	BackgroundAudio []interface{} `json:"background_audio"`
	RoleAudioList   []interface{} `json:"role_audio_list"`
}

// dashJSON is the top-level JSON structure.
type dashJSON struct {
	Code        int         `json:"code"`
	Message     string      `json:"message"`
	TTL         int         `json:"ttl"`
	Data        dashData    `json:"data"`
	DubbingInfo dubbingInfo `json:"dubbing_info"`
}

// ConvertToDashJson converts a PlayViewReply protobuf to a JSON string matching
// the Bilibili web API DASH format.
func ConvertToDashJson(resp *Response.PlayViewReply) (string, error) {
	if resp == nil {
		return "", fmt.Errorf("nil PlayViewReply")
	}

	result := dashJSON{
		Code:    0,
		Message: "0",
		TTL:     1,
		Data: dashData{
			Dash: dashInfo{
				Video: make([]interface{}, 0),
				Audio: make([]interface{}, 0),
			},
			ClipInfoList: make([]interface{}, 0),
		},
		DubbingInfo: dubbingInfo{
			BackgroundAudio: make([]interface{}, 0),
			RoleAudioList:   make([]interface{}, 0),
		},
	}

	var timelength uint64
	if resp.VideoInfo != nil && resp.VideoInfo.Timelength != nil {
		timelength = *resp.VideoInfo.Timelength
	}
	result.Data.TimeLength = timelength

	// Video streams
	if resp.VideoInfo != nil && resp.VideoInfo.StreamList != nil {
		for _, item := range resp.VideoInfo.StreamList {
			if item == nil || item.DashVideo == nil {
				continue
			}
			dv := item.DashVideo
			var bandwidth uint32
			if timelength > 0 && dv.Size != nil {
				bandwidth = uint32(*dv.Size * 8 / (timelength / 1000))
			}
			videoItem := dashVideoItem{
				ID:        safeUint32(item.StreamInfo.Quality),
				BaseURL:   safeString(dv.BaseUrl),
				BackupURL: dv.BackupUrl,
				Bandwidth: bandwidth,
				CodecID:   safeUint32(dv.Codecid),
			}
			result.Data.Dash.Video = append(result.Data.Dash.Video, videoItem)
		}
	}

	// Audio streams (standard DASH audio)
	if resp.VideoInfo != nil && resp.VideoInfo.DashAudio != nil {
		for _, item := range resp.VideoInfo.DashAudio {
			if item == nil {
				continue
			}
			audioItem := dashAudioItem{
				ID:        safeUint32(item.Id),
				BaseURL:   safeString(item.BaseUrl),
				BackupURL: item.BackupUrl,
				Bandwidth: safeUint32(item.Bandwidth),
				Codecs:    "M4A",
			}
			result.Data.Dash.Audio = append(result.Data.Dash.Audio, audioItem)
		}
	}

	// FLAC audio
	if resp.VideoInfo != nil && resp.VideoInfo.Flac != nil && resp.VideoInfo.Flac.Audio != nil {
		flac := resp.VideoInfo.Flac.Audio
		result.Data.Dash.Audio = append(result.Data.Dash.Audio, dashAudioItem{
			ID:        safeUint32(flac.Id),
			BaseURL:   safeString(flac.BaseUrl),
			BackupURL: flac.BackupUrl,
			Bandwidth: safeUint32(flac.Bandwidth),
			Codecs:    "FLAC",
		})
	}

	// Dolby audio
	if resp.VideoInfo != nil && resp.VideoInfo.Dolby != nil && resp.VideoInfo.Dolby.Audio != nil {
		dolby := resp.VideoInfo.Dolby.Audio
		result.Data.Dash.Audio = append(result.Data.Dash.Audio, dashAudioItem{
			ID:        safeUint32(dolby.Id),
			BaseURL:   safeString(dolby.BaseUrl),
			BackupURL: dolby.BackupUrl,
			Bandwidth: safeUint32(dolby.Bandwidth),
			Codecs:    "E-AC-3",
		})
	}

	// Clip info
	if resp.Business != nil && resp.Business.ClipInfo != nil {
		for _, clip := range resp.Business.ClipInfo {
			if clip == nil {
				continue
			}
			result.Data.ClipInfoList = append(result.Data.ClipInfoList, dashClip{
				Start:     safeInt32(clip.Start),
				End:       safeInt32(clip.End),
				ToastText: safeString(clip.ToastText),
			})
		}
	}

	// Dubbing info
	if resp.PlayExtInfo != nil && resp.PlayExtInfo.PlayDubbingInfo != nil {
		dub := resp.PlayExtInfo.PlayDubbingInfo
		if dub.BackgroundAudio != nil {
			bg := dub.BackgroundAudio
			for _, item := range bg.Audio {
				if item == nil {
					continue
				}
				result.DubbingInfo.BackgroundAudio = append(result.DubbingInfo.BackgroundAudio, dashAudioItem{
					ID:        safeUint32(item.Id),
					BaseURL:   safeString(item.BaseUrl),
					BackupURL: item.BackupUrl,
					Bandwidth: safeUint32(item.Bandwidth),
					Codecs:    "M4A",
				})
			}
		}

		for _, role := range dub.RoleAudioList {
			if role == nil {
				continue
			}
			for _, material := range role.AudioMaterialList {
				if material == nil {
					continue
				}
				var roleAudios []interface{}
				for _, item := range material.Audio {
					if item == nil {
						continue
					}
					roleAudios = append(roleAudios, dashAudioItem{
						ID:        safeUint32(item.Id),
						BaseURL:   safeString(item.BaseUrl),
						BackupURL: item.BackupUrl,
						Bandwidth: safeUint32(item.Bandwidth),
						Codecs:    "M4A",
					})
				}
				title := safeString(material.Title)
				if title == "" {
					title = safeString(material.AudioId)
				}
				personName := safeString(material.PersonName)
				if personName == "" {
					personName = safeString(material.Edition)
				}
				result.DubbingInfo.RoleAudioList = append(result.DubbingInfo.RoleAudioList, audioMaterial{
					AudioID:    safeString(material.AudioId),
					Title:      title,
					PersonName: personName,
					Audio:      roleAudios,
				})
			}
		}
	}

	b, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func safeUint32(u *uint32) uint32 {
	if u == nil {
		return 0
	}
	return *u
}

func safeInt32(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}
