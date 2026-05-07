package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nilaonai/bbdown-go/internal/cli"
	"github.com/nilaonai/bbdown-go/internal/core/entity"
)

var sleepFunc = time.Sleep

var archiveMutex sync.Mutex

func parseSelectPage(s string, total int) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return allPages(total), nil
	}

	upper := strings.ToUpper(s)
	upper = strings.Trim(upper, ",")
	if upper == "ALL" {
		return allPages(total), nil
	}

	lastPage := strconv.Itoa(total)
	for _, key := range []string{"LAST", "NEW", "LATEST"} {
		upper = strings.ReplaceAll(upper, key, lastPage)
	}

	selected := make(map[int]struct{})
	parts := strings.Split(upper, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			rng := strings.Split(part, "-")
			if len(rng) != 2 {
				return nil, fmt.Errorf("invalid range: %s", part)
			}
			start, err := strconv.Atoi(strings.TrimSpace(rng[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid range start: %s", part)
			}
			end, err := strconv.Atoi(strings.TrimSpace(rng[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid range end: %s", part)
			}
			if start < 1 || end > total || start > end {
				return nil, fmt.Errorf("range out of bounds: %s", part)
			}
			for i := start; i <= end; i++ {
				selected[i] = struct{}{}
			}
		} else {
			idx, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid page number: %s", part)
			}
			if idx < 1 || idx > total {
				return nil, fmt.Errorf("page number out of bounds: %d", idx)
			}
			selected[idx] = struct{}{}
		}
	}

	result := make([]int, 0, len(selected))
	for k := range selected {
		result = append(result, k)
	}
	sort.Ints(result)
	return result, nil
}

func allPages(total int) []int {
	pages := make([]int, total)
	for i := 0; i < total; i++ {
		pages[i] = i + 1
	}
	return pages
}

func checkAidInFile(aid string) bool {
	filePath := filepath.Join(appDir, "BBDown.archives")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
	items := strings.Split(string(data), "|")
	for _, item := range items {
		if item == aid {
			return true
		}
	}
	return false
}

func saveAidToFile(aid string) error {
	archiveMutex.Lock()
	defer archiveMutex.Unlock()
	filePath := filepath.Join(appDir, "BBDown.archives")
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(aid + "|")
	return err
}

func DownloadPages(ctx context.Context, opt *cli.Option, vInfo *entity.VInfo, workCfg *WorkConfig, deps DownloadDeps) error {
	pagesInfo := vInfo.PagesInfo
	totalCount := len(pagesInfo)

	selectedIndices, err := parseSelectPage(opt.SelectPage, totalCount)
	if err != nil {
		return fmt.Errorf("parse select page: %w", err)
	}

	indexSet := make(map[int]struct{}, len(selectedIndices))
	for _, idx := range selectedIndices {
		indexSet[idx] = struct{}{}
	}

	selectedPages := make([]entity.Page, 0, len(selectedIndices))
	for _, p := range pagesInfo {
		if _, ok := indexSet[p.Index]; ok {
			selectedPages = append(selectedPages, p)
		}
	}

	selectedStr := "ALL"
	if opt.SelectPage != "" {
		selectedStr = opt.SelectPage
	}
	deps.Logger.Info(fmt.Sprintf("共计 %d 个分P, 已选择：%s", totalCount, selectedStr))

	savePathFormat := opt.FilePattern
	if savePathFormat == "" {
		savePathFormat = cli.SinglePageDefaultSavePath
	}
	if totalCount > 1 || (vInfo.IsBangumi && !vInfo.IsBangumiEnd) {
		savePathFormat = opt.MultiFilePattern
		if savePathFormat == "" {
			savePathFormat = cli.MultiPageDefaultSavePath
		}
	}
	workCfg.SavePathFormat = savePathFormat

	downloadPage := deps.DownloadPage
	if downloadPage == nil {
		downloadPage = DownloadPage
	}

	for i := range selectedPages {
		p := &selectedPages[i]

		if len(selectedPages) > 1 && workCfg.Delay > 0 {
			deps.Logger.Info(fmt.Sprintf("停顿%d秒...", workCfg.Delay))
			sleepFunc(time.Duration(workCfg.Delay) * time.Second)
		}

		deps.Logger.Info(fmt.Sprintf("开始解析P%d: %s... (%d of %d)", p.Index, p.Aid, i+1, len(selectedPages)))

		if opt.SaveArchivesToFile {
			if checkAidInFile(p.Aid) {
				deps.Logger.Info(fmt.Sprintf("aid: %s已下载过, 跳过下载...", p.Aid))
				continue
			}
		}

		if err := downloadPage(ctx, p, opt, vInfo, selectedPages, workCfg, deps); err != nil {
			return fmt.Errorf("download page %d: %w", p.Index, err)
		}

		if opt.SaveArchivesToFile {
			if err := saveAidToFile(p.Aid); err != nil {
				deps.Logger.Warn("保存归档文件失败", "error", err)
			}
		}
	}

	deps.Logger.Info("任务完成")
	return nil
}
