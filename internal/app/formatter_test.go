package app

import (
	"strings"
	"testing"

	"github.com/sailist/BBDown-go/internal/core/entity"
)

func TestFormatSavePath_BasicPlaceholders(t *testing.T) {
	page := entity.Page{
		Index:     3,
		Aid:       "2",
		Cid:       "123456",
		Title:     "Test Title",
		OwnerName: "TestOwner",
		OwnerMid:  "789",
	}
	video := &entity.Video{
		ID:      "100",
		Dfn:     "1080P",
		Res:     "1920x1080",
		Fps:     "60",
		Codecs:  "avc1.640032",
		Bandwith: 5000000,
	}
	audio := &entity.Audio{
		ID:       "200",
		Codecs:   "mp4a.40.2",
		Bandwith: 128000,
	}

	format := "<videoTitle>_<pageNumber>_<pageNumberWithZero>_<pageTitle>_<bvid>_<aid>_<cid>_<ownerName>_<ownerMid>_<dfn>_<res>_<fps>_<videoCodecs>_<videoBandwidth>_<audioCodecs>_<audioBandwidth>_<apiType>"
	result := FormatSavePath(format, "My Video", video, audio, page, 10, "web", 1609459200)

	expected := "My Video_3_03_Test Title_BV1xx411c7mD_2_123456_TestOwner_789_1080P_1920x1080_60_avc1.640032_5000000_mp4a.40.2_128000_web.mp4"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatSavePath_PublishDate(t *testing.T) {
	page := entity.Page{Index: 1}
	video := &entity.Video{Dfn: "720P"}

	format := "<videoTitle>_<publishDate>"
	result := FormatSavePath(format, "Test", video, nil, page, 1, "web", 1609459200)

	expected := "Test_2021-01-01_00-00-00.mp4"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatSavePath_VideoDate(t *testing.T) {
	page := entity.Page{Index: 1, PubTime: 1609459200}
	video := &entity.Video{Dfn: "720P"}

	format := "<videoTitle>_<videoDate>"
	result := FormatSavePath(format, "Test", video, nil, page, 1, "web", 0)

	expected := "Test_2021-01-01_00-00-00.mp4"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatSavePath_CustomDateFormat(t *testing.T) {
	page := entity.Page{Index: 1}
	video := &entity.Video{Dfn: "720P"}

	format := "<videoTitle>_<publishDate:2006-01-02>"
	result := FormatSavePath(format, "Test", video, nil, page, 1, "web", 1609459200)

	expected := "Test_2021-01-01.mp4"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatSavePath_ZeroTimestamp(t *testing.T) {
	page := entity.Page{Index: 1}
	video := &entity.Video{Dfn: "720P"}

	format := "<videoTitle>_<publishDate>_<videoDate>"
	result := FormatSavePath(format, "Test", video, nil, page, 1, "web", 0)

	expected := "Test_null_null.mp4"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatSavePath_DefaultSuffix(t *testing.T) {
	page := entity.Page{Index: 1}
	result := FormatSavePath("output", "Title", nil, nil, page, 1, "web", 0)
	if !strings.HasSuffix(result, ".mp4") {
		t.Fatalf("expected .mp4 suffix, got %q", result)
	}
}

func TestFormatSavePath_PreserveExistingSuffix(t *testing.T) {
	page := entity.Page{Index: 1}
	result := FormatSavePath("output.mp4", "Title", nil, nil, page, 1, "web", 0)
	if result != "output.mp4" {
		t.Fatalf("expected 'output.mp4', got %q", result)
	}
}

func TestFormatSavePath_NonMp4SuffixAppended(t *testing.T) {
	page := entity.Page{Index: 1}
	result := FormatSavePath("output.mkv", "Title", nil, nil, page, 1, "web", 0)
	if result != "output.mkv.mp4" {
		t.Fatalf("expected 'output.mkv.mp4', got %q", result)
	}
}

func TestFormatSavePath_UnknownPlaceholder(t *testing.T) {
	page := entity.Page{Index: 1}
	result := FormatSavePath("<unknown>", "Title", nil, nil, page, 1, "web", 0)
	expected := "<unknown>.mp4"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatSavePath_NoPlaceholders(t *testing.T) {
	page := entity.Page{Index: 1}
	result := FormatSavePath("static_path", "Title", nil, nil, page, 1, "web", 0)
	expected := "static_path.mp4"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatSavePath_BackslashToSlash(t *testing.T) {
	page := entity.Page{Index: 1}
	result := FormatSavePath("dir\\file", "Title", nil, nil, page, 1, "web", 0)
	expected := "dir/file.mp4"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatSavePath_NilVideo(t *testing.T) {
	page := entity.Page{Index: 1}
	result := FormatSavePath("<dfn>_<res>_<fps>_<videoCodecs>_<videoBandwidth>", "Title", nil, nil, page, 1, "web", 0)
	expected := "____.mp4"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatSavePath_NilAudio(t *testing.T) {
	page := entity.Page{Index: 1}
	result := FormatSavePath("<audioCodecs>_<audioBandwidth>", "Title", nil, nil, page, 1, "web", 0)
	expected := "_.mp4"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatSavePath_PageNumberWithZero(t *testing.T) {
	page := entity.Page{Index: 5}
	result := FormatSavePath("<pageNumberWithZero>", "Title", nil, nil, page, 100, "web", 0)
	expected := "005.mp4"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatSavePath_PageNumberWithZeroSinglePage(t *testing.T) {
	page := entity.Page{Index: 1}
	result := FormatSavePath("<pageNumberWithZero>", "Title", nil, nil, page, 1, "web", 0)
	expected := "1.mp4"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatSavePath_TitleTrimming(t *testing.T) {
	page := entity.Page{Index: 1, Title: "  Hello World...  "}
	result := FormatSavePath("<pageTitle>", "Title", nil, nil, page, 1, "web", 0)
	expected := "Hello World....mp4"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatSavePath_OwnerNameTrimming(t *testing.T) {
	page := entity.Page{Index: 1, OwnerName: "  Owner Name...  "}
	result := FormatSavePath("<ownerName>", "Title", nil, nil, page, 1, "web", 0)
	expected := "Owner Name....mp4"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatSavePath_MultipleSamePlaceholder(t *testing.T) {
	page := entity.Page{Index: 1}
	video := &entity.Video{Dfn: "1080P"}
	result := FormatSavePath("<dfn>_<dfn>", "Title", video, nil, page, 1, "web", 0)
	expected := "1080P_1080P.mp4"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatSavePath_EmptyFormat(t *testing.T) {
	page := entity.Page{Index: 1}
	result := FormatSavePath("", "Title", nil, nil, page, 1, "web", 0)
	expected := ".mp4"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}
