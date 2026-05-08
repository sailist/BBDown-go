package api

import (
	"strings"
	"testing"

	"github.com/sailist/BBDown-go/internal/core/api/proto/Response"
)

func TestPackMessageRoundTrip(t *testing.T) {
	original := []byte("hello world, this is a test payload for gRPC packaging")

	packed, err := PackMessage(original)
	if err != nil {
		t.Fatalf("PackMessage failed: %v", err)
	}
	if len(packed) < 6 {
		t.Fatalf("packed data too short: %d", len(packed))
	}
	if packed[0] != 1 {
		t.Fatalf("expected flag 1, got %d", packed[0])
	}

	unpacked, err := ReadMessage(packed)
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	if string(unpacked) != string(original) {
		t.Fatalf("roundtrip mismatch: got %q, want %q", string(unpacked), string(original))
	}
}

func TestReadMessageUncompressed(t *testing.T) {
	payload := []byte("uncompressed payload")
	data := make([]byte, 5+len(payload))
	data[0] = 0
	data[1] = 0
	data[2] = 0
	data[3] = 0
	data[4] = byte(len(payload))
	copy(data[5:], payload)

	result, err := ReadMessage(data)
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	if string(result) != string(payload) {
		t.Fatalf("unexpected payload: got %q, want %q", string(result), string(payload))
	}
}

func TestReadMessageTooShort(t *testing.T) {
	_, err := ReadMessage([]byte{0x01, 0x00})
	if err == nil {
		t.Fatal("expected error for short data")
	}
}

func TestBuildGRPCHeaders(t *testing.T) {
	h := BuildGRPCHeaders("testtoken")
	if h["Host"] != "grpc.biliapi.net" {
		t.Errorf("unexpected Host: %s", h["Host"])
	}
	if !strings.Contains(h["authorization"], "testtoken") {
		t.Errorf("authorization missing token: %s", h["authorization"])
	}
	if h["grpc-encoding"] != "gzip" {
		t.Errorf("unexpected grpc-encoding: %s", h["grpc-encoding"])
	}
}

func TestConvertToDashJson(t *testing.T) {
	quality := uint32(112)
	baseURL := "https://example.com/video.m4s"
	bandwidth := uint32(1000000)
	codecid := uint32(7)
	timelength := uint64(120000)
	format := "hdflv2"

	resp := &Response.PlayViewReply{
		VideoInfo: &Response.VideoInfo{
			Timelength: &timelength,
			Format:     &format,
			Quality:    &quality,
			StreamList: []*Response.StreamItem{
				{
					StreamInfo: &Response.StreamInfo{
						Quality: &quality,
						Format:  &format,
					},
					DashVideo: &Response.DashVideo{
						BaseUrl:   &baseURL,
						BackupUrl: []string{"https://backup.example.com/video.m4s"},
						Bandwidth: &bandwidth,
						Codecid:   &codecid,
					},
				},
			},
			DashAudio: []*Response.DashItem{
				{
					Id:        func() *uint32 { u := uint32(30280); return &u }(),
					BaseUrl:   func() *string { s := "https://example.com/audio.m4s"; return &s }(),
					Bandwidth: func() *uint32 { u := uint32(128000); return &u }(),
				},
			},
		},
		Business: &Response.BusinessInfo{
			ClipInfo: []*Response.ClipInfo{
				{
					Start:     func() *int32 { i := int32(0); return &i }(),
					End:       func() *int32 { i := int32(5000); return &i }(),
					ToastText: func() *string { s := "skip intro"; return &s }(),
				},
			},
		},
	}

	jsonStr, err := ConvertToDashJson(resp)
	if err != nil {
		t.Fatalf("ConvertToDashJson failed: %v", err)
	}

	if !strings.Contains(jsonStr, `"code":0`) {
		t.Errorf("expected code 0 in output")
	}
	if !strings.Contains(jsonStr, `"timelength":120000`) {
		t.Errorf("expected timelength 120000 in output")
	}
	if !strings.Contains(jsonStr, `"base_url"`) {
		t.Errorf("expected base_url in output")
	}
	if !strings.Contains(jsonStr, `"video"`) {
		t.Errorf("expected video array in output")
	}
	if !strings.Contains(jsonStr, `"audio"`) {
		t.Errorf("expected audio array in output")
	}
	if !strings.Contains(jsonStr, `"clip_info_list"`) {
		t.Errorf("expected clip_info_list in output")
	}
	if !strings.Contains(jsonStr, `"dubbing_info"`) {
		t.Errorf("expected dubbing_info in output")
	}
}

func TestConvertToDashJsonNil(t *testing.T) {
	_, err := ConvertToDashJson(nil)
	if err == nil {
		t.Fatal("expected error for nil input")
	}
}
