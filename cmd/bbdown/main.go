package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/sailist/BBDown-go/internal/app"
	"github.com/sailist/BBDown-go/internal/cli"
	"github.com/sailist/BBDown-go/internal/config"
	"github.com/sailist/BBDown-go/internal/core/fetcher"
	"github.com/sailist/BBDown-go/internal/core/parser"
	"github.com/sailist/BBDown-go/internal/download"
	"github.com/sailist/BBDown-go/internal/login"
	"github.com/sailist/BBDown-go/internal/muxer"
	"github.com/sailist/BBDown-go/pkg/httpclient"
)

func init() {
	cli.RunRoot = func(ctx context.Context, opt *cli.Option) error {
		return runApp(ctx, opt)
	}
	cli.RunLogin = func(ctx context.Context) error {
		return runLogin(ctx)
	}
	cli.RunLoginTV = func(ctx context.Context) error {
		return runLoginTV(ctx)
	}
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic recovered", "error", r)
			os.Exit(1)
		}
	}()

	// Logger
	logLevel := slog.LevelInfo
	if os.Getenv("DEBUG") == "1" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// Load config file from APP_DIR (matching C# Program.APP_DIR behavior)
	configPath := filepath.Join(app.AppDir, "BBDown.config")
	args, err := config.HandleConfig(os.Args[1:], configPath)
	if err != nil {
		logger.Error("failed to load config file", "error", err)
		os.Exit(1)
	}

	// CLI option and root command
	opt := cli.NewOption()
	cmd := cli.NewRootCommand(opt)
	cmd.SetArgs(args)

	// Signal handling
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Execute
	if err := cmd.ExecuteContext(ctx); err != nil {
		logger.Error("command failed", "error", err)
		os.Exit(1)
	}
}

func runApp(ctx context.Context, opt *cli.Option) error {
	// Check update asynchronously
	go func() {
		client := httpclient.NewStandardClient(slog.Default())
		if version, err := app.CheckUpdate(ctx, client); err == nil {
			slog.Info("new version available", "version", version)
		}
	}()

	return doWork(ctx, opt)
}

func runLogin(ctx context.Context) error {
	client := httpclient.NewStandardClient(slog.Default())
	wl := login.NewWebLogin(client, slog.Default(), app.AppDir)
	return wl.Login(ctx)
}

func runLoginTV(ctx context.Context) error {
	client := httpclient.NewStandardClient(slog.Default())
	tl := login.NewTVLogin(client, slog.Default(), app.AppDir)
	return tl.Login(ctx)
}

func doWork(ctx context.Context, opt *cli.Option) error {
	logger := slog.Default()
	client := httpclient.NewStandardClient(logger)

	// Setup work
	workCfg, err := app.SetupWork(opt, logger)
	if err != nil {
		return fmt.Errorf("setup work: %w", err)
	}

	// Propagate cookie to HTTP client so video CDN requests are authenticated
	client.SetCookie(workCfg.Config.Cookie)

	// Get video info
	aidOri, vInfo, _, err := app.GetVideoInfo(ctx, opt, workCfg.Input, app.Deps{
		FetcherFactory: func(id string, useIntl bool) (fetcher.Fetcher, error) {
			factory := fetcher.NewFactory(client, workCfg.Config, logger)
			return factory.Create(id, useIntl)
		},
		HTTPClient: client,
		Logger:     logger,
	})
	if err != nil {
		return fmt.Errorf("get video info: %w", err)
	}

	// Set AidOri for downstream use
	workCfg.AidOri = aidOri

	// Auto-detect aria2c if user didn't explicitly set --use-aria2c
	if !opt.UseAria2cChanged {
		if download.IsAria2cAvailable() {
			opt.UseAria2c = true
			logger.Info("aria2c found, using aria2c for download")
		} else {
			opt.UseAria2c = false
			logger.Info("aria2c not found, falling back to built-in downloader; install aria2c for faster multi-connection downloads")
		}
	}

	// Set up downloader: try multi-thread first, fallback to single
	var dl download.Downloader
	if opt.MultiThread {
		dl = download.NewMultiThreadDownloader(client, workCfg.Config, logger)
	} else {
		dl = download.NewSingleDownloader(client, workCfg.Config, logger)
	}

	// Set up muxer
	var mx muxer.Muxer
	if opt.UseMP4box {
		mx = muxer.NewMP4BoxMuxer(logger)
	} else {
		mx = muxer.NewFFmpegMuxer(logger)
	}

	// Download pages
	deps := app.DownloadDeps{
		HTTPClient:    client,
		Logger:        logger,
		Downloader:    dl,
		Muxer:         mx,
		Config:        workCfg.Config,
		ExtractTracks: parser.ExtractTracks,
	}

	return app.DownloadPages(ctx, opt, vInfo, workCfg, deps)
}
