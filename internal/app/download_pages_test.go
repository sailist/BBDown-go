package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nilaonai/bbdown-go/internal/cli"
	"github.com/nilaonai/bbdown-go/internal/config"
	"github.com/nilaonai/bbdown-go/internal/core/entity"
)

func TestParseSelectPage_Empty(t *testing.T) {
	pages, err := parseSelectPage("", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pages) != 5 {
		t.Fatalf("expected 5 pages, got %d", len(pages))
	}
	for i, p := range pages {
		if p != i+1 {
			t.Errorf("expected page %d, got %d", i+1, p)
		}
	}
}

func TestParseSelectPage_All(t *testing.T) {
	pages, err := parseSelectPage("ALL", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pages) != 3 {
		t.Fatalf("expected 3 pages, got %d", len(pages))
	}
}

func TestParseSelectPage_Single(t *testing.T) {
	pages, err := parseSelectPage("2", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pages) != 1 || pages[0] != 2 {
		t.Errorf("expected [2], got %v", pages)
	}
}

func TestParseSelectPage_List(t *testing.T) {
	pages, err := parseSelectPage("1,3,5", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{1, 3, 5}
	if len(pages) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, pages)
	}
	for i := range expected {
		if pages[i] != expected[i] {
			t.Errorf("expected %v, got %v", expected, pages)
			break
		}
	}
}

func TestParseSelectPage_Range(t *testing.T) {
	pages, err := parseSelectPage("2-4", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{2, 3, 4}
	if len(pages) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, pages)
	}
	for i := range expected {
		if pages[i] != expected[i] {
			t.Errorf("expected %v, got %v", expected, pages)
			break
		}
	}
}

func TestParseSelectPage_Mixed(t *testing.T) {
	pages, err := parseSelectPage("1,3-5", 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{1, 3, 4, 5}
	if len(pages) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, pages)
	}
	for i := range expected {
		if pages[i] != expected[i] {
			t.Errorf("expected %v, got %v", expected, pages)
			break
		}
	}
}

func TestParseSelectPage_Keywords(t *testing.T) {
	pages, err := parseSelectPage("LAST", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pages) != 1 || pages[0] != 5 {
		t.Errorf("expected [5], got %v", pages)
	}

	pages, err = parseSelectPage("1,LATEST", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{1, 5}
	if len(pages) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, pages)
	}
}

func TestParseSelectPage_OutOfBounds(t *testing.T) {
	_, err := parseSelectPage("6", 5)
	if err == nil {
		t.Fatal("expected error for out-of-bounds page")
	}
	_, err = parseSelectPage("0", 5)
	if err == nil {
		t.Fatal("expected error for page 0")
	}
	_, err = parseSelectPage("3-6", 5)
	if err == nil {
		t.Fatal("expected error for out-of-bounds range")
	}
}

func TestParseSelectPage_Invalid(t *testing.T) {
	_, err := parseSelectPage("abc", 5)
	if err == nil {
		t.Fatal("expected error for invalid input")
	}
	_, err = parseSelectPage("1-2-3", 5)
	if err == nil {
		t.Fatal("expected error for invalid range")
	}
}

