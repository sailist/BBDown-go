package app

import (
	"os"
	"testing"

	"github.com/sailist/BBDown-go/internal/core/entity"
)

func TestSortVideoTracks_ByDfnPriority(t *testing.T) {
	tracks := []entity.Video{
		{ID: "1", Dfn: "720P", Codecs: "avc1", Bandwith: 1000},
		{ID: "2", Dfn: "1080P", Codecs: "avc1", Bandwith: 2000},
		{ID: "3", Dfn: "480P", Codecs: "avc1", Bandwith: 500},
	}

	dfnPriority := map[string]int{"1080P": 1, "720P": 2, "480P": 3}
	result := SortVideoTracks(tracks, dfnPriority, nil, false)

	if result[0].Dfn != "1080P" || result[1].Dfn != "720P" || result[2].Dfn != "480P" {
		t.Fatalf("unexpected order: %+v", result)
	}
}

func TestSortVideoTracks_ByEncodingPriority(t *testing.T) {
	tracks := []entity.Video{
		{ID: "1", Dfn: "1080P", Codecs: "hevc", Bandwith: 2000},
		{ID: "2", Dfn: "1080P", Codecs: "avc1", Bandwith: 2000},
		{ID: "3", Dfn: "1080P", Codecs: "av01", Bandwith: 2000},
	}

	encodingPriority := map[string]byte{"avc1": 1, "hevc": 2, "av01": 3}
	result := SortVideoTracks(tracks, nil, encodingPriority, false)

	if result[0].Codecs != "avc1" || result[1].Codecs != "hevc" || result[2].Codecs != "av01" {
		t.Fatalf("unexpected order: %+v", result)
	}
}

func TestSortVideoTracks_EncodingFirst(t *testing.T) {
	tracks := []entity.Video{
		{ID: "1", Dfn: "720P", Codecs: "hevc", Bandwith: 1000},
		{ID: "2", Dfn: "1080P", Codecs: "avc1", Bandwith: 2000},
	}

	dfnPriority := map[string]int{"1080P": 1, "720P": 2}
	encodingPriority := map[string]byte{"avc1": 1, "hevc": 2}

	// Simulate --encoding-priority before --dfn-priority
	oldArgs := os.Args
	os.Args = []string{"cmd", "--encoding-priority", "avc1", "--dfn-priority", "1080P"}
	defer func() { os.Args = oldArgs }()

	result := SortVideoTracks(tracks, dfnPriority, encodingPriority, false)

	// Encoding first: avc1 (1) < hevc (2), so avc1 video should be first
	if result[0].Codecs != "avc1" {
		t.Fatalf("expected encoding-first order, got: %+v", result)
	}
}

func TestSortVideoTracks_DfnFirst(t *testing.T) {
	tracks := []entity.Video{
		{ID: "1", Dfn: "720P", Codecs: "hevc", Bandwith: 1000},
		{ID: "2", Dfn: "1080P", Codecs: "avc1", Bandwith: 2000},
	}

	dfnPriority := map[string]int{"1080P": 1, "720P": 2}
	encodingPriority := map[string]byte{"avc1": 1, "hevc": 2}

	// Simulate --dfn-priority before --encoding-priority
	oldArgs := os.Args
	os.Args = []string{"cmd", "--dfn-priority", "1080P", "--encoding-priority", "avc1"}
	defer func() { os.Args = oldArgs }()

	result := SortVideoTracks(tracks, dfnPriority, encodingPriority, false)

	// Dfn first: 1080P (1) < 720P (2), so 1080P video should be first
	if result[0].Dfn != "1080P" {
		t.Fatalf("expected dfn-first order, got: %+v", result)
	}
}

func TestSortVideoTracks_ByIDDescending(t *testing.T) {
	tracks := []entity.Video{
		{ID: "1", Dfn: "1080P", Codecs: "avc1", Bandwith: 1000},
		{ID: "3", Dfn: "1080P", Codecs: "avc1", Bandwith: 1000},
		{ID: "2", Dfn: "1080P", Codecs: "avc1", Bandwith: 1000},
	}

	result := SortVideoTracks(tracks, nil, nil, false)

	if result[0].ID != "3" || result[1].ID != "2" || result[2].ID != "1" {
		t.Fatalf("expected ID descending, got: %+v", result)
	}
}

func TestSortVideoTracks_ByBandwidthAscending(t *testing.T) {
	tracks := []entity.Video{
		{ID: "1", Dfn: "1080P", Codecs: "avc1", Bandwith: 3000},
		{ID: "1", Dfn: "1080P", Codecs: "avc1", Bandwith: 1000},
		{ID: "1", Dfn: "1080P", Codecs: "avc1", Bandwith: 2000},
	}

	result := SortVideoTracks(tracks, nil, nil, true)

	if result[0].Bandwith != 1000 || result[1].Bandwith != 2000 || result[2].Bandwith != 3000 {
		t.Fatalf("expected bandwidth ascending, got: %+v", result)
	}
}

