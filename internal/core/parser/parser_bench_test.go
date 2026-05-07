package parser

import "testing"

var benchDashJSON = `{
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

var benchFlvJSON = `{
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

func BenchmarkParseDashTracks(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, _, _, _ = ParseDashTracks(benchDashJSON, false, false, false)
	}
}

func BenchmarkParseFlvTracks(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ParseFlvTracks(benchFlvJSON)
	}
}