func TestCheckAidInFile(t *testing.T) {
	tmpDir := t.TempDir()
	origAppDir := appDir
	appDir = tmpDir
	defer func() { appDir = origAppDir }()

	if checkAidInFile("123") {
		t.Error("expected false when file does not exist")
	}

	if err := saveAidToFile("123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !checkAidInFile("123") {
		t.Error("expected true after saving aid")
	}
	if checkAidInFile("456") {
		t.Error("expected false for different aid")
	}

	if err := saveAidToFile("456"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !checkAidInFile("456") {
		t.Error("expected true after saving second aid")
	}
}

func TestDownloadPages_SinglePageSavePath(t *testing.T) {
	opt := cli.NewOption()
	vInfo := &entity.VInfo{
		Title:     "Test",
		PagesInfo: []entity.Page{{Index: 1, Aid: "1"}},
	}
	workCfg := &WorkConfig{Config: config.NewConfig()}

	var capturedFormat string
	deps := DownloadDeps{
		Logger: discardLogger(),
		DownloadPage: func(ctx context.Context, p *entity.Page, opt *cli.Option, vInfo *entity.VInfo, selectedPages []entity.Page, workCfg *WorkConfig, deps DownloadDeps) error {
			capturedFormat = workCfg.SavePathFormat
			return nil
		},
	}

	if err := DownloadPages(context.Background(), opt, vInfo, workCfg, deps); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedFormat != cli.SinglePageDefaultSavePath {
		t.Errorf("expected single page default format, got %q", capturedFormat)
	}
}

func TestDownloadPages_MultiPageSavePath(t *testing.T) {
	opt := cli.NewOption()
	vInfo := &entity.VInfo{
		Title: "Test",
		PagesInfo: []entity.Page{
			{Index: 1, Aid: "1"},
			{Index: 2, Aid: "2"},
		},
	}
	workCfg := &WorkConfig{Config: config.NewConfig()}

	var capturedFormat string
	deps := DownloadDeps{
		Logger: discardLogger(),
		DownloadPage: func(ctx context.Context, p *entity.Page, opt *cli.Option, vInfo *entity.VInfo, selectedPages []entity.Page, workCfg *WorkConfig, deps DownloadDeps) error {
			capturedFormat = workCfg.SavePathFormat
			return nil
		},
	}

	if err := DownloadPages(context.Background(), opt, vInfo, workCfg, deps); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedFormat != cli.MultiPageDefaultSavePath {
		t.Errorf("expected multi page default format, got %q", capturedFormat)
	}
}

func TestDownloadPages_BangumiNotEndSavePath(t *testing.T) {
	opt := cli.NewOption()
	vInfo := &entity.VInfo{
		Title:        "Test",
		IsBangumi:    true,
		IsBangumiEnd: false,
		PagesInfo:    []entity.Page{{Index: 1, Aid: "1"}},
	}
	workCfg := &WorkConfig{Config: config.NewConfig()}

	var capturedFormat string
	deps := DownloadDeps{
		Logger: discardLogger(),
		DownloadPage: func(ctx context.Context, p *entity.Page, opt *cli.Option, vInfo *entity.VInfo, selectedPages []entity.Page, workCfg *WorkConfig, deps DownloadDeps) error {
			capturedFormat = workCfg.SavePathFormat
			return nil
		},
	}

	if err := DownloadPages(context.Background(), opt, vInfo, workCfg, deps); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedFormat != cli.MultiPageDefaultSavePath {
		t.Errorf("expected multi page default format for ongoing bangumi, got %q", capturedFormat)
	}
}

func TestDownloadPages_CustomSavePath(t *testing.T) {
	opt := cli.NewOption()
	opt.FilePattern = "custom_single"
	opt.MultiFilePattern = "custom_multi"
	vInfo := &entity.VInfo{
		Title: "Test",
		PagesInfo: []entity.Page{
			{Index: 1, Aid: "1"},
			{Index: 2, Aid: "2"},
		},
	}
	workCfg := &WorkConfig{Config: config.NewConfig()}

	var capturedFormat string
	deps := DownloadDeps{
		Logger: discardLogger(),
		DownloadPage: func(ctx context.Context, p *entity.Page, opt *cli.Option, vInfo *entity.VInfo, selectedPages []entity.Page, workCfg *WorkConfig, deps DownloadDeps) error {
			capturedFormat = workCfg.SavePathFormat
			return nil
		},
	}

	if err := DownloadPages(context.Background(), opt, vInfo, workCfg, deps); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedFormat != "custom_multi" {
		t.Errorf("expected custom multi format, got %q", capturedFormat)
	}
}

func TestDownloadPages_PageFiltering(t *testing.T) {
	opt := cli.NewOption()
	opt.SelectPage = "1,3-4"
	vInfo := &entity.VInfo{
		Title: "Test",
		PagesInfo: []entity.Page{
			{Index: 1, Aid: "1"},
			{Index: 2, Aid: "2"},
			{Index: 3, Aid: "3"},
			{Index: 4, Aid: "4"},
			{Index: 5, Aid: "5"},
		},
	}
	workCfg := &WorkConfig{Config: config.NewConfig()}

	var downloaded []int
	deps := DownloadDeps{
		Logger: discardLogger(),
		DownloadPage: func(ctx context.Context, p *entity.Page, opt *cli.Option, vInfo *entity.VInfo, selectedPages []entity.Page, workCfg *WorkConfig, deps DownloadDeps) error {
			downloaded = append(downloaded, p.Index)
			return nil
		},
	}

	if err := DownloadPages(context.Background(), opt, vInfo, workCfg, deps); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []int{1, 3, 4}
	if len(downloaded) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, downloaded)
	}
	for i := range expected {
		if downloaded[i] != expected[i] {
			t.Errorf("expected %v, got %v", expected, downloaded)
			break
		}
	}
}

