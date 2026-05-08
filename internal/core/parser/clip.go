package parser

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/sailist/BBDown-go/internal/core/entity"
)

// clipInfoItem represents a single item in the clip_info_list array.
type clipInfoItem struct {
	ToastText string `json:"toastText"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
}

// viewpointItem represents a single item in the view_points array.
type viewpointItem struct {
	Content string `json:"content"`
	From    int    `json:"from"`
	To      int    `json:"to"`
}

// ParseClipInfoList parses a JSON string containing clip_info_list, processes the
// clips by stripping the "即将跳过" prefix from titles, sorting them, and inserting
// "正片" (main content) segments between clips to create continuous coverage.
func ParseClipInfoList(jsonStr string) ([]entity.ViewPoint, error) {
	var root struct {
		ClipInfoList []clipInfoItem `json:"clip_info_list"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &root); err != nil {
		return nil, fmt.Errorf("parse clip info: %w", err)
	}

	if len(root.ClipInfoList) == 0 {
		return nil, nil
	}

	points := make([]entity.ViewPoint, 0, len(root.ClipInfoList))
	for _, clip := range root.ClipInfoList {
		title := strings.Replace(clip.ToastText, "即将跳过", "", 1)
		points = append(points, entity.ViewPoint{
			Title: title,
			Start: clip.Start,
			End:   clip.End,
		})
	}

	sort.Slice(points, func(i, j int) bool {
		return points[i].Start < points[j].Start
	})

	// Insert "正片" segments between clips to create continuous coverage.
	result := make([]entity.ViewPoint, 0, len(points)*2+1)
	lastEnd := 0
	for _, point := range points {
		if lastEnd < point.Start {
			result = append(result, entity.ViewPoint{
				Title: "正片",
				Start: lastEnd,
				End:   point.Start,
			})
		}
		result = append(result, point)
		lastEnd = point.End
	}

	return result, nil
}

// ParseViewPoints parses a JSON response from the player/wbi/v2 API and extracts
// the view_points array into a slice of ViewPoint.
func ParseViewPoints(jsonStr string) ([]entity.ViewPoint, error) {
	var root struct {
		Data struct {
			ViewPoints []viewpointItem `json:"view_points"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &root); err != nil {
		return nil, fmt.Errorf("parse viewpoints: %w", err)
	}

	if len(root.Data.ViewPoints) == 0 {
		return nil, nil
	}

	points := make([]entity.ViewPoint, 0, len(root.Data.ViewPoints))
	for _, vp := range root.Data.ViewPoints {
		points = append(points, entity.ViewPoint{
			Title: vp.Content,
			Start: vp.From,
			End:   vp.To,
		})
	}

	return points, nil
}
