package entity

// Video represents a video stream track.
type Video struct {
	ID      string `json:"id"`
	Dfn     string `json:"dfn"`
	BaseUrl string `json:"baseUrl"`
	Res     string `json:"res"`
	Fps     string `json:"fps"`
	Codecs  string `json:"codecs"`
	Bandwith int64 `json:"bandwith"`
	Dur     int    `json:"dur"`
	Size    float64 `json:"size"`
}

// Equal reports whether v and other represent the same video track.
func (v Video) Equal(other Video) bool {
	return v.ID == other.ID &&
		v.Dfn == other.Dfn &&
		v.Res == other.Res &&
		v.Fps == other.Fps &&
		v.Codecs == other.Codecs &&
		v.Bandwith == other.Bandwith &&
		v.Dur == other.Dur
}

// Audio represents an audio stream track.
type Audio struct {
	ID       string `json:"id"`
	Dfn      string `json:"dfn"`
	BaseUrl  string `json:"baseUrl"`
	Codecs   string `json:"codecs"`
	Bandwith int64  `json:"bandwith"`
	Dur      int    `json:"dur"`
}

// ShortCodecs returns a normalized version of Codecs without hyphens and in upper case.
func (a Audio) ShortCodecs() string {
	result := make([]rune, 0, len(a.Codecs))
	for _, r := range a.Codecs {
		if r == '-' {
			continue
		}
		if r >= 'a' && r <= 'z' {
			result = append(result, r-'a'+'A')
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

// Equal reports whether a and other represent the same audio track.
func (a Audio) Equal(other Audio) bool {
	return a.ID == other.ID &&
		a.Dfn == other.Dfn &&
		a.Codecs == other.Codecs &&
		a.Bandwith == other.Bandwith &&
		a.Dur == other.Dur
}