func TestDownloadPages_ArchiveDedup(t *testing.T) {
	tmpDir := t.TempDir()
	origAppDir := appDir
	appDir = tmpDir
	defer func() { appDir = origAppDir }()

	if err := os.WriteFile(filepath.Join(tmpDir, "BBDown.archives"), []byte("1|"), 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	opt := cli.NewOption()
	opt.SaveArchivesToFile = true
	vInfo := &entity.VInfo{
		Title: "Test",
		PagesInfo: []entity.Page{
			{Index: 1, Aid: "1"},
			{Index: 2, Aid: "2"},
		},
	}
	workCfg := &WorkConfig{Config: config.NewConfig()}

	var downloaded []string
	deps := DownloadDeps{
		Logger: discardLogger(),
		DownloadPage: func(ctx context.Context, p *entity.Page, opt *cli.Option, vInfo *entity.VInfo, selectedPages []entity.Page, workCfg *WorkConfig, deps DownloadDeps) error {
			downloaded = append(downloaded, p.Aid)
			return nil
		},
	}

	if err := DownloadPages(context.Background(), opt, vInfo, workCfg, deps); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(downloaded) != 1 || downloaded[0] != "2" {
		t.Errorf("expected only page 2 to be downloaded, got %v", downloaded)
	}

	if !checkAidInFile("2") {
		t.Error("expected aid 2 to be saved to archive")
	}
}

func TestDownloadPages_DelayBetweenPages(t *testing.T) {
	opt := cli.NewOption()
	vInfo := &entity.VInfo{
		Title: "Test",
		PagesInfo: []entity.Page{
			{Index: 1, Aid: "1"},
			{Index: 2, Aid: "2"},
		},
	}
	workCfg := &WorkConfig{Config: config.NewConfig(), Delay: 1}

	var sleeps []time.Duration
	origSleep := sleepFunc
	sleepFunc = func(d time.Duration) {
		sleeps = append(sleeps, d)
	}
	defer func() { sleepFunc = origSleep }()

	deps := DownloadDeps{
		Logger: discardLogger(),
		DownloadPage: func(ctx context.Context, p *entity.Page, opt *cli.Option, vInfo *entity.VInfo, selectedPages []entity.Page, workCfg *WorkConfig, deps DownloadDeps) error {
			return nil
		},
	}

	if err := DownloadPages(context.Background(), opt, vInfo, workCfg, deps); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sleeps) != 2 {
		t.Fatalf("expected 2 sleeps, got %d", len(sleeps))
	}
	for i, d := range sleeps {
		if d != 1*time.Second {
			t.Errorf("expected sleep 1s at call %d, got %v", i, d)
		}
	}
}
