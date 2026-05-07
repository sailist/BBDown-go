# BBDown-Go

A Go rewrite of [BBDown](https://github.com/nilaoda/BBDown), a command-line Bilibili downloader.

> **Work in Progress** — Core download pipeline (URL parse → fetch → download → mux) is functional and tested. Some advanced features (aria2c backend, APP API gRPC) are partially implemented but not yet fully wired.

---

## Features

| Feature | C# BBDown | BBDown-Go |
|---|---|---|
| Single video / BV / AV download | ✅ | ✅ |
| Multi-page / Bangumi / Cheese support | ✅ | ✅ |
| TV / APP / International API | ✅ | ✅ |
| Multi-threaded download | ✅ | ✅ |
| aria2c backend | ✅ | 🚧 (module ready, not wired) |
| FFmpeg / MP4Box muxing | ✅ | ✅ |
| Subtitle & cover download | ✅ | ✅ |
| Danmaku (ASS) download | ✅ | ✅ |
| QR code login (WEB / TV) | ✅ | ✅ |
| HTTP API server (`serve`) | ❌ | ✅ |
| Config file support | ✅ | ✅ |
| Cross-platform (Win/Linux/macOS) | ✅ | ✅ |
| Structured JSON logging | ❌ | ✅ |
| Interface-driven, mockable tests | Partial | ✅ |

---

## Installation

### Prerequisites

- **Go 1.25+**
- (Optional) **FFmpeg** — for video/audio muxing
- (Optional) **aria2c** — for aria2c download backend
- (Optional) **MP4Box** — for MP4Box muxing mode

### Build from source

```bash
go install github.com/nilaonai/bbdown-go/cmd/bbdown@latest
```

Or clone and build manually:

```bash
git clone https://github.com/nilaonai/bbdown-go.git
cd bbdown-go
go build -o bbdown ./cmd/bbdown
```

### Download release

Check the [Releases](https://github.com/nilaonai/bbdown-go/releases) page for pre-built binaries.

---

## Usage

### Basic download

```bash
bbdown "https://www.bilibili.com/video/BV1xx411c7mD"
```

### Download specific pages

```bash
bbdown -p "1,3,5-7" "https://www.bilibili.com/video/BV1xx411c7mD"
```

### Select quality interactively

```bash
bbdown --interactive "https://www.bilibili.com/video/BV1xx411c7mD"
```

### Audio only

```bash
bbdown --audio-only "https://www.bilibili.com/video/BV1xx411c7mD"
```

### Subtitle only

```bash
bbdown --sub-only "https://www.bilibili.com/video/BV1xx411c7mD"
```

### Use TV API (often higher quality)

```bash
bbdown --use-tv-api "https://www.bilibili.com/video/BV1xx411c7mD"
```

### Download with aria2c

```bash
bbdown --use-aria2c --aria2c-args "--max-concurrent-downloads=16" "https://www.bilibili.com/video/BV1xx411c7mD"
```

### Only show video info

```bash
bbdown --only-show-info "https://www.bilibili.com/video/BV1xx411c7mD"
```

### Login via QR code

```bash
bbdown login      # WEB QR code
bbdown logintv    # TV QR code
```

### Start API server

```bash
bbdown serve -l "http://0.0.0.0:23333"
```

---

## Available Flags (Key)

| Flag | Short | Description |
|---|---|---|
| `--use-tv-api` | | Use TV API |
| `--use-app-api` | | Use APP API |
| `--use-intl-api` | | Use international API |
| `--encoding-priority` | `-e` | Encoding priority, e.g. `hevc,avc,av1` |
| `--dfn-priority` | `-q` | Quality priority |
| `--only-show-info` | | Only show video info |
| `--interactive` | | Interactive mode |
| `--use-aria2c` | | Use aria2c for downloading |
| `--aria2c-args` | | Extra arguments passed to aria2c |
| `--multi-thread` | | Enable multi-threaded download (default `true`) |
| `--select-page` | `-p` | Select pages, e.g. `1,2,3-5` |
| `--simply-mux` | | Simple mux mode |
| `--audio-only` | | Download audio only |
| `--video-only` | | Download video only |
| `--danmaku-only` | | Download danmaku only |
| `--cover-only` | | Download cover only |
| `--sub-only` | | Download subtitle only |
| `--skip-mux` | | Skip muxing step |
| `--skip-subtitle` | | Skip downloading subtitles |
| `--skip-cover` | | Skip downloading cover |
| `--download-danmaku` | | Download danmaku |
| `--skip-ai` | | Skip AI-generated subtitles (default `true`) |
| `--cookie` | `-c` | Cookie string or file path |
| `--access-token` | | Access token |
| `--work-dir` | | Working directory |
| `--ffmpeg-path` | | Path to ffmpeg executable |
| `--mp4box-path` | | Path to mp4box executable |
| `--aria2c-path` | | Path to aria2c executable |
| `--file-pattern` | `-F` | Output file name pattern |
| `--multi-file-pattern` | `-M` | Multi-page output file name pattern |
| `--host` | | API host (default `api.bilibili.com`) |
| `--config-file` | | Path to config file |
| `--debug` | | Enable debug logging |

Run `bbdown --help` for the full list.

---

## Development Setup

```bash
# 1. Clone the repository
git clone https://github.com/nilaonai/bbdown-go.git
cd bbdown-go

# 2. Run tests
make test

# 3. Build binary
make build

# 4. Run linter (requires golangci-lint)
make lint
```

### Requirements

- Go **1.25+**
- Make

---

## Architecture Overview

```
cmd/bbdown/           # Entry point
internal/
  cli/                # Cobra commands & flags
  app/                # Business logic (URL parse, info, sort, download orchestration)
  core/
    api/              # Bilibili API endpoints
    entity/           # Domain models
    fetcher/          # Data fetchers (normal, bangumi, cheese, series, etc.)
    parser/           # Response parsers
    util/             # Helpers
  download/           # Download engines (single, multi-thread, aria2c, progress)
  muxer/              # Muxers (FFmpeg, MP4Box)
  danmaku/            # Danmaku parsing & ASS generation
  login/              # QR code login (WEB / TV)
  server/             # HTTP API server (Gin)
  config/             # Config file & env parsing
pkg/
  httpclient/         # HTTP client abstraction (interface + standard implementation)
  logger/             # slog handler wrapper
  bvconv/             # BV / AV number converter
```

Key design decisions are recorded as Architecture Decision Records (ADRs) in [`docs/adr/`](docs/adr/).

---

## License

This project is licensed under the [MIT License](BBDown/LICENSE), consistent with the original C# BBDown project.