func TestSortVideoTracks_ByBandwidthDescending(t *testing.T) {
	tracks := []entity.Video{
		{ID: "1", Dfn: "1080P", Codecs: "avc1", Bandwith: 1000},
		{ID: "1", Dfn: "1080P", Codecs: "avc1", Bandwith: 3000},
		{ID: "1", Dfn: "1080P", Codecs: "avc1", Bandwith: 2000},
	}

	result := SortVideoTracks(tracks, nil, nil, false)

	if result[0].Bandwith != 3000 || result[1].Bandwith != 2000 || result[2].Bandwith != 1000 {
		t.Fatalf("expected bandwidth descending, got: %+v", result)
	}
}

func TestSortVideoTracks_Empty(t *testing.T) {
	var tracks []entity.Video
	result := SortVideoTracks(tracks, nil, nil, false)
	if len(result) != 0 {
		t.Fatalf("expected empty result")
	}
}

func TestSortAudioTracks_ByEncodingPriority(t *testing.T) {
	tracks := []entity.Audio{
		{ID: "1", Codecs: "flac", Bandwith: 1000},
		{ID: "2", Codecs: "mp4a.40.2", Bandwith: 1000},
		{ID: "3", Codecs: "ec-3", Bandwith: 1000},
	}

	encodingPriority := map[string]byte{"MP4A.40.2": 1, "FLAC": 2, "EC-3": 3}
	result := SortAudioTracks(tracks, encodingPriority, false)

	if result[0].Codecs != "mp4a.40.2" || result[1].Codecs != "flac" || result[2].Codecs != "ec-3" {
		t.Fatalf("unexpected order: %+v", result)
	}
}

func TestSortAudioTracks_ByBandwidthAscending(t *testing.T) {
	tracks := []entity.Audio{
		{ID: "1", Codecs: "mp4a.40.2", Bandwith: 3000},
		{ID: "2", Codecs: "mp4a.40.2", Bandwith: 1000},
		{ID: "3", Codecs: "mp4a.40.2", Bandwith: 2000},
	}

	result := SortAudioTracks(tracks, nil, true)

	if result[0].Bandwith != 1000 || result[1].Bandwith != 2000 || result[2].Bandwith != 3000 {
		t.Fatalf("expected bandwidth ascending, got: %+v", result)
	}
}

func TestSortAudioTracks_ByBandwidthDescending(t *testing.T) {
	tracks := []entity.Audio{
		{ID: "1", Codecs: "mp4a.40.2", Bandwith: 1000},
		{ID: "2", Codecs: "mp4a.40.2", Bandwith: 3000},
		{ID: "3", Codecs: "mp4a.40.2", Bandwith: 2000},
	}

	result := SortAudioTracks(tracks, nil, false)

	if result[0].Bandwith != 3000 || result[1].Bandwith != 2000 || result[2].Bandwith != 1000 {
		t.Fatalf("expected bandwidth descending, got: %+v", result)
	}
}

func TestSortAudioTracks_Empty(t *testing.T) {
	var tracks []entity.Audio
	result := SortAudioTracks(tracks, nil, false)
	if len(result) != 0 {
		t.Fatalf("expected empty result")
	}
}

func TestSortVideoTracks_Composite(t *testing.T) {
	tracks := []entity.Video{
		{ID: "1", Dfn: "720P", Codecs: "avc1", Bandwith: 3000},
		{ID: "2", Dfn: "1080P", Codecs: "hevc", Bandwith: 2000},
		{ID: "3", Dfn: "1080P", Codecs: "avc1", Bandwith: 1000},
		{ID: "4", Dfn: "720P", Codecs: "hevc", Bandwith: 500},
	}

	dfnPriority := map[string]int{"1080P": 1, "720P": 2}
	encodingPriority := map[string]byte{"avc1": 1, "hevc": 2}

	result := SortVideoTracks(tracks, dfnPriority, encodingPriority, false)

	// Expected: 1080P avc1 (id 3), 1080P hevc (id 2), 720P avc1 (id 1), 720P hevc (id 4)
	expected := []string{"3", "2", "1", "4"}
	for i, exp := range expected {
		if result[i].ID != exp {
			t.Fatalf("position %d: expected id %s, got %s. Full: %+v", i, exp, result[i].ID, result)
		}
	}
}

func TestSortAudioTracks_Composite(t *testing.T) {
	tracks := []entity.Audio{
		{ID: "1", Codecs: "mp4a.40.2", Bandwith: 3000},
		{ID: "2", Codecs: "flac", Bandwith: 2000},
		{ID: "3", Codecs: "mp4a.40.2", Bandwith: 1000},
		{ID: "4", Codecs: "flac", Bandwith: 500},
	}

	encodingPriority := map[string]byte{"MP4A.40.2": 1, "FLAC": 2}

	result := SortAudioTracks(tracks, encodingPriority, false)

	// Expected: mp4a.40.2 3000 (id 1), mp4a.40.2 1000 (id 3), flac 2000 (id 2), flac 500 (id 4)
	expected := []string{"1", "3", "2", "4"}
	for i, exp := range expected {
		if result[i].ID != exp {
			t.Fatalf("position %d: expected id %s, got %s. Full: %+v", i, exp, result[i].ID, result)
		}
	}
}
