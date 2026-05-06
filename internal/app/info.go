package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nilaonai/bbdown-go/internal/cli"
	"github.com/nilaonai/bbdown-go/internal/core/entity"
	"github.com/nilaonai/bbdown-go/internal/core/fetcher"
	"github.com/nilaonai/bbdown-go/internal/core/util"
	"github.com/nilaonai/bbdown-go/pkg/bvconv"
	"github.com/nilaonai/bbdown-go/pkg/httpclient"
)

// Deps holds dependencies for GetVideoInfo, enabling test mocking.
type Deps struct {
	FetcherFactory func(id string, useIntl bool) (fetcher.Fetcher, error)
	HTTPClient     httpclient.Client
	Logger         *slog.Logger
}

// GetVideoInfo resolves input to video metadata.
func GetVideoInfo(ctx context.Context, opt *cli.Option, input string, deps Deps) (fetchedAid string, vInfo *entity.VInfo, apiType string, err error) {
	// 1. Load credentials
	cookie := loadCookie(opt)

	// 2. Check login
	if !opt.UseTvApi && !opt.UseIntlApi && opt.Area == "" {
		deps.Logger.Info("检测账号登录...")
		isLogin, wbiKey, checkErr := CheckLogin(ctx, deps.HTTPClient, cookie)
		if checkErr != nil {
			deps.Logger.Warn("检测登录状态失败", "error", checkErr)
		} else if !isLogin {
			deps.Logger.Warn("你尚未登录B站账号, 解析可能受到限制")
		}
		if wbiKey != "" {
			deps.Logger.Debug("wbi key generated", "key", wbiKey)
		}
	}

	// 3. Parse input
	deps.Logger.Info("获取aid...")
	aidOri, err := ParseInput(ctx, deps.HTTPClient, input)
	if err != nil {
		return "", nil, "", fmt.Errorf("解析输入失败: %w", err)
	}
	deps.Logger.Info("获取aid结束", "aid", aidOri)
	if aidOri == "" {
		return "", nil, "", errors.New("输入有误")
	}

	// 4. Fetch video info
	deps.Logger.Info("获取视频信息...")
	f, err := deps.FetcherFactory(aidOri, opt.UseIntlApi)
	if err != nil {
		return "", nil, "", fmt.Errorf("创建 fetcher 失败: %w", err)
	}

	vInfo, err = f.Fetch(ctx, aidOri)
	if err != nil {
		if errors.Is(err, fetcher.ErrKeyNotFound) && !strings.HasPrefix(aidOri, "cheese:") {
			deps.Logger.Warn("未找到此 EP/SS 对应番剧信息, 正在尝试按课程查找。")
			aidOri = strings.Replace(aidOri, "ep:", "cheese:", 1)
			deps.Logger.Info("新的 aid", "aid", aidOri)
			if aidOri == "" {
				return "", nil, "", errors.New("输入有误")
			}
			f, err = deps.FetcherFactory(aidOri, opt.UseIntlApi)
			if err != nil {
				return "", nil, "", fmt.Errorf("创建 fetcher 失败: %w", err)
			}
			vInfo, err = f.Fetch(ctx, aidOri)
			if err != nil {
				return "", nil, "", err
			}
		} else {
			return "", nil, "", err
		}
	}

	// 7. Interactive video check (before apiType to match C# behavior)
	if vInfo.IsSteinGate && opt.UseTvApi {
		deps.Logger.Warn("视频为互动视频，暂时不支持tv下载，修改为默认下载")
		opt.UseTvApi = false
	}

	// 6. Determine API type
	switch {
	case opt.UseTvApi:
		apiType = "TV"
	case opt.UseAppApi:
		apiType = "APP"
	case opt.UseIntlApi:
		apiType = "INTL"
	default:
		apiType = "WEB"
	}

	// 8. Log video info
	bvURL := ""
	if len(vInfo.PagesInfo) > 0 && !opt.UseIntlApi {
		aid, parseErr := strconv.ParseInt(vInfo.PagesInfo[0].Aid, 10, 64)
		if parseErr == nil && aid > 0 {
			bvURL = fmt.Sprintf("https://www.bilibili.com/video/%s/", bvconv.Encode(aid))
		}
	}
	upMid := ""
	for _, p := range vInfo.PagesInfo {
		if p.OwnerMid != "" {
			upMid = p.OwnerMid
			break
		}
	}

	deps.Logger.Info("视频标题: " + vInfo.Title)
	if vInfo.PubTime != 0 {
		deps.Logger.Info("发布时间", "pubTime", vInfo.PubTime)
	}
	if bvURL != "" {
		deps.Logger.Info("视频URL: " + bvURL)
	}
	if upMid != "" {
		deps.Logger.Info("UP主页: https://space.bilibili.com/" + upMid)
	}

	// 9. Print pages
	printPages(deps.Logger, vInfo.PagesInfo, opt.ShowAll)

	return aidOri, vInfo, apiType, nil
}

func printPages(logger *slog.Logger, pages []entity.Page, showAll bool) {
	if len(pages) == 0 {
		return
	}
	more := false
	for _, p := range pages {
		if !showAll {
			if more && p.Index != len(pages) {
				continue
			}
			if !more && p.Index > 5 {
				logger.Info("......")
				more = true
				continue
			}
		}
		logger.Info(fmt.Sprintf("P%d: [%s] [%s] [%s]", p.Index, p.Cid, p.Title, util.FormatTime(p.Dur, false)))
	}
}

// CheckLogin checks login status and derives the WBI mixin key.
func CheckLogin(ctx context.Context, client httpclient.Client, cookie string) (bool, string, error) {
	api := "https://api.bilibili.com/x/web-interface/nav"
	var opts []httpclient.RequestOption
	if cookie != "" {
		opts = append(opts, httpclient.WithCookie(cookie))
	}
	resp, err := client.Get(ctx, api, opts...)
	if err != nil {
		return false, "", fmt.Errorf("check login request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, "", fmt.Errorf("read check login response: %w", err)
	}

	var result struct {
		Data struct {
			IsLogin bool `json:"isLogin"`
			WbiImg  struct {
				ImgURL string `json:"img_url"`
				SubURL string `json:"sub_url"`
			} `json:"wbi_img"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, "", fmt.Errorf("parse check login response: %w", err)
	}

	isLogin := result.Data.IsLogin
	wbiKey := ""
	if result.Data.WbiImg.ImgURL != "" && result.Data.WbiImg.SubURL != "" {
		imgKey := extractFilename(result.Data.WbiImg.ImgURL)
		subKey := extractFilename(result.Data.WbiImg.SubURL)
		wbiKey = util.GetMixinKey(imgKey + subKey)
	}

	return isLogin, wbiKey, nil
}

func extractFilename(url string) string {
	lastSlash := strings.LastIndex(url, "/")
	if lastSlash == -1 {
		return ""
	}
	s := url[lastSlash+1:]
	lastDot := strings.LastIndex(s, ".")
	if lastDot == -1 {
		return s
	}
	return s[:lastDot]
}

func loadCookie(opt *cli.Option) string {
	if opt.Cookie != "" {
		return opt.Cookie
	}
	cookiePath := filepath.Join(appDir, "BBDown.data")
	data, err := os.ReadFile(cookiePath)
	if err == nil {
		return string(data)
	}
	return ""
}
