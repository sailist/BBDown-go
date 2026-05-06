package danmaku

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComputeTime(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{0.00, "0:00:00.00"},
		{1.50, "0:00:01.50"},
		{61.23, "0:01:01.23"},
		{3661.45, "1:01:01.45"},
		{3600.00, "1:00:00.00"},
	}

	for _, tt := range tests {
		result := computeTime(tt.input)
		if result != tt.expected {
			t.Errorf("computeTime(%.2f) = %s; want %s", tt.input, result, tt.expected)
		}
	}
}

func TestParseXML(t *testing.T) {
	xmlContent := `<i>
  <d p="0.00,1,25,16777215,1234567890,0,abc,0">Hello</d>
  <d p="1.50,5,25,16711680,1234567891,0,def,0">World</d>
  <d p="3.00,4,25,0,1234567892,0,ghi,0">Bottom</d>
</i>`

	tmpDir := t.TempDir()
	xmlPath := filepath.Join(tmpDir, "danmaku.xml")
	if err := os.WriteFile(xmlPath, []byte(xmlContent), 0644); err != nil {
		t.Fatalf("failed to write test xml: %v", err)
	}

	items, err := ParseXML(xmlPath)
	if err != nil {
		t.Fatalf("ParseXML failed: %v", err)
	}

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	// First item: scroll (mode 1)
	if items[0].Content != "Hello" {
		t.Errorf("item[0].Content = %s; want Hello", items[0].Content)
	}
	if items[0].StartTime != "0:00:00.00" {
		t.Errorf("item[0].StartTime = %s; want 0:00:00.00", items[0].StartTime)
	}
	if items[0].EndTime != "0:00:08.00" {
		t.Errorf("item[0].EndTime = %s; want 0:00:08.00", items[0].EndTime)
	}
	if items[0].Second != 0.00 {
		t.Errorf("item[0].Second = %f; want 0.00", items[0].Second)
	}
	if items[0].DanmakuMode != 1 {
		t.Errorf("item[0].DanmakuMode = %d; want 1", items[0].DanmakuMode)
	}
	if items[0].FontSize != "25" {
		t.Errorf("item[0].FontSize = %s; want 25", items[0].FontSize)
	}
	if items[0].Color != "FFFFFF" {
		t.Errorf("item[0].Color = %s; want FFFFFF", items[0].Color)
	}
	if items[0].Timestamp != "1234567890" {
		t.Errorf("item[0].Timestamp = %s; want 1234567890", items[0].Timestamp)
	}

	// Second item: top (mode 2, mapped from 5)
	if items[1].Content != "World" {
		t.Errorf("item[1].Content = %s; want World", items[1].Content)
	}
	if items[1].DanmakuMode != 2 {
		t.Errorf("item[1].DanmakuMode = %d; want 2", items[1].DanmakuMode)
	}
	if items[1].Color != "FF0000" {
		t.Errorf("item[1].Color = %s; want FF0000", items[1].Color)
	}
	if items[1].EndTime != "0:00:05.50" {
		t.Errorf("item[1].EndTime = %s; want 0:00:05.50", items[1].EndTime)
	}

	// Third item: bottom (mode 3, mapped from 4)
	if items[2].Content != "Bottom" {
		t.Errorf("item[2].Content = %s; want Bottom", items[2].Content)
	}
	if items[2].DanmakuMode != 3 {
		t.Errorf("item[2].DanmakuMode = %d; want 3", items[2].DanmakuMode)
	}
	if items[2].Color != "000000" {
		t.Errorf("item[2].Color = %s; want 000000", items[2].Color)
	}
	if items[2].EndTime != "0:00:07.00" {
		t.Errorf("item[2].EndTime = %s; want 0:00:07.00", items[2].EndTime)
	}
}

