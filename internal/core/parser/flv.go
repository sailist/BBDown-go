package parser

import (
	"encoding/json"
	"fmt"

	"github.com/sailist/BBDown-go/internal/config"
	"github.com/sailist/BBDown-go/internal/core/entity"
)

// ParseFlvTracks parses FLV stream tracks from a Bilibili playurl JSON response.
// It returns a ParsedResult containing video tracks, clip URLs, and available quality definitions.
func ParseFlvTracks(jsonStr string) (*entity.ParsedResult, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &root); err != nil {
		return nil, fmt.Errorf("parse flv root: %w", err)
	}

	result := &entity.ParsedResult{}

	// Try to locate the FLV data node under various root paths.
	// A valid node must contain "durl" to be considered the FLV data container.
	var flvNode map[string]json.RawMessage

	// Helper to check if a map has durl.
	hasDurl := func(m map[string]json.RawMessage) bool {
		_, ok := m["durl"]
		return ok
	}

	// Path 1: root -> data
	if dataRaw, ok := root["data"]; ok {
		var dataObj map[string]json.RawMessage
		if err := json.Unmarshal(dataRaw, &dataObj); err == nil && hasDurl(dataObj) {
			flvNode = dataObj
		}
	}

	// Path 2: root -> result
	if flvNode == nil {
		if resultRaw, ok := root["result"]; ok {
			var resultObj map[string]json.RawMessage
			if err := json.Unmarshal(resultRaw, &resultObj); err == nil && hasDurl(resultObj) {
				flvNode = resultObj
			}
		}
	}

	// Path 3: root -> result -> video_info
	if flvNode == nil {
		if resultRaw, ok := root["result"]; ok {
			var resultObj map[string]json.RawMessage
			if err := json.Unmarshal(resultRaw, &resultObj); err == nil {
				if viRaw, ok := resultObj["video_info"]; ok {
					var viObj map[string]json.RawMessage
					if err := json.Unmarshal(viRaw, &viObj); err == nil && hasDurl(viObj) {
						flvNode = viObj
					}
				}
			}
		}
	}

	// Path 4: root-level durl
	if flvNode == nil && hasDurl(root) {
		flvNode = root
	}

	if flvNode == nil {
		return result, nil
	}

	quality := rawToString(flvNode["quality"])
	videoCodecid := rawToString(flvNode["video_codecid"])

	// Parse durl segments.
	if durlRaw, ok := flvNode["durl"]; ok {
		var durlArr []map[string]json.RawMessage
		if err := json.Unmarshal(durlRaw, &durlArr); err == nil && durlArr != nil {
			var totalSize float64
			var totalLength float64

			for _, node := range durlArr {
				if url := rawToString(node["url"]); url != "" {
					result.Clips = append(result.Clips, url)
				}
				totalSize += rawToFloat64(node["size"])
				totalLength += rawToFloat64(node["length"])
			}

			v := entity.Video{
				ID:     quality,
				Dfn:    config.Qualities[quality],
				Codecs: getVideoCodec(videoCodecid),
				Dur:    int(totalLength) / 1000,
				Size:   totalSize,
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

	// Available qualities: TV mode uses qn_extras.
	if qnExtrasRaw, ok := flvNode["qn_extras"]; ok {
		var qnExtrasArr []map[string]json.RawMessage
		if err := json.Unmarshal(qnExtrasRaw, &qnExtrasArr); err == nil && qnExtrasArr != nil {
			for _, node := range qnExtrasArr {
				if qn := rawToString(node["qn"]); qn != "" {
					result.Dfns = append(result.Dfns, qn)
				}
			}
		}
	} else if acceptQualityRaw, ok := flvNode["accept_quality"]; ok {
		// Non-TV mode uses accept_quality.
		var acceptQualityArr []json.RawMessage
		if err := json.Unmarshal(acceptQualityRaw, &acceptQualityArr); err == nil && acceptQualityArr != nil {
			for _, raw := range acceptQualityArr {
				qn := rawToString(raw)
				if qn != "" {
					result.Dfns = append(result.Dfns, qn)
				}
			}
		}
	}

	return result, nil
}
