package app

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nilaonai/bbdown-go/internal/cli"
	"github.com/nilaonai/bbdown-go/internal/config"
	"github.com/nilaonai/bbdown-go/internal/muxer"
)

// WorkConfig holds the result of SetupWork.
type WorkConfig struct {
	EncodingPriority       map[string]byte
	DfnPriority            map[string]int
	FirstEncoding          string
	DownloadDanmaku        bool
	DownloadDanmakuFormats []string
	Input                  string
	SavePathFormat         string
	Lang                   string
	AidOri                 string
	Delay                  int
	Config                 *config.Config
}

// appDir is the directory used for credential files. Override for tests.
var appDir = func() string {
	ex, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(ex)
}()

// SetupWork performs all pre-download setup and validation.
func SetupWork(opt *cli.Option, logger *slog.Logger) (*WorkConfig, error) {
	handleDeprecatedOptions(opt, logger)
	handleConflictingOptions(opt)

	if err := findBinaries(opt); err != nil {
		return nil, err
	}

	if err := changeWorkingDir(opt, logger); err != nil {
		return nil, err
	}

	encodingPriority, firstEncoding := ParseEncodingPriority(opt.EncodingPriority)
	dfnPriority := ParseDfnPriority(opt.DfnPriority)

	downloadDanmaku := opt.DownloadDanmaku || opt.DanmakuOnly
	downloadDanmakuFormats := parseDownloadDanmakuFormats(opt.DownloadDanmakuFormats)

	input := opt.URL
	savePathFormat := opt.FilePattern
	if savePathFormat == "" {
		savePathFormat = cli.SinglePageDefaultSavePath
	}

	lang := opt.Language

	delay, _ := strconv.Atoi(opt.DelayPerPage)

	cfg := config.NewConfig()
	cfg.DebugLog = opt.Debug
	cfg.Host = opt.Host
	cfg.EpHost = opt.EpHost
	cfg.TvHost = opt.TvHost
	cfg.Area = opt.Area

	loadCredentials(opt, cfg, logger)

	if opt.AccessToken != "" {
		cfg.Token = strings.Replace(opt.AccessToken, "access_token=", "", 1)
	}

	return &WorkConfig{
		EncodingPriority:       encodingPriority,
		DfnPriority:            dfnPriority,
		FirstEncoding:          firstEncoding,
		DownloadDanmaku:        downloadDanmaku,
		DownloadDanmakuFormats: downloadDanmakuFormats,
		Input:                  input,
		SavePathFormat:         savePathFormat,
		Lang:                   lang,
		AidOri:                 "",
		Delay:                  delay,
		Config:                 cfg,
	}, nil
}

func handleDeprecatedOptions(opt *cli.Option, logger *slog.Logger) {
	if opt.AddDfnSubfix {
		logger.Warn("--add-dfn-subfix is deprecated, use --file-pattern/-F or --multi-file-pattern/-M to customize output filename")
		if opt.FilePattern == "" && opt.MultiFilePattern == "" {
			opt.FilePattern = cli.SinglePageDefaultSavePath + "[<dfn>]"
			opt.MultiFilePattern = cli.MultiPageDefaultSavePath + "[<dfn>]"
			logger.Warn("switched to -F " + opt.FilePattern + " -M " + opt.MultiFilePattern)
		}
	}
	if opt.Aria2cProxy != "" {
		logger.Warn("--aria2c-proxy is deprecated, use --aria2c-args to set aria2c proxy, adding proxy to aria2c args for this run")
		opt.Aria2cArgs += fmt.Sprintf(" --all-proxy=\"%s\"", opt.Aria2cProxy)
	}
	if opt.OnlyHevc {
		logger.Warn("--only-hevc/-hevc is deprecated, use --encoding-priority to set encoding priority, setting hevc as highest priority for this run")
		opt.EncodingPriority = "hevc"
	}
	if opt.OnlyAvc {
		logger.Warn("--only-avc/-avc is deprecated, use --encoding-priority to set encoding priority, setting avc as highest priority for this run")
		opt.EncodingPriority = "avc"
	}
	if opt.OnlyAv1 {
		logger.Warn("--only-av1/-av1 is deprecated, use --encoding-priority to set encoding priority, setting av1 as highest priority for this run")
		opt.EncodingPriority = "av1"
	}
	if opt.NoPaddingPageNum {
		logger.Warn("--no-padding-page-num is deprecated, use --file-pattern/-F or --multi-file-pattern/-M to customize output filename")
		if opt.FilePattern == "" && opt.MultiFilePattern == "" {
			opt.MultiFilePattern = strings.ReplaceAll(cli.MultiPageDefaultSavePath, "<pageNumberWithZero>", "<pageNumber>")
			logger.Warn("switched to -M " + opt.MultiFilePattern)
		}
	}
	if opt.BandwithAscending {
		logger.Warn("--bandwith-ascending is deprecated, use --video-ascending and --audio-ascending to specify whether video or audio should be sorted ascending, setting both to true for this run")
		opt.VideoAscending = true
		opt.AudioAscending = true
	}
}

