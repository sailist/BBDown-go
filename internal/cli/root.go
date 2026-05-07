package cli

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
)

// Default save path constants.
const (
	SinglePageDefaultSavePath = "<videoTitle>"
	MultiPageDefaultSavePath  = "<videoTitle>/[P<pageNumberWithZero>]<pageTitle>"
)

// Option holds all CLI flags mapped from the C# MyOption class.
type Option struct {
	URL string

	UseTvApi                bool
	UseAppApi               bool
	UseIntlApi              bool
	UseMP4box               bool
	EncodingPriority        string
	DfnPriority             string
	OnlyShowInfo            bool
	HideStreams             bool
	Interactive             bool
	ShowAll                 bool
	UseAria2c               bool
	Aria2cArgs              string
	MultiThread             bool
	SelectPage              string
	SimplyMux               bool
	AudioOnly               bool
	VideoOnly               bool
	DanmakuOnly             bool
	CoverOnly               bool
	SubOnly                 bool
	Debug                   bool
	SkipMux                 bool
	SkipSubtitle            bool
	SkipCover               bool
	ForceHttp               bool
	DownloadDanmaku         bool
	DownloadDanmakuFormats  string
	SkipAi                  bool
	VideoAscending          bool
	AudioAscending          bool
	AllowPcdn               bool
	Language                string
	UserAgent               string
	Cookie                  string
	AccessToken             string
	WorkDir                 string
	FFmpegPath              string
	Mp4boxPath              string
	Aria2cPath              string
	UposHost                string
	ForceReplaceHost        bool
	SaveArchivesToFile      bool
	DelayPerPage            string
	FilePattern             string
	MultiFilePattern        string
	Host                    string
	EpHost                  string
	TvHost                  string
	Area                    string
	ConfigFile              string

	// Hidden / deprecated flags.
	Aria2cProxy       string
	OnlyHevc          bool
	OnlyAvc           bool
	OnlyAv1           bool
	AddDfnSubfix      bool
	NoPaddingPageNum  bool
	BandwithAscending bool
}

// NewOption returns an Option with default values.
func NewOption() *Option {
	return &Option{
		MultiThread:        true,
		SimplyMux:          false,
		ForceHttp:          true,
		DownloadDanmaku:    false,
		SkipAi:             true,
		VideoAscending:     false,
		AudioAscending:     false,
		AllowPcdn:          false,
		ForceReplaceHost:   true,
		SaveArchivesToFile: false,
		DelayPerPage:       "0",
		Host:               "api.bilibili.com",
		EpHost:             "api.bilibili.com",
		TvHost:             "api.snm0516.aisee.tv",
	}
}

