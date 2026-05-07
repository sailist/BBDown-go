package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nilaonai/bbdown-go/internal/app"
	"github.com/nilaonai/bbdown-go/internal/cli"
	"github.com/nilaonai/bbdown-go/internal/core/fetcher"
	"github.com/nilaonai/bbdown-go/internal/core/parser"
	"github.com/nilaonai/bbdown-go/internal/download"
	"github.com/nilaonai/bbdown-go/internal/muxer"
	"github.com/nilaonai/bbdown-go/pkg/httpclient"
)

func init() {
	cli.RunRoot = func(ctx context.Context, opt *cli.Option) error {
		return runApp(ctx, opt)
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

	// CLI option and root command
	opt := cli.NewOption()
	cmd := cli.NewRootCommand(opt)

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

func doWork(ctx context.Context, opt *cli.Option) error {
	logger := slog.Default()
	client := httpclient.NewStandardClient(logger)

	// Setup work
	workCfg, err := app.SetupWork(opt, logger)
	if err != nil {
		return fmt.Errorf("setup work: %w", err)
	}

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
		ExtractTracks: parser.ExtractTracks,
	}

	return app.DownloadPages(ctx, opt, vInfo, workCfg, deps)
}
