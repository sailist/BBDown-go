package app

import (
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/sailist/BBDown-go/internal/core/entity"
)

// SortVideoTracks sorts video tracks by dfn/encoding priority, id, and bandwidth.
func SortVideoTracks(tracks []entity.Video, dfnPriority map[string]int, encodingPriority map[string]byte, ascending bool) []entity.Video {
	if len(tracks) == 0 {
		return tracks
	}

	encodingFirst := false
	if len(dfnPriority) > 0 && len(encodingPriority) > 0 {
		encodingFirst = encodingPriorityBeforeDfnPriority()
	}

	return sortVideoTracksWithOrder(tracks, dfnPriority, encodingPriority, ascending, encodingFirst)
}

// SortAudioTracks sorts audio tracks by encoding priority and bandwidth.
func SortAudioTracks(tracks []entity.Audio, encodingPriority map[string]byte, ascending bool) []entity.Audio {
	if len(tracks) == 0 {
		return tracks
	}

	result := make([]entity.Audio, len(tracks))
	copy(result, tracks)

	sort.SliceStable(result, func(i, j int) bool {
		a, b := result[i], result[j]

		var aEncPri, bEncPri int
		if len(encodingPriority) > 0 {
			aEncPri = int(getBytePriority(encodingPriority, a.ShortCodecs(), 100))
			bEncPri = int(getBytePriority(encodingPriority, b.ShortCodecs(), 100))
		}

		if aEncPri != bEncPri {
			return aEncPri < bEncPri
		}

		if ascending {
			return a.Bandwith < b.Bandwith
		}
		return a.Bandwith > b.Bandwith
	})

	return result
}

func sortVideoTracksWithOrder(tracks []entity.Video, dfnPriority map[string]int, encodingPriority map[string]byte, ascending, encodingFirst bool) []entity.Video {
	result := make([]entity.Video, len(tracks))
	copy(result, tracks)

	sort.SliceStable(result, func(i, j int) bool {
		a, b := result[i], result[j]

		var aDfnPri, bDfnPri int
		if len(dfnPriority) > 0 {
			aDfnPri = getPriority(dfnPriority, a.Dfn, 100)
			bDfnPri = getPriority(dfnPriority, b.Dfn, 100)
		}

		var aEncPri, bEncPri int
		if len(encodingPriority) > 0 {
			aEncPri = int(getBytePriority(encodingPriority, a.Codecs, 100))
			bEncPri = int(getBytePriority(encodingPriority, b.Codecs, 100))
		}

		if encodingFirst {
			if aEncPri != bEncPri {
				return aEncPri < bEncPri
			}
			if aDfnPri != bDfnPri {
				return aDfnPri < bDfnPri
			}
		} else {
			if aDfnPri != bDfnPri {
				return aDfnPri < bDfnPri
			}
			if aEncPri != bEncPri {
				return aEncPri < bEncPri
			}
		}

		aID, _ := strconv.Atoi(a.ID)
		bID, _ := strconv.Atoi(b.ID)
		if aID != bID {
			return aID > bID
		}

		if ascending {
			return a.Bandwith < b.Bandwith
		}
		return a.Bandwith > b.Bandwith
	})

	return result
}

func encodingPriorityBeforeDfnPriority() bool {
	encIdx := -1
	dfnIdx := -1
	for i, arg := range os.Args {
		if strings.HasPrefix(arg, "--encoding-priority") {
			encIdx = i
		} else if strings.HasPrefix(arg, "--dfn-priority") {
			dfnIdx = i
		}
	}
	return encIdx != -1 && dfnIdx != -1 && encIdx < dfnIdx
}

func getPriority(m map[string]int, key string, defaultVal int) int {
	if v, ok := m[key]; ok {
		return v
	}
	return defaultVal
}

func getBytePriority(m map[string]byte, key string, defaultVal byte) byte {
	if v, ok := m[key]; ok {
		return v
	}
	return defaultVal
}
