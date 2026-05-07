# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-05-07

### Added

- **Core infrastructure**
  - Go module initialization with Go 1.25
  - Structured logging with `slog` and color support
  - Observable HTTP client interface with retry, cookie, and timeout support
  - BV/AV codec conversion (`pkg/bvconv`)
  - WBI sign and crypto utilities
  - Runtime config struct with quality mapping

- **Video info fetching**
  - Fetcher interface and factory pattern
  - Normal, Bangumi, Cheese, International, Collection, and Space fetchers
  - APP gRPC helper and protobuf definitions

- **Stream parsing**
  - Playurl API builder (WEB, TV, APP, International)
  - DASH stream parser with video/audio/background-audio/role-audio support
  - FLV stream parser with clip segmentation
  - Clip info and viewpoint parser
  - `ExtractTracks` orchestrator with VIP fallback and max-QN re-fetch

- **Download engine**
  - Downloader interface (single-threaded, multi-threaded range download)
  - aria2c integration
  - Progress bar reporter

- **Muxing**
  - Muxer interface (ffmpeg, mp4box)
  - External binary finder

- **CLI**
  - Cobra root command with 60+ flags mapped from C# `MyOption`
  - Login subcommands (WEB QR, TV QR)
  - Serve subcommand for HTTP API server mode

- **Workflow orchestration**
  - URL parser (BV, AV, EP, SS, MD, space, international)
  - Save path formatter with template variables
  - Track sorter with encoding/dfn priority
  - Setup and validation logic
  - `GetVideoInfo` workflow
  - `DownloadPage` workflow (info/subtitle, stream selection, download/mux)
  - `DownloadPages` batch workflow with page selection, delay, and archive deduplication
  - `main.go` entry point with signal handling and panic recovery

- **Utilities**
  - Subtitle fetcher and JSON-to-SRT converter
  - Danmaku XML parser and ASS converter with collision-free positioning
  - Console QR code renderer
  - Update checker

- **Server mode**
  - HTTP API server with Gin
  - Task queue with progress tracking
  - Webhook callbacks

- **Testing & CI**
  - Unit tests for all packages (80%+ coverage target)
  - Benchmarks for BV codec and parser
  - Integration test scaffold for complete download workflow
  - GitHub Actions CI (build, test, lint, coverage, Conventional Commits check)
  - goreleaser configuration for cross-platform releases

- **Documentation**
  - README with feature comparison, installation, usage examples
  - Architecture Decision Records (ADR) for logging, HTTP client, and testing strategy

- **Performance**
  - pprof endpoints (server mode)
  - HTTP connection pool tuning
  - Download concurrency limit with semaphore

[0.1.0]: https://github.com/nilaonai/bbdown-go/releases/tag/v0.1.0