// NewRootCommand creates the cobra root command with all flags registered.
func NewRootCommand(opt *Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bbdown [URL]",
		Short: "BBDown is a command-line Bilibili downloader",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opt.URL = args[0]
			}
			return RunRoot(cmd.Context(), opt)
		},
	}

	// API selection.
	cmd.Flags().BoolVar(&opt.UseTvApi, "use-tv-api", false, "use TV API")
	cmd.Flags().BoolVar(&opt.UseAppApi, "use-app-api", false, "use APP API")
	cmd.Flags().BoolVar(&opt.UseIntlApi, "use-intl-api", false, "use international API")

	// Tools.
	cmd.Flags().BoolVar(&opt.UseMP4box, "use-mp4box", false, "use MP4Box for muxing")
	cmd.Flags().StringVarP(&opt.EncodingPriority, "encoding-priority", "e", "", "encoding priority (e.g. hevc,avc,av1)")
	cmd.Flags().StringVarP(&opt.DfnPriority, "dfn-priority", "q", "", "quality priority")

	// Info / interaction.
	cmd.Flags().BoolVar(&opt.OnlyShowInfo, "only-show-info", false, "only show video info")
	cmd.Flags().BoolVar(&opt.HideStreams, "hide-streams", false, "hide available streams")
	cmd.Flags().BoolVar(&opt.Interactive, "interactive", false, "interactive mode")
	cmd.Flags().BoolVar(&opt.ShowAll, "show-all", false, "show all pages")

	// Download engine.
	cmd.Flags().BoolVar(&opt.UseAria2c, "use-aria2c", false, "use aria2c for downloading")
	cmd.Flags().StringVar(&opt.Aria2cArgs, "aria2c-args", "", "extra arguments passed to aria2c")
	cmd.Flags().BoolVar(&opt.MultiThread, "multi-thread", true, "enable multi-threaded download")

	// Page selection.
	cmd.Flags().StringVarP(&opt.SelectPage, "select-page", "p", "", "select pages to download (e.g. 1,2,3-5)")

	// Output mode.
	cmd.Flags().BoolVar(&opt.SimplyMux, "simply-mux", false, "simple mux mode")
	cmd.Flags().BoolVar(&opt.AudioOnly, "audio-only", false, "download audio only")
	cmd.Flags().BoolVar(&opt.VideoOnly, "video-only", false, "download video only")
	cmd.Flags().BoolVar(&opt.DanmakuOnly, "danmaku-only", false, "download danmaku only")
	cmd.Flags().BoolVar(&opt.CoverOnly, "cover-only", false, "download cover only")
	cmd.Flags().BoolVar(&opt.SubOnly, "sub-only", false, "download subtitle only")

	// Debug / skip.
	cmd.Flags().BoolVar(&opt.Debug, "debug", false, "enable debug logging")
	cmd.Flags().BoolVar(&opt.SkipMux, "skip-mux", false, "skip muxing step")
	cmd.Flags().BoolVar(&opt.SkipSubtitle, "skip-subtitle", false, "skip downloading subtitles")
	cmd.Flags().BoolVar(&opt.SkipCover, "skip-cover", false, "skip downloading cover")

	// HTTP / network.
	cmd.Flags().BoolVar(&opt.ForceHttp, "force-http", true, "force HTTP instead of HTTPS")
	cmd.Flags().BoolVar(&opt.DownloadDanmaku, "download-danmaku", false, "download danmaku")
	cmd.Flags().StringVar(&opt.DownloadDanmakuFormats, "download-danmaku-formats", "", "danmaku output formats")
	cmd.Flags().BoolVar(&opt.SkipAi, "skip-ai", true, "skip AI-generated subtitles")
	cmd.Flags().BoolVar(&opt.VideoAscending, "video-ascending", false, "sort video tracks ascending")
	cmd.Flags().BoolVar(&opt.AudioAscending, "audio-ascending", false, "sort audio tracks ascending")
	cmd.Flags().BoolVar(&opt.AllowPcdn, "allow-pcdn", false, "allow PCDN URLs")

	// Misc.
	cmd.Flags().StringVar(&opt.Language, "language", "", "preferred subtitle language")
	cmd.Flags().StringVar(&opt.UserAgent, "user-agent", "", "custom User-Agent")
	cmd.Flags().StringVarP(&opt.Cookie, "cookie", "c", "", "cookie string or file path")
	cmd.Flags().StringVar(&opt.AccessToken, "access-token", "", "access token")
	cmd.Flags().StringVar(&opt.WorkDir, "work-dir", "", "working directory")
	cmd.Flags().StringVar(&opt.FFmpegPath, "ffmpeg-path", "", "path to ffmpeg executable")
	cmd.Flags().StringVar(&opt.Mp4boxPath, "mp4box-path", "", "path to mp4box executable")
	cmd.Flags().StringVar(&opt.Aria2cPath, "aria2c-path", "", "path to aria2c executable")
	cmd.Flags().StringVar(&opt.UposHost, "upos-host", "", "custom UPOS host")
	cmd.Flags().BoolVar(&opt.ForceReplaceHost, "force-replace-host", true, "force replace host in URLs")
	cmd.Flags().BoolVar(&opt.SaveArchivesToFile, "save-archives-to-file", false, "save archive list to file")
	cmd.Flags().StringVar(&opt.DelayPerPage, "delay-per-page", "0", "delay between pages")
	cmd.Flags().StringVarP(&opt.FilePattern, "file-pattern", "F", "", "output file name pattern")
	cmd.Flags().StringVarP(&opt.MultiFilePattern, "multi-file-pattern", "M", "", "multi-page output file name pattern")

	// Host overrides.
	cmd.Flags().StringVar(&opt.Host, "host", "api.bilibili.com", "API host")
	cmd.Flags().StringVar(&opt.EpHost, "ep-host", "api.bilibili.com", "EP API host")
	cmd.Flags().StringVar(&opt.TvHost, "tv-host", "api.snm0516.aisee.tv", "TV API host")
	cmd.Flags().StringVar(&opt.Area, "area", "", "area code for international API")
	cmd.Flags().StringVar(&opt.ConfigFile, "config-file", "", "path to config file")

	// Hidden / deprecated flags.
	cmd.Flags().StringVar(&opt.Aria2cProxy, "aria2c-proxy", "", "aria2c proxy (deprecated)")
	_ = cmd.Flags().MarkHidden("aria2c-proxy")

	cmd.Flags().BoolVar(&opt.OnlyHevc, "only-hevc", false, "only download HEVC streams (deprecated)")
	_ = cmd.Flags().MarkHidden("only-hevc")

	cmd.Flags().BoolVar(&opt.OnlyAvc, "only-avc", false, "only download AVC streams (deprecated)")
	_ = cmd.Flags().MarkHidden("only-avc")

	cmd.Flags().BoolVar(&opt.OnlyAv1, "only-av1", false, "only download AV1 streams (deprecated)")
	_ = cmd.Flags().MarkHidden("only-av1")

	cmd.Flags().BoolVar(&opt.AddDfnSubfix, "add-dfn-subfix", false, "add quality suffix to filename (deprecated)")
	_ = cmd.Flags().MarkHidden("add-dfn-subfix")

	cmd.Flags().BoolVar(&opt.NoPaddingPageNum, "no-padding-page-num", false, "do not pad page number (deprecated)")
	_ = cmd.Flags().MarkHidden("no-padding-page-num")

	cmd.Flags().BoolVar(&opt.BandwithAscending, "bandwith-ascending", false, "sort by bandwidth ascending (deprecated)")
	_ = cmd.Flags().MarkHidden("bandwith-ascending")

	return cmd
}

// RunRoot is the entry point for the root command.
var RunRoot = func(ctx context.Context, opt *Option) error {
	slog.DebugContext(ctx, "root command executed", "url", opt.URL)
	return fmt.Errorf("not implemented: %w", fmt.Errorf("root command handler not yet wired"))
}
