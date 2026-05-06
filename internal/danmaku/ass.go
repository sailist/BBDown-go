package danmaku

import (
	"fmt"
	"os"
	"sort"
)

// SaveAsAss converts danmaku items to ASS subtitle format and writes to outputPath.
func SaveAsAss(items []DanmakuItem, outputPath string) error {
	// Sort by start time.
	sorted := make([]DanmakuItem, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Second < sorted[j].Second
	})

	var sb []byte
	sb = append(sb, "[Script Info]\n"...)
	sb = append(sb, "Script Updated By: BBDown(https://github.com/nilaoda/BBDown)\n"...)
	sb = append(sb, "ScriptType: v4.00+\n"...)
	sb = append(sb, fmt.Sprintf("PlayResX: %d\n", monitorWidth)...)
	sb = append(sb, fmt.Sprintf("PlayResY: %d\n", monitorHeight)...)
	sb = append(sb, fmt.Sprintf("Aspect Ratio: %d:%d\n", monitorWidth, monitorHeight)...)
	sb = append(sb, "Collisions: Normal\n"...)
	sb = append(sb, "WrapStyle: 2\n"...)
	sb = append(sb, "ScaledBorderAndShadow: yes\n"...)
	sb = append(sb, "YCbCr Matrix: TV.601\n"...)
	sb = append(sb, "[V4+ Styles]\n"...)
	sb = append(sb, "Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n"...)
	sb = append(sb, fmt.Sprintf("Style: BBDOWN_Style, 黑体, %d, &H00FFFFFF, &H00FFFFFF, &H00000000, &H00000000, 0, 0, 0, 0, 100, 100, 0.00, 0.00, 1, 2, 0, 7, 0, 0, 0, 0\n", fontSize)...)
	sb = append(sb, "[Events]\n"...)
	sb = append(sb, "Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n"...)

	controller := NewPositionController()
	for _, item := range sorted {
		height := controller.UpdatePosition(item.DanmakuMode, item.Second, len(item.Content))
		if height == -1 {
			continue
		}

		effect := ""
		switch item.DanmakuMode {
		case 3:
			effect = fmt.Sprintf("\\an8\\pos(%d, %d)", monitorWidth/2, monitorHeight-fontSize-height)
		case 2:
			effect = fmt.Sprintf("\\an8\\pos(%d, %d)", monitorWidth/2, height)
		default:
			effect = fmt.Sprintf("\\move(%d, %d, %d, %d)", monitorWidth, height, -len(item.Content)*fontSize, height)
		}

		if item.Color != "FFFFFF" {
			effect += fmt.Sprintf("\\c&%s&", item.Color)
		}

		line := fmt.Sprintf("Dialogue: 2,%s,%s,BBDOWN_Style,,0000,0000,0000,,{%s}%s\n",
			item.StartTime, item.EndTime, effect, item.Content)
		sb = append(sb, line...)
	}

	if err := os.WriteFile(outputPath, sb, 0644); err != nil {
		return fmt.Errorf("writing ass file: %w", err)
	}
	return nil
}
