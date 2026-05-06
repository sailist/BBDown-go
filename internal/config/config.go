package config

// Config holds runtime configuration for BBDown.
type Config struct {
	Cookie   string
	Token    string
	DebugLog bool
	Host     string
	EpHost   string
	TvHost   string
	Area     string
	WBI      string
}

// NewConfig returns a new Config with default values.
func NewConfig() *Config {
	return &Config{
		Host:   "api.bilibili.com",
		EpHost: "api.bilibili.com",
		TvHost: "api.snm0516.aisee.tv",
	}
}

// Qualities maps quality codes to human-readable descriptions.
var Qualities = map[string]string{
	"127": "8K 超高清",
	"126": "杜比视界",
	"125": "HDR 真彩",
	"120": "4K 超清",
	"116": "1080P 高帧率",
	"112": "1080P 高码率",
	"100": "智能修复",
	"80":  "1080P 高清",
	"74":  "720P 高帧率",
	"64":  "720P 高清",
	"48":  "720P 高清",
	"32":  "480P 清晰",
	"16":  "360P 流畅",
	"5":   "144P 流畅",
	"6":   "240P 流畅",
}
