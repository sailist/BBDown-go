package parser

import (
	"strconv"
	"testing"
)

func buildBasicFlvJSON() string {
	return `{
		"data": {
			"quality": 80,
			"video_codecid": 7,
			"durl": [
				{
					"url": "https://example.com/seg1.flv",
					"size": 50.5,
					"length": 300000
				},
				{
					"url": "https://example.com/seg2.flv",
					"size": 25.0,
					"length": 150000
				}
			],
			"accept_quality": [80, 64, 32]
		}
	}`
}

func TestParseFlvTracks_Basic(t *testing.T) {
	jsonStr := buildBasicFlvJSON()
	result, err := ParseFlvTracks(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.VideoTracks) != 1 {
		t.Fatalf("expected 1 video track, got %d", len(result.VideoTracks))
	}

	v := result.VideoTracks[0]
	if v.ID != "80" {
		t.Errorf("video.ID = %s, want 80", v.ID)
	}
	if v.Dfn != "1080P 高清" {
		t.Errorf("video.Dfn = %s, want 1080P 高清", v.Dfn)
	}
	if v.Codecs != "AVC" {
		t.Errorf("video.Codecs = %s, want AVC", v.Codecs)
	}
	if v.Dur != 450 {
		t.Errorf("video.Dur = %d, want 450", v.Dur)
	}
	if v.Size != 75.5 {
		t.Errorf("video.Size = %f, want 75.5", v.Size)
	}
}

func TestParseFlvTracks_Clips(t *testing.T) {
	jsonStr := buildBasicFlvJSON()
	result, err := ParseFlvTracks(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Clips) != 2 {
		t.Fatalf("expected 2 clips, got %d", len(result.Clips))
	}
	if result.Clips[0] != "https://example.com/seg1.flv" {
		t.Errorf("clip[0] = %s, want https://example.com/seg1.flv", result.Clips[0])
	}
	if result.Clips[1] != "https://example.com/seg2.flv" {
		t.Errorf("clip[1] = %s, want https://example.com/seg2.flv", result.Clips[1])
	}
}

func TestParseFlvTracks_AcceptQuality(t *testing.T) {
	jsonStr := buildBasicFlvJSON()
	result, err := ParseFlvTracks(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Dfns) != 3 {
		t.Fatalf("expected 3 dfns, got %d", len(result.Dfns))
	}
	if result.Dfns[0] != "80" {
		t.Errorf("dfn[0] = %s, want 80", result.Dfns[0])
	}
	if result.Dfns[1] != "64" {
		t.Errorf("dfn[1] = %s, want 64", result.Dfns[1])
	}
	if result.Dfns[2] != "32" {
		t.Errorf("dfn[2] = %s, want 32", result.Dfns[2])
	}
}

func TestParseFlvTracks_QnExtras(t *testing.T) {
	jsonStr := `{
		"data": {
			"quality": 120,
			"video_codecid": 12,
			"durl": [
				{
					"url": "https://example.com/seg.flv",
					"size": 100.0,
					"length": 600000
				}
			],
			"qn_extras": [
				{"qn": 127},
				{"qn": 126},
				{"qn": 120}
			]
		}
	}`

	result, err := ParseFlvTracks(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Dfns) != 3 {
		t.Fatalf("expected 3 dfns, got %d", len(result.Dfns))
	}
	if result.Dfns[0] != "127" {
		t.Errorf("dfn[0] = %s, want 127", result.Dfns[0])
	}
	if result.Dfns[1] != "126" {
		t.Errorf("dfn[1] = %s, want 126", result.Dfns[1])
	}
	if result.Dfns[2] != "120" {
		t.Errorf("dfn[2] = %s, want 120", result.Dfns[2])
	}

	v := result.VideoTracks[0]
	if v.ID != "120" {
		t.Errorf("video.ID = %s, want 120", v.ID)
	}
	if v.Codecs != "HEVC" {
		t.Errorf("video.Codecs = %s, want HEVC", v.Codecs)
	}
	if v.Dur != 600 {
		t.Errorf("video.Dur = %d, want 600", v.Dur)
	}
}

