package parser

import (
	"testing"

	"github.com/sailist/BBDown-go/internal/core/entity"
)

func TestParseClipInfoList(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    []entity.ViewPoint
		wantErr bool
	}{
		{
			name: "single clip",
			json: `{"clip_info_list":[{"toastText":"即将跳过片头","start":0,"end":85}]}`,
			want: []entity.ViewPoint{
				{Title: "片头", Start: 0, End: 85},
			},
		},
		{
			name: "two clips with main content inserted",
			json: `{"clip_info_list":[` +
				`{"toastText":"即将跳过片头","start":0,"end":85},` +
				`{"toastText":"即将跳过片尾","start":1300,"end":1420}` +
				`]}`,
			want: []entity.ViewPoint{
				{Title: "片头", Start: 0, End: 85},
				{Title: "正片", Start: 85, End: 1300},
				{Title: "片尾", Start: 1300, End: 1420},
			},
		},
		{
			name: "unsorted clips are sorted",
			json: `{"clip_info_list":[` +
				`{"toastText":"即将跳过片尾","start":1300,"end":1420},` +
				`{"toastText":"即将跳过片头","start":0,"end":85}` +
				`]}`,
			want: []entity.ViewPoint{
				{Title: "片头", Start: 0, End: 85},
				{Title: "正片", Start: 85, End: 1300},
				{Title: "片尾", Start: 1300, End: 1420},
			},
		},
		{
			name: "three clips with main content",
			json: `{"clip_info_list":[` +
				`{"toastText":"即将跳过片头","start":0,"end":90},` +
				`{"toastText":"即将跳过OP","start":300,"end":380},` +
				`{"toastText":"即将跳过片尾","start":1200,"end":1320}` +
				`]}`,
			want: []entity.ViewPoint{
				{Title: "片头", Start: 0, End: 90},
				{Title: "正片", Start: 90, End: 300},
				{Title: "OP", Start: 300, End: 380},
				{Title: "正片", Start: 380, End: 1200},
				{Title: "片尾", Start: 1200, End: 1320},
			},
		},
		{
			name: "overlapping clips no duplicate main content",
			json: `{"clip_info_list":[` +
				`{"toastText":"即将跳过片头","start":0,"end":85},` +
				`{"toastText":"即将跳过片尾","start":80,"end":1420}` +
				`]}`,
			want: []entity.ViewPoint{
				{Title: "片头", Start: 0, End: 85},
				{Title: "片尾", Start: 80, End: 1420},
			},
		},
		{
			name:    "empty clip list",
			json:    `{"clip_info_list":[]}`,
			want:    nil,
			wantErr: false,
		},
		{
			name:    "missing clip_info_list",
			json:    `{}`,
			want:    nil,
			wantErr: false,
		},
		{
			name:    "invalid json",
			json:    `{"clip_info_list":`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseClipInfoList(tt.json)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseClipInfoList() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ParseClipInfoList() got %d points, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ParseClipInfoList()[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseViewPoints(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    []entity.ViewPoint
		wantErr bool
	}{
		{
			name: "single viewpoint",
			json: `{"data":{"view_points":[{"content":"开场","from":0,"to":30}]}}`,
			want: []entity.ViewPoint{
				{Title: "开场", Start: 0, End: 30},
			},
		},
		{
			name: "multiple viewpoints",
			json: `{"data":{"view_points":[` +
				`{"content":"开场","from":0,"to":30},` +
				`{"content":"高潮","from":300,"to":450},` +
				`{"content":"结尾","from":1200,"to":1320}` +
				`]}}`,
			want: []entity.ViewPoint{
				{Title: "开场", Start: 0, End: 30},
				{Title: "高潮", Start: 300, End: 450},
				{Title: "结尾", Start: 1200, End: 1320},
			},
		},
		{
			name:    "empty view_points",
			json:    `{"data":{"view_points":[]}}`,
			want:    nil,
			wantErr: false,
		},
		{
			name:    "missing view_points",
			json:    `{"data":{}}`,
			want:    nil,
			wantErr: false,
		},
		{
			name:    "missing data",
			json:    `{}`,
			want:    nil,
			wantErr: false,
		},
		{
			name:    "invalid json",
			json:    `{"data":`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseViewPoints(tt.json)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseViewPoints() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ParseViewPoints() got %d points, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ParseViewPoints()[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
