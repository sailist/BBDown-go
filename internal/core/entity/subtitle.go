package entity

// Subtitle represents a subtitle resource.
type Subtitle struct {
	Lan  string `json:"lan"`
	Url  string `json:"url"`
	Path string `json:"path"`
}

// Clip represents a time-range clip.
type Clip struct {
	Index int   `json:"index"`
	From  int64 `json:"from"`
	To    int64 `json:"to"`
}
