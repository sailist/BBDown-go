package danmaku

import (
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	moveSpendTime = 8.00
	topSpendTime  = 4.00
)

// DanmakuItem represents a single danmaku (bullet comment).
type DanmakuItem struct {
	Content     string
	StartTime   string  // ASS format: H:MM:SS.cc
	EndTime     string  // ASS format: H:MM:SS.cc
	Second      float64
	DanmakuMode int     // 1=scroll, 2=top, 3=bottom
	FontSize    string
	Color       string  // hex format like "FFFFFF"
	Timestamp   string
}

// ParseXML parses a Bilibili danmaku XML file and returns a slice of DanmakuItem.
func ParseXML(xmlPath string) ([]DanmakuItem, error) {
	data, err := os.ReadFile(xmlPath)
	if err != nil {
		return nil, fmt.Errorf("reading danmaku xml: %w", err)
	}

	type D struct {
		P       string `xml:"p,attr"`
		Content string `xml:",chardata"`
	}
	type I struct {
		XMLName xml.Name `xml:"i"`
		Ds      []D      `xml:"d"`
	}

	var doc I
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing danmaku xml: %w", err)
	}

	var items []DanmakuItem
	for _, d := range doc.Ds {
		attrs := strings.Split(d.P, ",")
		if len(attrs) < 8 {
			continue
		}

		item := DanmakuItem{}
		item.Content = d.Content

		second, err := strconv.ParseFloat(attrs[0], 64)
		if err != nil {
			continue
		}
		item.Second = second
		item.StartTime = computeTime(second)

		switch attrs[1] {
		case "4":
			item.DanmakuMode = 3 // bottom
		case "5":
			item.DanmakuMode = 2 // top
		default:
			item.DanmakuMode = 1 // scroll/move
		}

		if item.DanmakuMode == 1 {
			item.EndTime = computeTime(second + moveSpendTime)
		} else {
			item.EndTime = computeTime(second + topSpendTime)
		}

		item.FontSize = attrs[2]

		colorD, err := strconv.Atoi(attrs[3])
		if err != nil {
			continue
		}
		item.Color = fmt.Sprintf("%06X", colorD)

		item.Timestamp = attrs[4]
		items = append(items, item)
	}

	return items, nil
}

func computeTime(second float64) string {
	hour := int(second) / 3600
	minute := (int(second) % 3600) / 60
	sec := second - float64(hour*3600+minute*60)
	return fmt.Sprintf("%d:%02d:%05.2f", hour, minute, sec)
}