func TestParseXMLEmpty(t *testing.T) {
	xmlContent := `<i></i>`
	tmpDir := t.TempDir()
	xmlPath := filepath.Join(tmpDir, "empty.xml")
	if err := os.WriteFile(xmlPath, []byte(xmlContent), 0644); err != nil {
		t.Fatalf("failed to write test xml: %v", err)
	}

	items, err := ParseXML(xmlPath)
	if err != nil {
		t.Fatalf("ParseXML failed: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestSaveAsAss(t *testing.T) {
	items := []DanmakuItem{
		{
			Content:     "Hello",
			StartTime:   "0:00:00.00",
			EndTime:     "0:00:08.00",
			Second:      0.0,
			DanmakuMode: 1,
			FontSize:    "25",
			Color:       "FFFFFF",
			Timestamp:   "1234567890",
		},
		{
			Content:     "World",
			StartTime:   "0:00:01.50",
			EndTime:     "0:00:05.50",
			Second:      1.5,
			DanmakuMode: 2,
			FontSize:    "25",
			Color:       "FF0000",
			Timestamp:   "1234567891",
		},
	}

	tmpDir := t.TempDir()
	assPath := filepath.Join(tmpDir, "output.ass")

	if err := SaveAsAss(items, assPath); err != nil {
		t.Fatalf("SaveAsAss failed: %v", err)
	}

	data, err := os.ReadFile(assPath)
	if err != nil {
		t.Fatalf("failed to read ass file: %v", err)
	}
	content := string(data)

	// Verify headers
	if !strings.Contains(content, "[Script Info]") {
		t.Error("missing [Script Info] section")
	}
	if !strings.Contains(content, "PlayResX: 1920") {
		t.Error("missing PlayResX: 1920")
	}
	if !strings.Contains(content, "PlayResY: 1080") {
		t.Error("missing PlayResY: 1080")
	}
	if !strings.Contains(content, "[V4+ Styles]") {
		t.Error("missing [V4+ Styles] section")
	}
	if !strings.Contains(content, "[Events]") {
		t.Error("missing [Events] section")
	}

	// Verify Style line
	if !strings.Contains(content, "Style: BBDOWN_Style, 黑体, 40,") {
		t.Error("missing or incorrect Style line")
	}

	// Verify dialogue lines
	if !strings.Contains(content, "Dialogue: 2,0:00:00.00,0:00:08.00,BBDOWN_Style,,0000,0000,0000,,{\\move(1920, 0, -200, 0)}Hello") {
		t.Error("missing or incorrect scroll dialogue line")
	}
	if !strings.Contains(content, "Dialogue: 2,0:00:01.50,0:00:05.50,BBDOWN_Style,,0000,0000,0000,,{\\an8\\pos(960, 0)\\c&FF0000&}World") {
		t.Error("missing or incorrect top dialogue line with color")
	}
}

func TestPositionController(t *testing.T) {
	pc := NewPositionController()

	// maxLine = 1080 * 50 / 40 / 100 = 13
	if pc.maxLine != 13 {
		t.Errorf("expected maxLine = 13, got %d", pc.maxLine)
	}

	// Assign positions for top mode
	for i := 0; i < pc.maxLine; i++ {
		h := pc.UpdatePosition(2, 0.0, 5)
		if h != i*fontSize {
			t.Errorf("expected height %d, got %d", i*fontSize, h)
		}
	}

	// No more positions available
	h := pc.UpdatePosition(2, 0.0, 5)
	if h != -1 {
		t.Errorf("expected -1 when full, got %d", h)
	}

	// After top spend time, positions should be free
	for i := 0; i < pc.maxLine; i++ {
		h := pc.UpdatePosition(2, 4.01, 5)
		if h != i*fontSize {
			t.Errorf("expected height %d after time elapsed, got %d", i*fontSize, h)
		}
	}
}

func TestPositionControllerMoveMode(t *testing.T) {
	pc := NewPositionController()

	// Move mode should use a different display time calculation
	h1 := pc.UpdatePosition(1, 0.0, 5)
	if h1 != 0 {
		t.Errorf("expected height 0, got %d", h1)
	}

	h2 := pc.UpdatePosition(1, 0.0, 5)
	if h2 != fontSize {
		t.Errorf("expected height %d, got %d", fontSize, h2)
	}
}

func TestPositionControllerMixedModes(t *testing.T) {
	pc := NewPositionController()

	// Top and bottom queues are independent
	hTop := pc.UpdatePosition(2, 0.0, 5)
	hBottom := pc.UpdatePosition(3, 0.0, 5)

	if hTop != 0 {
		t.Errorf("expected top height 0, got %d", hTop)
	}
	if hBottom != 0 {
		t.Errorf("expected bottom height 0, got %d", hBottom)
	}
}
