# BBDown-Go

[BBDown](https://github.com/nilaoda/BBDown) 的 Go 语言重写版本，一个命令行 Bilibili 视频下载工具。

> **注意**：核心下载链路（URL 解析 → 获取信息 → 下载 → 混流）已完成并经过测试。部分高级功能（aria2c 后端、APP API gRPC）已部分实现，尚未完全接入。

---

## 功能特性

| 功能 | C# 版 BBDown | BBDown-Go |
|---|---|---|
| 单视频 / BV / AV 下载 | ✅ | ✅ |
| 多 P / 番剧 / 课程支持 | ✅ | ✅ |
| TV / APP / 国际版 API | ✅ | ✅ |
| 多线程下载 | ✅ | ✅ |
| aria2c 后端 | ✅ | 🚧（模块就绪，未完全接入） |
| FFmpeg / MP4Box 混流 | ✅ | ✅ |
| 字幕与封面下载 | ✅ | ✅ |
| 弹幕（ASS）下载 | ✅ | ✅ |
| 扫码登录（WEB / TV） | ✅ | ✅ |
| HTTP API 服务器（`serve`） | ❌ | ✅ |
| 配置文件支持 | ✅ | ✅ |
| 跨平台（Win / Linux / macOS） | ✅ | ✅ |
| 结构化 JSON 日志 | ❌ | ✅ |
| 接口驱动、可 Mock 测试 | 部分 | ✅ |

---

## 安装方法

### 前置依赖

- **Go 1.25+**（如使用 `go install` 安装）
-（可选）**FFmpeg** — 用于视频/音频混流
-（可选）**aria2c** — 用于 aria2c 下载后端
-（可选）**MP4Box** — 用于 MP4Box 混流模式

### 方法一：go install（推荐）

```bash
go install github.com/sailist/BBDown-go/cmd/bbdown@latest
```

安装完成后，确保 `$GOPATH/bin` 或 `$GOBIN` 目录已添加到系统的 `PATH` 环境变量中。

### 方法二：下载预编译二进制文件

前往 [Releases](https://github.com/sailist/BBDown-go/releases) 页面，下载对应平台的压缩包，解压后即可使用。

### 方法三：从源码编译

```bash
git clone https://github.com/sailist/BBDown-go.git
cd BBDown-go
go build -o bbdown ./cmd/bbdown
```

---

## 使用说明

### 基础下载

```bash
bbdown "https://www.bilibili.com/video/BV1xx411c7mD"
```

### 下载指定分 P

```bash
bbdown -p "1,3,5-7" "https://www.bilibili.com/video/BV1xx411c7mD"
```

### 交互式选择画质

```bash
bbdown --interactive "https://www.bilibili.com/video/BV1xx411c7mD"
```

### 仅下载音频

```bash
bbdown --audio-only "https://www.bilibili.com/video/BV1xx411c7mD"
```

### 仅下载字幕

```bash
bbdown --sub-only "https://www.bilibili.com/video/BV1xx411c7mD"
```

### 使用 TV API（通常画质更高）

```bash
bbdown --use-tv-api "https://www.bilibili.com/video/BV1xx411c7mD"
```

### 使用 aria2c 下载

```bash
bbdown --use-aria2c --aria2c-args "--max-concurrent-downloads=16" "https://www.bilibili.com/video/BV1xx411c7mD"
```

### 仅查看视频信息

```bash
bbdown --only-show-info "https://www.bilibili.com/video/BV1xx411c7mD"
```

### 扫码登录

```bash
bbdown login      # WEB 端扫码
bbdown logintv    # TV 端扫码
```

### 启动 API 服务器

```bash
bbdown serve -l "http://0.0.0.0:23333"
```

---

## 常用参数

| 参数 | 简写 | 说明 |
|---|---|---|
| `--use-tv-api` | | 使用 TV API |
| `--use-app-api` | | 使用 APP API |
| `--use-intl-api` | | 使用国际版 API |
| `--encoding-priority` | `-e` | 编码优先级，如 `hevc,avc,av1` |
| `--dfn-priority` | `-q` | 画质优先级 |
| `--only-show-info` | | 仅显示视频信息 |
| `--interactive` | | 交互模式 |
| `--use-aria2c` | | 使用 aria2c 下载 |
| `--aria2c-args` | | 传递给 aria2c 的额外参数 |
| `--multi-thread` | | 启用多线程下载（默认 `true`） |
| `--select-page` | `-p` | 选择分 P，如 `1,2,3-5` |
| `--simply-mux` | | 简单混流模式 |
| `--audio-only` | | 仅下载音频 |
| `--video-only` | | 仅下载视频 |
| `--danmaku-only` | | 仅下载弹幕 |
| `--cover-only` | | 仅下载封面 |
| `--sub-only` | | 仅下载字幕 |
| `--skip-mux` | | 跳过混流步骤 |
| `--skip-subtitle` | | 跳过字幕下载 |
| `--skip-cover` | | 跳过封面下载 |
| `--download-danmaku` | | 下载弹幕 |
| `--skip-ai` | | 跳过 AI 生成字幕（默认 `true`） |
| `--cookie` | `-c` | Cookie 字符串或文件路径 |
| `--access-token` | | Access Token |
| `--work-dir` | | 工作目录 |
| `--ffmpeg-path` | | ffmpeg 可执行文件路径 |
| `--mp4box-path` | | mp4box 可执行文件路径 |
| `--aria2c-path` | | aria2c 可执行文件路径 |
| `--file-pattern` | `-F` | 输出文件名模板 |
| `--multi-file-pattern` | `-M` | 多 P 输出文件名模板 |
| `--host` | | API 主机（默认 `api.bilibili.com`） |
| `--config-file` | | 配置文件路径 |
| `--debug` | | 启用调试日志 |

运行 `bbdown --help` 查看完整参数列表。

---

## 开发环境

```bash
# 1. 克隆仓库
git clone https://github.com/sailist/BBDown-go.git
cd BBDown-go

# 2. 运行测试
make test

# 3. 编译二进制文件
make build

# 4. 运行代码检查
make vet
```

### 环境要求

- Go **1.25+**
- Make

---

## 架构概览

```
cmd/bbdown/           # 入口程序
internal/
  cli/                # Cobra 命令与参数
  app/                # 业务逻辑（URL 解析、信息获取、排序、下载编排）
  core/
    api/              # Bilibili API 接口
    entity/           # 领域模型
    fetcher/          # 数据获取器（普通视频、番剧、课程、合集等）
    parser/           # 响应解析器
    util/             # 工具函数
  download/           # 下载引擎（单线程、多线程、aria2c、进度条）
  muxer/              # 混流器（FFmpeg、MP4Box）
  danmaku/            # 弹幕解析与 ASS 生成
  login/              # 扫码登录（WEB / TV）
  server/             # HTTP API 服务器（Gin）
  config/             # 配置文件与环境变量解析
pkg/
  httpclient/         # HTTP 客户端抽象（接口 + 标准实现）
  logger/             # slog 处理器封装
  bvconv/             # BV / AV 号转换器
```

关键设计决策记录见 [`docs/adr/`](docs/adr/)。

---

## 免责声明

1. **本工具仅供个人学习、研究及技术交流使用**，严禁用于任何商业用途。
2. 使用本工具下载的内容版权归原著作权人所有，用户应遵守相关法律法规及 Bilibili 平台的服务条款。
3. **请勿将本工具用于侵犯他人知识产权的行为**，包括但不限于未经授权的复制、传播、修改等。
4. 因使用本工具而产生的任何法律纠纷或责任，均由用户自行承担，与本项目开发者及贡献者无关。
5. 本项目开发者不对用户使用本工具的行为承担任何担保责任，也不对用户因使用本工具而造成的任何直接或间接损失负责。
6. 如您认为本工具侵犯了您的合法权益，请及时联系我们，我们将立即采取相应措施。

---

## 许可证

本项目采用 [MIT 许可证](BBDown/LICENSE)，与原 C# 版 BBDown 项目保持一致。