func handleConflictingOptions(opt *cli.Option) {
	if opt.Interactive {
		opt.HideStreams = false
	}
	if opt.AudioOnly && opt.VideoOnly {
		opt.AudioOnly = false
		opt.VideoOnly = false
	}
	if opt.SkipSubtitle {
		opt.SubOnly = false
	}
}

func findBinaries(opt *cli.Option) error {
	if opt.FFmpegPath != "" {
		if _, err := os.Stat(opt.FFmpegPath); err != nil {
			return fmt.Errorf("ffmpeg path does not exist: %w", err)
		}
	}

	if opt.Mp4boxPath != "" {
		if _, err := os.Stat(opt.Mp4boxPath); err != nil {
			return fmt.Errorf("mp4box path does not exist: %w", err)
		}
	}

	if opt.Aria2cPath != "" {
		if _, err := os.Stat(opt.Aria2cPath); err != nil {
			return fmt.Errorf("aria2c path does not exist: %w", err)
		}
	}

	if !opt.SkipMux {
		if opt.UseMP4box {
			path := opt.Mp4boxPath
			if path == "" {
				var err error
				path, err = muxer.FindExecutable("mp4box")
				if err != nil {
					path, _ = muxer.FindExecutable("MP4box")
				}
			}
			if path == "" {
				return fmt.Errorf("cannot find mp4box executable")
			}
		} else {
			path := opt.FFmpegPath
			if path == "" {
				var err error
				path, err = muxer.FindExecutable("ffmpeg")
				if err != nil {
					return fmt.Errorf("cannot find ffmpeg executable: %w", err)
				}
			}
		}
	}

	if opt.UseAria2c {
		path := opt.Aria2cPath
		if path == "" {
			var err error
			path, err = muxer.FindExecutable("aria2c")
			if err != nil {
				return fmt.Errorf("cannot find aria2c executable: %w", err)
			}
		}
	}

	return nil
}

func changeWorkingDir(opt *cli.Option, logger *slog.Logger) error {
	if opt.WorkDir == "" {
		return nil
	}

	dir := os.ExpandEnv(opt.WorkDir)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolve work dir: %w", err)
	}

	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		if err := os.MkdirAll(absDir, 0o755); err != nil {
			return fmt.Errorf("create work dir: %w", err)
		}
	}

	if err := os.Chdir(absDir); err != nil {
		return fmt.Errorf("change work dir: %w", err)
	}

	logger.Debug("changed working directory", "dir", absDir)
	return nil
}

// ParseEncodingPriority splits by comma, uppercases, removes hyphens.
// Returns the priority map and the first encoding encountered.
func ParseEncodingPriority(s string) (map[string]byte, string) {
	encodingPriority := make(map[string]byte)
	firstEncoding := ""

	if s == "" {
		return encodingPriority, firstEncoding
	}

	s = strings.ReplaceAll(s, "，", ",")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ToUpper(s)

	parts := strings.Split(s, ",")
	var index byte
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if firstEncoding == "" {
			firstEncoding = part
		}
		if _, exists := encodingPriority[part]; exists {
			continue
		}
		encodingPriority[part] = index
		index++
	}

	return encodingPriority, firstEncoding
}

// ParseDfnPriority splits by comma, uppercases, trims.
func ParseDfnPriority(s string) map[string]int {
	dfnPriority := make(map[string]int)

	if s == "" {
		return dfnPriority
	}

	s = strings.ReplaceAll(s, "，", ",")
	parts := strings.Split(s, ",")
	index := 0
	for _, part := range parts {
		part = strings.ToUpper(strings.TrimSpace(part))
		if part == "" {
			continue
		}
		if _, exists := dfnPriority[part]; exists {
			continue
		}
		dfnPriority[part] = index
		index++
	}

	return dfnPriority
}

func parseDownloadDanmakuFormats(s string) []string {
	if s == "" {
		return []string{"ass", "xml"}
	}

	s = strings.ReplaceAll(s, "，", ",")
	s = strings.ToLower(s)
	parts := strings.Split(s, ",")
	var formats []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		formats = append(formats, part)
	}

	if len(formats) == 0 {
		return []string{"ass", "xml"}
	}

	return formats
}

func loadCredentials(opt *cli.Option, cfg *config.Config, logger *slog.Logger) {
	if cfg.Cookie == "" {
		cookiePath := filepath.Join(appDir, "BBDown.data")
		data, err := os.ReadFile(cookiePath)
		if err == nil {
			logger.Info("loaded local cookie", "path", cookiePath)
			cfg.Cookie = string(data)
		}
	}

	if cfg.Token == "" && opt.UseTvApi {
		tokenPath := filepath.Join(appDir, "BBDownTV.data")
		data, err := os.ReadFile(tokenPath)
		if err == nil {
			logger.Info("loaded local token", "path", tokenPath)
			cfg.Token = strings.Replace(string(data), "access_token=", "", 1)
		}
	}

	if cfg.Token == "" && opt.UseAppApi {
		tokenPath := filepath.Join(appDir, "BBDownApp.data")
		data, err := os.ReadFile(tokenPath)
		if err == nil {
			logger.Info("loaded local token", "path", tokenPath)
			cfg.Token = strings.Replace(string(data), "access_token=", "", 1)
		}
	}
}
