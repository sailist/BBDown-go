package entity

// AudioMaterial represents a raw audio material entry.
type AudioMaterial struct {
	Title      string `json:"title"`
	PersonName string `json:"personName"`
	Path       string `json:"path"`
}

// AudioMaterialInfo represents an audio material entry with parsed audio tracks.
type AudioMaterialInfo struct {
	Title      string  `json:"title"`
	PersonName string  `json:"personName"`
	Path       string  `json:"path"`
	Audio      []Audio `json:"audio"`
}
