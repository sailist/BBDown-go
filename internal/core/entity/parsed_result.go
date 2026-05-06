package entity

// ParsedResult holds the result of parsing a Bilibili page for downloadable streams.
type ParsedResult struct {
	WebJsonString         string              `json:"webJsonString"`
	VideoTracks           []Video             `json:"videoTracks"`
	AudioTracks           []Audio             `json:"audioTracks"`
	BackgroundAudioTracks []Audio             `json:"backgroundAudioTracks"`
	RoleAudioList         []AudioMaterialInfo `json:"roleAudioList"`
	ExtraPoints           []ViewPoint         `json:"extraPoints"`
	Clips                 []string            `json:"clips"`
	Dfns                  []string            `json:"dfns"`
}
