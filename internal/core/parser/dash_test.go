package parser

import (
	"strconv"
	"testing"
)

func buildBasicDashJSON() string {
	return `{
		"data": {
			"aid": "12345",
			"cid": "67890",
			"timelength": 300000,
			"dash": {
				"duration": 300,
				"video": [
					{
						"id": 80,
						"base_url": "https://example.com/video80.mp4",
						"backup_url": ["https://backup.example.com/video80.mp4"],
						"bandwidth": 2000000,
						"codecid": 7,
						"width": 1920,
						"height": 1080,
						"frame_rate": "30",
						"size": 75.5
					},
					{
						"id": 64,
						"base_url": "https://example.com/video64.mp4",
						"backup_url": [],
						"bandwidth": 1000000,
						"codecid": 7,
						"width": 1280,
						"height": 720,
						"frame_rate": "30",
						"size": 40.0
					},
					{
						"id": 80,
						"base_url": "https://example.com/video80_hevc.mp4",
						"backup_url": [],
						"bandwidth": 1500000,
						"codecid": 12,
						"width": 1920,
						"height": 1080,
						"frame_rate": "30",
						"size": 55.0
					}
				],
				"audio": [
					{
						"id": 30280,
						"base_url": "https://example.com/audio30280.m4a",
						"backup_url": null,
						"bandwidth": 128000,
						"codecs": "mp4a.40.2"
					},
					{
						"id": 30216,
						"base_url": "https://example.com/audio30216.m4a",
						"backup_url": [],
						"bandwidth": 64000,
						"codecs": "mp4a.40.5"
					}
				]
			}
		}
	}`
}

