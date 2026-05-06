package config

import "testing"

func TestNewConfigDefaults(t *testing.T) {
	c := NewConfig()
	if c.Host != "api.bilibili.com" {
		t.Errorf("expected Host = api.bilibili.com, got %s", c.Host)
	}
	if c.EpHost != "api.bilibili.com" {
		t.Errorf("expected EpHost = api.bilibili.com, got %s", c.EpHost)
	}
	if c.TvHost != "api.snm0516.aisee.tv" {
		t.Errorf("expected TvHost = api.snm0516.aisee.tv, got %s", c.TvHost)
	}
	if c.Cookie != "" {
		t.Errorf("expected Cookie = empty string, got %s", c.Cookie)
	}
	if c.Token != "" {
		t.Errorf("expected Token = empty string, got %s", c.Token)
	}
	if c.DebugLog != false {
		t.Errorf("expected DebugLog = false, got %v", c.DebugLog)
	}
	if c.Area != "" {
		t.Errorf("expected Area = empty string, got %s", c.Area)
	}
	if c.WBI != "" {
		t.Errorf("expected WBI = empty string, got %s", c.WBI)
	}
}

func TestQualities(t *testing.T) {
	cases := map[string]string{
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
	for code, expected := range cases {
		if got, ok := Qualities[code]; !ok {
			t.Errorf("expected Qualities to contain %s", code)
		} else if got != expected {
			t.Errorf("expected Qualities[%s] = %s, got %s", code, expected, got)
		}
	}
}