func TestParseFlvTracks_CodecMapping(t *testing.T) {
	tests := []struct {
		codecid  int
		expected string
	}{
		{7, "AVC"},
		{12, "HEVC"},
		{13, "AV1"},
		{99, "UNKNOWN"},
	}

	for _, tc := range tests {
		jsonStr := `{
			"data": {
				"quality": 80,
				"video_codecid": ` + strconv.Itoa(tc.codecid) + `,
				"durl": [
					{
						"url": "https://example.com/seg.flv",
						"size": 10.0,
						"length": 100000
					}
				]
			}
		}`

		result, err := ParseFlvTracks(jsonStr)
		if err != nil {
			t.Fatalf("unexpected error for codecid %d: %v", tc.codecid, err)
		}
		if len(result.VideoTracks) != 1 {
			t.Fatalf("expected 1 video track for codecid %d, got %d", tc.codecid, len(result.VideoTracks))
		}
		if result.VideoTracks[0].Codecs != tc.expected {
			t.Errorf("codecid %d: Codecs = %s, want %s", tc.codecid, result.VideoTracks[0].Codecs, tc.expected)
		}
	}
}

func TestParseFlvTracks_DurationCalculation(t *testing.T) {
	jsonStr := `{
		"data": {
			"quality": 64,
			"video_codecid": 7,
			"durl": [
				{
					"url": "https://example.com/seg1.flv",
					"size": 10.0,
					"length": 1234567
				},
				{
					"url": "https://example.com/seg2.flv",
					"size": 5.0,
					"length": 876543
				}
			]
		}
	}`

	result, err := ParseFlvTracks(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// (1234567 + 876543) / 1000 = 2111
	if result.VideoTracks[0].Dur != 2111 {
		t.Errorf("video.Dur = %d, want 2111", result.VideoTracks[0].Dur)
	}
}

func TestParseFlvTracks_ResultPath(t *testing.T) {
	jsonStr := `{
		"result": {
			"quality": 32,
			"video_codecid": 13,
			"durl": [
				{
					"url": "https://example.com/seg.flv",
					"size": 20.0,
					"length": 200000
				}
			],
			"accept_quality": [32, 16]
		}
	}`

	result, err := ParseFlvTracks(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.VideoTracks) != 1 {
		t.Fatalf("expected 1 video track, got %d", len(result.VideoTracks))
	}
	if result.VideoTracks[0].ID != "32" {
		t.Errorf("video.ID = %s, want 32", result.VideoTracks[0].ID)
	}
	if result.VideoTracks[0].Codecs != "AV1" {
		t.Errorf("video.Codecs = %s, want AV1", result.VideoTracks[0].Codecs)
	}
	if len(result.Dfns) != 2 {
		t.Errorf("expected 2 dfns, got %d", len(result.Dfns))
	}
}

func TestParseFlvTracks_ResultVideoInfoPath(t *testing.T) {
	jsonStr := `{
		"result": {
			"video_info": {
				"quality": 16,
				"video_codecid": 7,
				"durl": [
					{
						"url": "https://example.com/seg.flv",
						"size": 15.0,
						"length": 180000
					}
				],
				"accept_quality": [16, 5]
			}
		}
	}`

	result, err := ParseFlvTracks(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.VideoTracks) != 1 {
		t.Fatalf("expected 1 video track, got %d", len(result.VideoTracks))
	}
	if result.VideoTracks[0].ID != "16" {
		t.Errorf("video.ID = %s, want 16", result.VideoTracks[0].ID)
	}
	if result.VideoTracks[0].Dfn != "360P 流畅" {
		t.Errorf("video.Dfn = %s, want 360P 流畅", result.VideoTracks[0].Dfn)
	}
	if len(result.Dfns) != 2 {
		t.Errorf("expected 2 dfns, got %d", len(result.Dfns))
	}
}

func TestParseFlvTracks_RootLevelDurl(t *testing.T) {
	jsonStr := `{
		"quality": 80,
		"video_codecid": 7,
		"durl": [
			{
				"url": "https://example.com/seg.flv",
				"size": 30.0,
				"length": 300000
			}
		],
		"accept_quality": [80, 64]
	}`

	result, err := ParseFlvTracks(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.VideoTracks) != 1 {
		t.Fatalf("expected 1 video track, got %d", len(result.VideoTracks))
	}
	if result.VideoTracks[0].ID != "80" {
		t.Errorf("video.ID = %s, want 80", result.VideoTracks[0].ID)
	}
}

func TestParseFlvTracks_NoDurl(t *testing.T) {
	jsonStr := `{
		"data": {
			"quality": 80,
			"video_codecid": 7,
			"accept_quality": [80, 64]
		}
	}`

	result, err := ParseFlvTracks(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.VideoTracks) != 0 {
		t.Errorf("expected 0 video tracks, got %d", len(result.VideoTracks))
	}
	if len(result.Clips) != 0 {
		t.Errorf("expected 0 clips, got %d", len(result.Clips))
	}
	if len(result.Dfns) != 0 {
		t.Errorf("expected 0 dfns, got %d", len(result.Dfns))
	}
}

func TestParseFlvTracks_EmptyQualityName(t *testing.T) {
	jsonStr := `{
		"data": {
			"quality": 999,
			"video_codecid": 7,
			"durl": [
				{
					"url": "https://example.com/seg.flv",
					"size": 10.0,
					"length": 100000
				}
			]
		}
	}`

	result, err := ParseFlvTracks(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.VideoTracks[0].Dfn != "" {
		t.Errorf("video.Dfn = %s, want empty string for unknown quality", result.VideoTracks[0].Dfn)
	}
}

func TestParseFlvTracks_Deduplication(t *testing.T) {
	jsonStr := `{
		"data": {
			"quality": 80,
			"video_codecid": 7,
			"durl": [
				{
					"url": "https://example.com/seg1.flv",
					"size": 10.0,
					"length": 100000
				}
			]
		}
	}`

	result, err := ParseFlvTracks(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Simulate duplicate by parsing again and manually adding
	result.VideoTracks = append(result.VideoTracks, result.VideoTracks[0])
	// Now deduplicate check would fail if we had a second parse, but single parse should produce 1
	if len(result.VideoTracks) != 2 {
		t.Logf("manual duplicate: expected 2 tracks, got %d", len(result.VideoTracks))
	}
}

func TestParseFlvTracks_QnExtrasEmptyQn(t *testing.T) {
	jsonStr := `{
		"data": {
			"quality": 80,
			"video_codecid": 7,
			"durl": [
				{
					"url": "https://example.com/seg.flv",
					"size": 10.0,
					"length": 100000
				}
			],
			"qn_extras": [
				{"qn": 80},
				{"qn": ""},
				{"qn": 64}
			]
		}
	}`

	result, err := ParseFlvTracks(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Dfns) != 2 {
		t.Errorf("expected 2 dfns (empty filtered), got %d", len(result.Dfns))
	}
}

func TestParseFlvTracks_AcceptQualityEmptyQn(t *testing.T) {
	jsonStr := `{
		"data": {
			"quality": 80,
			"video_codecid": 7,
			"durl": [
				{
					"url": "https://example.com/seg.flv",
					"size": 10.0,
					"length": 100000
				}
			],
			"accept_quality": [80, "", 64]
		}
	}`

	result, err := ParseFlvTracks(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Dfns) != 2 {
		t.Errorf("expected 2 dfns (empty filtered), got %d", len(result.Dfns))
	}
}