func TestParseDashTracks_VideoTracks(t *testing.T) {
	jsonStr := buildBasicDashJSON()
	videos, _, bgAudios, roleAudios, err := ParseDashTracks(jsonStr, false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(videos) != 3 {
		t.Fatalf("expected 3 video tracks, got %d", len(videos))
	}

	v1 := videos[0]
	if v1.ID != "80" {
		t.Errorf("video[0].ID = %s, want 80", v1.ID)
	}
	if v1.Dfn != "1080P 高清" {
		t.Errorf("video[0].Dfn = %s, want 1080P 高清", v1.Dfn)
	}
	if v1.Bandwith != 2000 {
		t.Errorf("video[0].Bandwith = %d, want 2000", v1.Bandwith)
	}
	if v1.Codecs != "AVC" {
		t.Errorf("video[0].Codecs = %s, want AVC", v1.Codecs)
	}
	if v1.Res != "1920x1080" {
		t.Errorf("video[0].Res = %s, want 1920x1080", v1.Res)
	}
	if v1.Fps != "30" {
		t.Errorf("video[0].Fps = %s, want 30", v1.Fps)
	}
	if v1.Size != 75.5 {
		t.Errorf("video[0].Size = %f, want 75.5", v1.Size)
	}
	if v1.Dur != 300 {
		t.Errorf("video[0].Dur = %d, want 300", v1.Dur)
	}
	if v1.BaseUrl != "https://example.com/video80.mp4" {
		t.Errorf("video[0].BaseUrl = %s, want https://example.com/video80.mp4", v1.BaseUrl)
	}

	v3 := videos[2]
	if v3.ID != "80" {
		t.Errorf("video[2].ID = %s, want 80", v3.ID)
	}
	if v3.Codecs != "HEVC" {
		t.Errorf("video[2].Codecs = %s, want HEVC", v3.Codecs)
	}

	if len(bgAudios) != 0 || len(roleAudios) != 0 {
		t.Error("expected no background/role audio tracks in this focused test")
	}
}

func TestParseDashTracks_AudioTracks(t *testing.T) {
	jsonStr := buildBasicDashJSON()
	_, audios, _, _, err := ParseDashTracks(jsonStr, false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(audios) != 2 {
		t.Fatalf("expected 2 audio tracks, got %d", len(audios))
	}

	a1 := audios[0]
	if a1.ID != "30280" {
		t.Errorf("audio[0].ID = %s, want 30280", a1.ID)
	}
	if a1.Codecs != "M4A" {
		t.Errorf("audio[0].Codecs = %s, want M4A", a1.Codecs)
	}
	if a1.Bandwith != 128 {
		t.Errorf("audio[0].Bandwith = %d, want 128", a1.Bandwith)
	}
	if a1.Dur != 300 {
		t.Errorf("audio[0].Dur = %d, want 300", a1.Dur)
	}

	a2 := audios[1]
	if a2.Codecs != "M4A" {
		t.Errorf("audio[1].Codecs = %s, want M4A", a2.Codecs)
	}
}

func TestParseDashTracks_DolbyAndFlac(t *testing.T) {
	jsonStr := `{
		"data": {
			"timelength": 600000,
			"dash": {
				"duration": 600,
				"video": [],
				"audio": [
					{
						"id": 30280,
						"base_url": "https://example.com/audio.m4a",
						"backup_url": [],
						"bandwidth": 128000,
						"codecs": "mp4a.40.2"
					}
				],
				"dolby": {
					"audio": [
						{
							"id": 30250,
							"base_url": "https://example.com/dolby.mp4",
							"backup_url": [],
							"bandwidth": 256000,
							"codecs": "ec-3"
						}
					]
				},
				"flac": {
					"audio": {
						"id": 30251,
						"base_url": "https://example.com/flac.mp4",
						"backup_url": [],
						"bandwidth": 1024000,
						"codecs": "fLaC"
					}
				}
			}
		}
	}`

	_, audios, _, _, err := ParseDashTracks(jsonStr, false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(audios) != 3 {
		t.Fatalf("expected 3 audio tracks, got %d", len(audios))
	}

	if audios[1].ID != "30250" {
		t.Errorf("audio[1].ID = %s, want 30250", audios[1].ID)
	}
	if audios[1].Codecs != "E-AC-3" {
		t.Errorf("audio[1].Codecs = %s, want E-AC-3", audios[1].Codecs)
	}

	if audios[2].ID != "30251" {
		t.Errorf("audio[2].ID = %s, want 30251", audios[2].ID)
	}
	if audios[2].Codecs != "FLAC" {
		t.Errorf("audio[2].Codecs = %s, want FLAC", audios[2].Codecs)
	}
}

func TestParseDashTracks_PCDNFilter(t *testing.T) {
	jsonStr := `{
		"data": {
			"timelength": 100000,
			"dash": {
				"duration": 100,
				"video": [
					{
						"id": 80,
						"base_url": "http://pcdn.example.com:8080/video.mp4",
						"backup_url": ["https://example.com/video.mp4"],
						"bandwidth": 1000000,
						"codecid": 7,
						"width": 1920,
						"height": 1080,
						"frame_rate": "30"
					}
				],
				"audio": [
					{
						"id": 30280,
						"base_url": "http://pcdn.example.com:9090/audio.m4a",
						"backup_url": ["http://pcdn2.example.com:9091/audio.m4a", "https://example.com/audio.m4a"],
						"bandwidth": 128000,
						"codecs": "mp4a.40.2"
					}
				]
			}
		}
	}`

	videos, audios, _, _, err := ParseDashTracks(jsonStr, false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if videos[0].BaseUrl != "https://example.com/video.mp4" {
		t.Errorf("video BaseUrl = %s, want https://example.com/video.mp4", videos[0].BaseUrl)
	}

	if audios[0].BaseUrl != "https://example.com/audio.m4a" {
		t.Errorf("audio BaseUrl = %s, want https://example.com/audio.m4a", audios[0].BaseUrl)
	}
}

func TestParseDashTracks_Deduplication(t *testing.T) {
	jsonStr := `{
		"data": {
			"timelength": 100000,
			"dash": {
				"duration": 100,
				"video": [
					{
						"id": 80,
						"base_url": "https://example.com/video1.mp4",
						"backup_url": [],
						"bandwidth": 1000000,
						"codecid": 7,
						"width": 1920,
						"height": 1080,
						"frame_rate": "30"
					},
					{
						"id": 80,
						"base_url": "https://example.com/video2.mp4",
						"backup_url": [],
						"bandwidth": 1000000,
						"codecid": 7,
						"width": 1920,
						"height": 1080,
						"frame_rate": "30"
					}
				],
				"audio": [
					{
						"id": 30280,
						"base_url": "https://example.com/audio1.m4a",
						"backup_url": [],
						"bandwidth": 128000,
						"codecs": "mp4a.40.2"
					},
					{
						"id": 30280,
						"base_url": "https://example.com/audio2.m4a",
						"backup_url": [],
						"bandwidth": 128000,
						"codecs": "mp4a.40.2"
					}
				]
			}
		}
	}`

	videos, audios, _, _, err := ParseDashTracks(jsonStr, false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(videos) != 1 {
		t.Errorf("expected 1 video track after dedup, got %d", len(videos))
	}

	if len(audios) != 1 {
		t.Errorf("expected 1 audio track after dedup, got %d", len(audios))
	}
}

func TestParseDashTracks_BackgroundAndRoleAudio(t *testing.T) {
	jsonStr := `{
		"data": {
			"aid": "111",
			"cid": "222",
			"timelength": 100000,
			"dash": {
				"duration": 100,
				"video": [],
				"audio": []
			},
			"dubbing_info": {
				"background_audio": [
					{
						"id": 30280,
						"base_url": "https://example.com/bg.m4a",
						"backup_url": [],
						"bandwidth": 128000,
						"codecs": "mp4a.40.2"
					}
				],
				"role_audio_list": [
					{
						"audio_id": "1001",
						"title": "Role Title",
						"person_name": "Actor Name",
						"audio": [
							{
								"id": 30281,
								"base_url": "https://example.com/role.m4a",
								"backup_url": [],
								"bandwidth": 128000,
								"codecs": "mp4a.40.2"
							}
						]
					}
				]
			}
		}
	}`

	_, _, bgAudios, roleAudios, err := ParseDashTracks(jsonStr, false, true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(bgAudios) != 1 {
		t.Fatalf("expected 1 background audio track, got %d", len(bgAudios))
	}
	if bgAudios[0].ID != "30280" {
		t.Errorf("bgAudio[0].ID = %s, want 30280", bgAudios[0].ID)
	}

	if len(roleAudios) != 1 {
		t.Fatalf("expected 1 role audio entry, got %d", len(roleAudios))
	}
	if roleAudios[0].Title != "Role Title" {
		t.Errorf("roleAudio[0].Title = %s, want Role Title", roleAudios[0].Title)
	}
	if roleAudios[0].PersonName != "Actor Name" {
		t.Errorf("roleAudio[0].PersonName = %s, want Actor Name", roleAudios[0].PersonName)
	}
	if roleAudios[0].Path != "111/111.222.1001.m4a" {
		t.Errorf("roleAudio[0].Path = %s, want 111/111.222.1001.m4a", roleAudios[0].Path)
	}
	if len(roleAudios[0].Audio) != 1 {
		t.Errorf("expected 1 role audio track, got %d", len(roleAudios[0].Audio))
	}
}

func TestParseDashTracks_NoDash(t *testing.T) {
	jsonStr := `{"data": {"timelength": 100000}}`
	videos, audios, bgAudios, roleAudios, err := ParseDashTracks(jsonStr, false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(videos) != 0 || len(audios) != 0 || len(bgAudios) != 0 || len(roleAudios) != 0 {
		t.Error("expected empty results when no dash")
	}
}

func TestParseDashTracks_TvApiNoResFps(t *testing.T) {
	jsonStr := `{
		"data": {
			"timelength": 100000,
			"dash": {
				"duration": 100,
				"video": [
					{
						"id": 80,
						"base_url": "https://example.com/video.mp4",
						"backup_url": [],
						"bandwidth": 1000000,
						"codecid": 7,
						"width": 1920,
						"height": 1080,
						"frame_rate": "30"
					}
				],
				"audio": []
			}
		}
	}`

	videos, _, _, _, err := ParseDashTracks(jsonStr, true, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if videos[0].Res != "" {
		t.Errorf("tvApi: expected empty Res, got %s", videos[0].Res)
	}
	if videos[0].Fps != "" {
		t.Errorf("tvApi: expected empty Fps, got %s", videos[0].Fps)
	}
}

func TestParseDashTracks_AppApiNoResFps(t *testing.T) {
	jsonStr := `{
		"data": {
			"timelength": 100000,
			"dash": {
				"duration": 100,
				"video": [
					{
						"id": 80,
						"base_url": "https://example.com/video.mp4",
						"backup_url": [],
						"bandwidth": 1000000,
						"codecid": 7,
						"width": 1920,
						"height": 1080,
						"frame_rate": "30"
					}
				],
				"audio": []
			}
		}
	}`

	videos, _, _, _, err := ParseDashTracks(jsonStr, false, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if videos[0].Res != "" {
		t.Errorf("appApi: expected empty Res, got %s", videos[0].Res)
	}
	if videos[0].Fps != "" {
		t.Errorf("appApi: expected empty Fps, got %s", videos[0].Fps)
	}
}

func TestParseDashTracks_DolbyNotProcessedWhenTvApi(t *testing.T) {
	jsonStr := `{
		"data": {
			"timelength": 100000,
			"dash": {
				"duration": 100,
				"video": [],
				"audio": [
					{
						"id": 30280,
						"base_url": "https://example.com/audio.m4a",
						"backup_url": [],
						"bandwidth": 128000,
						"codecs": "mp4a.40.2"
					}
				],
				"dolby": {
					"audio": [
						{
							"id": 30250,
							"base_url": "https://example.com/dolby.mp4",
							"backup_url": [],
							"bandwidth": 256000,
							"codecs": "ec-3"
						}
					]
				}
			}
		}
	}`

	_, audios, _, _, err := ParseDashTracks(jsonStr, true, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(audios) != 1 {
		t.Fatalf("expected 1 audio track (no dolby when tvApi), got %d", len(audios))
	}
}

func TestParseDashTracks_NoAudioFieldNoDolbyFlac(t *testing.T) {
	jsonStr := `{
		"data": {
			"timelength": 100000,
			"dash": {
				"duration": 100,
				"video": [],
				"dolby": {
					"audio": [
						{
							"id": 30250,
							"base_url": "https://example.com/dolby.mp4",
							"backup_url": [],
							"bandwidth": 256000,
							"codecs": "ec-3"
						}
					]
				},
				"flac": {
					"audio": {
						"id": 30251,
						"base_url": "https://example.com/flac.mp4",
						"backup_url": [],
						"bandwidth": 1024000,
						"codecs": "fLaC"
					}
				}
			}
		}
	}`

	_, audios, _, _, err := ParseDashTracks(jsonStr, false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(audios) != 0 {
		t.Fatalf("expected 0 audio tracks when no audio field, got %d", len(audios))
	}
}

func TestParseDashTracks_BackupUrlNull(t *testing.T) {
	jsonStr := `{
		"data": {
			"timelength": 100000,
			"dash": {
				"duration": 100,
				"video": [
					{
						"id": 80,
						"base_url": "https://example.com/video.mp4",
						"backup_url": null,
						"bandwidth": 1000000,
						"codecid": 7,
						"width": 1920,
						"height": 1080,
						"frame_rate": "30"
					}
				],
				"audio": [
					{
						"id": 30280,
						"base_url": "https://example.com/audio.m4a",
						"backup_url": null,
						"bandwidth": 128000,
						"codecs": "mp4a.40.2"
					}
				]
			}
		}
	}`

	videos, audios, _, _, err := ParseDashTracks(jsonStr, false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if videos[0].BaseUrl != "https://example.com/video.mp4" {
		t.Errorf("video BaseUrl = %s, want https://example.com/video.mp4", videos[0].BaseUrl)
	}
	if audios[0].BaseUrl != "https://example.com/audio.m4a" {
		t.Errorf("audio BaseUrl = %s, want https://example.com/audio.m4a", audios[0].BaseUrl)
	}
}

func TestParseDashTracks_ResultPath(t *testing.T) {
	jsonStr := `{
		"result": {
			"timelength": 120000,
			"dash": {
				"duration": 120,
				"video": [
					{
						"id": 32,
						"base_url": "https://example.com/video32.mp4",
						"backup_url": [],
						"bandwidth": 500000,
						"codecid": 13,
						"width": 854,
						"height": 480,
						"frame_rate": "24"
					}
				],
				"audio": []
			}
		}
	}`

	videos, _, _, _, err := ParseDashTracks(jsonStr, false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(videos) != 1 {
		t.Fatalf("expected 1 video track, got %d", len(videos))
	}
	if videos[0].ID != "32" {
		t.Errorf("video.ID = %s, want 32", videos[0].ID)
	}
	if videos[0].Codecs != "AV1" {
		t.Errorf("video.Codecs = %s, want AV1", videos[0].Codecs)
	}
	if videos[0].Dur != 120 {
		t.Errorf("video.Dur = %d, want 120", videos[0].Dur)
	}
}

func TestParseDashTracks_ResultVideoInfoPath(t *testing.T) {
	jsonStr := `{
		"result": {
			"video_info": {
				"timelength": 150000,
				"dash": {
					"duration": 150,
					"video": [
						{
							"id": 16,
							"base_url": "https://example.com/video16.mp4",
							"backup_url": [],
							"bandwidth": 300000,
							"codecid": 99,
							"width": 640,
							"height": 360,
							"frame_rate": "24"
						}
					],
					"audio": []
				}
			}
		}
	}`

	videos, _, _, _, err := ParseDashTracks(jsonStr, false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(videos) != 1 {
		t.Fatalf("expected 1 video track, got %d", len(videos))
	}
	if videos[0].ID != "16" {
		t.Errorf("video.ID = %s, want 16", videos[0].ID)
	}
	if videos[0].Codecs != "UNKNOWN" {
		t.Errorf("video.Codecs = %s, want UNKNOWN", videos[0].Codecs)
	}
	if videos[0].Dur != 150 {
		t.Errorf("video.Dur = %d, want 150", videos[0].Dur)
	}
}

func TestParseDashTracks_FlacNull(t *testing.T) {
	jsonStr := `{
		"data": {
			"timelength": 100000,
			"dash": {
				"duration": 100,
				"video": [],
				"audio": [
					{
						"id": 30280,
						"base_url": "https://example.com/audio.m4a",
						"backup_url": [],
						"bandwidth": 128000,
						"codecs": "mp4a.40.2"
					}
				],
				"flac": {
					"audio": null
				}
			}
		}
	}`

	_, audios, _, _, err := ParseDashTracks(jsonStr, false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(audios) != 1 {
		t.Fatalf("expected 1 audio track, got %d", len(audios))
	}
}

func TestParseDashTracks_DubbingNotParsedWithoutAppApiBangumi(t *testing.T) {
	jsonStr := `{
		"data": {
			"aid": "111",
			"cid": "222",
			"timelength": 100000,
			"dash": {
				"duration": 100,
				"video": [],
				"audio": []
			},
			"dubbing_info": {
				"background_audio": [
					{
						"id": 30280,
						"base_url": "https://example.com/bg.m4a",
						"backup_url": [],
						"bandwidth": 128000,
						"codecs": "mp4a.40.2"
					}
				],
				"role_audio_list": []
			}
		}
	}`

	_, _, bgAudios, roleAudios, err := ParseDashTracks(jsonStr, false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bgAudios) != 0 {
		t.Errorf("expected 0 background audio without appApi+bangumi, got %d", len(bgAudios))
	}
	if len(roleAudios) != 0 {
		t.Errorf("expected 0 role audio without appApi+bangumi, got %d", len(roleAudios))
	}
}

func TestParseDashTracks_EqualAudioDeduplication(t *testing.T) {
	// Two audio tracks with same Equal fields but different BaseUrl
	jsonStr := `{
		"data": {
			"timelength": 100000,
			"dash": {
				"duration": 100,
				"video": [],
				"audio": [
					{
						"id": 30280,
						"base_url": "https://example.com/audio1.m4a",
						"backup_url": [],
						"bandwidth": 128000,
						"codecs": "mp4a.40.2"
					},
					{
						"id": 30280,
						"base_url": "https://example.com/audio2.m4a",
						"backup_url": [],
						"bandwidth": 128000,
						"codecs": "mp4a.40.2"
					}
				]
			}
		}
	}`

	_, audios, _, _, err := ParseDashTracks(jsonStr, false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(audios) != 1 {
		t.Fatalf("expected 1 audio track after dedup, got %d", len(audios))
	}
	// Should keep the first non-PCDN base URL
	if audios[0].BaseUrl != "https://example.com/audio1.m4a" {
		t.Errorf("audio BaseUrl = %s, want https://example.com/audio1.m4a", audios[0].BaseUrl)
	}
}

func TestParseDashTracks_VideoTrackDetails(t *testing.T) {
	// Test all video codec mappings
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
				"timelength": 100000,
				"dash": {
					"duration": 100,
					"video": [
						{
							"id": 80,
							"base_url": "https://example.com/video.mp4",
							"backup_url": [],
							"bandwidth": 1000000,
							"codecid": ` + strconv.Itoa(tc.codecid) + `,
							"width": 1920,
							"height": 1080,
							"frame_rate": "30"
						}
					],
					"audio": []
				}
			}
		}`

		videos, _, _, _, err := ParseDashTracks(jsonStr, false, false, false)
		if err != nil {
			t.Fatalf("unexpected error for codecid %d: %v", tc.codecid, err)
		}
		if len(videos) != 1 {
			t.Fatalf("expected 1 video track for codecid %d, got %d", tc.codecid, len(videos))
		}
		if videos[0].Codecs != tc.expected {
			t.Errorf("codecid %d: Codecs = %s, want %s", tc.codecid, videos[0].Codecs, tc.expected)
		}
	}
}
