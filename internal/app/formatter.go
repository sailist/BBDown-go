package app

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sailist/BBDown-go/internal/core/entity"
	"github.com/sailist/BBDown-go/internal/core/util"
)

var infoRegex = regexp.MustCompile(`<([\w:\-.]+?)>`)

// FormatSavePath formats a save path template with video/audio/page metadata.
func FormatSavePath(format, title string, video *entity.Video, audio *entity.Audio, page entity.Page, pagesCount int, apiType string, pubTime int64) string {
	result := strings.ReplaceAll(format, "\\", "/")

	matches := infoRegex.FindAllStringSubmatchIndex(result, -1)
	if matches == nil {
		if !strings.HasSuffix(result, ".mp4") {
			result += ".mp4"
		}
		return result
	}

	// Process from right to left to preserve match indices
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]
		key := result[m[2]:m[3]]

		defaultDateFormat := "2006-01-02_15-04-05"
		for _, prefix := range []string{"publishDate:", "videoDate:"} {
			if strings.HasPrefix(key, prefix) {
				defaultDateFormat = key[strings.Index(key, ":")+1:]
				key = strings.TrimSuffix(prefix, ":")
				break
			}
		}

		var v string
		switch key {
		case "videoTitle":
			v = strings.TrimSpace(strings.TrimRight(util.GetValidFileName(title, "", true), "."))
		case "pageNumber":
			v = strconv.Itoa(page.Index)
		case "pageNumberWithZero":
			v = fmt.Sprintf("%0*d", len(strconv.Itoa(pagesCount)), page.Index)
		case "pageTitle":
			v = strings.TrimSpace(strings.TrimRight(util.GetValidFileName(page.Title, "", true), "."))
		case "bvid":
			v = page.BVid()
		case "aid":
			v = page.Aid
		case "cid":
			v = page.Cid
		case "ownerName":
			if page.OwnerName != "" {
				v = strings.TrimSpace(strings.TrimRight(util.GetValidFileName(page.OwnerName, "", true), "."))
			}
		case "ownerMid":
			v = page.OwnerMid
		case "dfn":
			if video != nil {
				v = video.Dfn
			}
		case "res":
			if video != nil {
				v = video.Res
			}
		case "fps":
			if video != nil {
				v = video.Fps
			}
		case "videoCodecs":
			if video != nil {
				v = video.Codecs
			}
		case "videoBandwidth":
			if video != nil {
				v = strconv.FormatInt(video.Bandwith, 10)
			}
		case "audioCodecs":
			if audio != nil {
				v = audio.Codecs
			}
		case "audioBandwidth":
			if audio != nil {
				v = strconv.FormatInt(audio.Bandwith, 10)
			}
		case "publishDate":
			v = formatTimestamp(pubTime, defaultDateFormat)
		case "videoDate":
			v = formatTimestamp(page.PubTime, defaultDateFormat)
		case "apiType":
			v = apiType
		default:
			v = fmt.Sprintf("<%s>", key)
		}

		result = result[:m[0]] + v + result[m[1]:]
	}

	if !strings.HasSuffix(result, ".mp4") {
		result += ".mp4"
	}
	return result
}

// GetValidFileName replaces invalid filename characters with replacement.
// If filterSlash is true, it also explicitly replaces '/' and '\\'.
func GetValidFileName(input string, replaceWith string, filterSlash bool) string {
	return util.GetValidFileName(input, replaceWith, filterSlash)
}

func formatTimestamp(ts int64, format string) string {
	if ts == 0 {
		return "null"
	}
	return time.Unix(ts, 0).UTC().Format(format)
}
