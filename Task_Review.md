# BBDown-Go 重写计划：Review & 优化版原子化提交路线图

> 本文件基于对原 BBDown C# 代码库的深度分析，对 `Task.md` 进行全面 Review，并输出一份更精确、更可测试、更符合 Go 最佳实践的原子化提交计划。

---

## Conventional Commits 规范（强制）

本项目**所有提交必须严格遵守 [Conventional Commits v1.0.0](https://www.conventionalcommits.org/zh-hans/v1.0.0/) 规范**。不允许出现任何不符合规范的提交信息。

### 提交信息格式

```
<type>[(optional scope)][!]: <description>

[optional body]

[optional footer(s)]
```

### 允许的 type

| type | 用途 | 示例 |
|------|------|------|
| `feat` | 新功能 | `feat(parser): add DASH stream parser` |
| `fix` | 修复 bug | `fix(download): correct range request boundary` |
| `docs` | 文档变更 | `docs: update README with install instructions` |
| `style` | 代码格式（不影响功能） | `style: format with gofmt` |
| `refactor` | 重构（非 feat/fix） | `refactor: unify error wrapping` |
| `perf` | 性能优化 | `perf(httpclient): reuse TCP connections` |
| `test` | 测试相关 | `test(fetcher): add NormalInfoFetcher mock tests` |
| `build` | 构建系统/依赖 | `build: bump cobra to v1.8.0` |
| `ci` | CI/CD 配置 | `ci: add commitlint check` |
| `chore` | 杂项/工具 | `chore: initialize go module` |
| `revert` | 回滚提交 | `revert: feat(parser): add DASH stream parser` |

### scope 规范

scope 必须与代码目录层级对应：

- `pkg/logger` → `pkg/logger` 或 `logger`
- `pkg/httpclient` → `pkg/httpclient` 或 `httpclient`
- `pkg/bvconv` → `pkg/bvconv` 或 `bvconv`
- `internal/core/entity` → `core/entity` 或 `entity`
- `internal/core/fetcher` → `core/fetcher` 或 `fetcher`
- `internal/core/parser` → `core/parser` 或 `parser`
- `internal/core/api` → `core/api` 或 `api`
- `internal/core/util` → `core/util` 或 `util`
- `internal/download` → `download`
- `internal/muxer` → `muxer`
- `internal/cli` → `cli`
- `internal/config` → `config`
- `internal/app` → `app`
- `internal/login` → `login`
- `internal/danmaku` → `danmaku`
- `internal/server` → `server`
- `cmd/bbdown` → `cmd/bbdown` 或 `bbdown`
- 全局/多模块 → 省略 scope

### description 规范

- 使用英文小写开头，**禁止大写开头**
- 使用祈使句/现在时（`add` 而非 `added`/`adds`）
- 末尾**不加句号**
- 长度不超过 72 字符

### body 规范（可选）

- 解释 **why** 而非 **what**（what 从 diff 可见）
- 每行不超过 100 字符
- 与 description 之间空一行

### footer 规范（可选）

- `Closes #123` / `Fixes #456`
- `BREAKING CHANGE: xxx`

### 自动化检查

- **提交 1** 配置 `lefthook` + `commitlint`（或等效 Git Hook）阻止不规范提交
- **提交 53 (CI)** 配置 `wagoid/commitlint-github-action` 在 PR 中检查
- **禁止**使用 `--no-verify` 跳过检查

---

## 一、原 Task.md 的问题诊断

### 1.1 日志系统：自研 Logger 是反模式
**原方案**：提交 2 建议"自研彩色 logger"。
**问题**：
- 放弃了 Go 1.21+ 标准库 `log/slog` 的结构化能力
- 无法平滑支持服务器模式下的 JSON 日志输出
- 没有日志级别概念（原 C# 只有 `DEBUG_LOG` bool 开关）
- 与 Go 生态的可观测性工具链（OpenTelemetry、Prometheus 等）割裂

**改进**：使用 `log/slog` + 自定义 `slog.Handler`，CLI 模式下彩色文本输出，服务器模式下 JSON 输出。从 Day 1 就要有 `Debug/Info/Warn/Error` 四级分级。

### 1.2 可测试性：测试提交过于集中
**原方案**：Phase 6（提交 38-39）才集中补测试。
**问题**：
- 前期代码缺乏接口设计，后期补测试需要大量重构
- 没有 TDD 导向，模块耦合后难以 mock

**改进**：**每个提交都必须包含 `_test.go`**。Fetcher、Downloader、Muxer、Parser 全部定义为接口，从第一行生产代码开始就考虑测试。

### 1.3 全局状态：C# 风格的静态类直接平移
**原方案**：`internal/config/config.go` 复制 C# `Config.cs` 的全局静态变量风格。
**问题**：
- `Config.COOKIE`、`Config.TOKEN` 等全局变量在并发场景下（服务器模式）是灾难
- 单元测试之间会互相污染状态
- 不符合 Go "显式优于隐式"的哲学

**改进**：配置封装为结构体，通过构造函数注入（`NewXXX(cfg *Config)`）。仅保留真正全局的常量（如 quality map）。

### 1.4 Context 传播：事后补票
**原方案**：提交 40 才统一审查 `context.Context`。
**问题**：Context 是 Go 并发的根，事后重构成本极高。

**改进**：从第一个 IO 操作开始就传递 `ctx context.Context`。

### 1.5 错误处理：缺乏统一策略
**原方案**：未提及错误处理策略。
**问题**：C# 代码大量使用裸 `throw new Exception(...)`，Go 需要更精细的错误处理。

**改进**：定义领域错误类型（`var ErrLoginRequired = errors.New(...)`），使用 `fmt.Errorf("...: %w", err)` 包装，配合 `errors.Is/As` 做错误判断。

### 1.6 HTTP 客户端：裸用标准库
**原方案**：直接用 `net/http` 不加封装。
**问题**：BBDown 的 HTTP 逻辑复杂（Cookie 注入、UA 随机、Referer、gRPC header、gzip、重定向、Range、重试），裸用会导致重复代码。

**改进**：封装 `pkg/httpclient.Client` 接口，内置重试、日志、header 管理。

### 1.7 并发控制：未提及 errgroup
**原方案**：多线程下载用 goroutine + channel。
**问题**：BBDown 的多线程下载涉及"多个片段并发下载，任一失败整体失败"，裸 goroutine 难以优雅处理。

**改进**：使用 `golang.org/x/sync/errgroup`。

### 1.8 提交粒度：部分提交过大
**原方案问题提交**：
- 提交 12：DASH + FLV parser 合在一起，代码量过大
- 提交 29：DownloadPage workflow 包含封面、字幕、弹幕、下载、混流、清理，是一个巨大的事务脚本
- 提交 34：Server 模式一次性提交

**改进**：将这些大提交拆分为更小的、可独立测试的单元。

---

## 二、关键技术决策（修正版）

| 决策点 | 修正后选择 | 理由 |
|--------|-----------|------|
| **日志** | `log/slog` + 自定义彩色 Handler | 标准库结构化日志，支持文本/JSON切换，可观测性最佳实践 |
| **HTTP 客户端** | `net/http` + 自定义 Client 封装 | 轻量，但内置重试、日志、header 模板 |
| **并发控制** | `golang.org/x/sync/errgroup` | 并发子任务错误处理最佳实践 |
| **CLI 框架** | `spf13/cobra` | 保持，参数多但生态成熟 |
| **JSON 解析** | `encoding/json` | 标准库，无需第三方 |
| **gRPC/Proto** | `protoc-gen-go` | 从现有 `.proto` 直接生成，保真度高 |
| **XML 解析** | `encoding/xml` | 标准库，弹幕 XML 结构简单 |
| **进度条** | `schollz/progressbar/v3` | 保持，成熟 |
| **QR 码** | `skip2/go-qrcode` | 保持 |
| **配置文件** | 简单文本解析 | 原 `BBDown.config` 格式简单，无需 Viper |
| **测试** | 每个提交必带 `_test.go` | 接口 + httptest + mock |
| **Context** | 从 Day 1 全链路传递 | Go 并发与取消的标准做法 |

---

## 三、项目架构（优化版）

```
bbdown-go/
├── cmd/bbdown/              # 主入口
├── internal/
│   ├── app/                 # 应用层（编排、工作流）
│   │   ├── workflow.go      # DoWork, GetVideoInfo, DownloadPages
│   │   ├── url_parser.go    # ParseInput (GetAvId)
│   │   ├── formatter.go     # FormatSavePath
│   │   ├── sorter.go        # Track sorting
│   │   └── setup.go         # SetupWork
│   ├── cli/                 # 命令行解析 (cobra)
│   ├── config/              # 配置管理
│   │   ├── config.go        # Option struct, runtime config
│   │   └── parser.go        # BBDown.config file parser
│   ├── core/                # 核心业务（无外部依赖）
│   │   ├── entity/          # 数据模型（纯 struct）
│   │   ├── fetcher/         # Fetcher interface + implementations
│   │   ├── parser/          # Stream parser (DASH/FLV)
│   │   ├── api/             # API builders (WEB/TV/INTL/APP gRPC)
│   │   └── util/            # WBI sign, subtitle, BV conv
│   ├── download/            # Download engine
│   ├── muxer/               # FFmpeg/MP4Box muxer
│   ├── login/               # QR code login
│   ├── danmaku/             # Danmaku XML -> ASS
│   └── server/              # HTTP API server (gin or stdlib)
├── pkg/
│   ├── logger/              # slog Handler (color text + JSON)
│   └── httpclient/          # HTTP client with retry & observability
└── go.mod
```

### 设计原则
1. **依赖注入**：所有服务通过 `NewService(cfg, logger, client)` 创建，不读全局变量
2. **接口隔离**：Fetcher / Downloader / Muxer / HTTPClient 全部是 interface
3. **Context 贯穿**：所有阻塞/IO 函数签名第一参数是 `ctx context.Context`
4. **结构化日志**：使用 `slog.Logger`，通过 `With("key", value)` 注入 trace 字段
5. **错误链**：所有错误用 `%w` 包装，领域错误用 `errors.New` 定义常量

---

## 四、原子化提交路线图（修订版）

### Phase 0: 骨架与可观测基础设施（提交 1-8）

本阶段的核心目标是建立**可测试、可观测、无全局状态**的基础设施。每个提交附带测试。

#### 提交 1: `chore: initialize go module and project structure`
- `go mod init github.com/yourname/bbdown-go`
- 创建完整目录结构（空目录 + `.gitkeep`）
- `.gitignore`（Go 标准 + 二进制 + 临时下载目录 `*.tmp` `*.vclip` `*.aclip`）
- `Makefile`（build, test, lint, fmt, vet）
- `go.mod` 预声明依赖（cobra, progressbar, errgroup, go-qrcode）
- **配置 Conventional Commits 检查**：
  - 添加 `lefthook.yml`（Go 原生 git hooks 管理器）
  - 配置 `commitlint` 规则（`@commitlint/config-conventional` 的轻量 Go 替代方案，或直接调用 `commitlint` CLI）
  - 推荐方案：安装 `lefthook`，在 `commit-msg` hook 中运行自定义脚本检查提交格式
  - 检查规则脚本示例：
    ```bash
    #!/bin/bash
    # scripts/commit-msg-check.sh
    commit_msg_file=$1
    commit_msg=$(head -n1 "$commit_msg_file")
    pattern="^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\(.+\))?!?: .+"
    if ! echo "$commit_msg" | grep -qE "$pattern"; then
        echo "ERROR: Commit message does not follow Conventional Commits."
        echo "Expected format: <type>[(scope)]: <description>"
        exit 1
    fi
    ```
- 添加 `COMMIT_CONVENTION.md` 到项目根目录
- **目标**: `go build ./cmd/bbdown` 通过空 main；`git commit` 不规范时自动拦截

#### 提交 2: `feat(pkg/logger): implement structured slog handler with color support`
- 实现 `pkg/logger/handler.go`：自定义 `slog.Handler`
  - 支持 `DEBUG/INFO/WARN/ERROR` 四级
  - CLI 模式：彩色文本输出（时间/级别/消息/属性），兼容 Windows（`colorable`）
  - 服务器模式（通过 `WithFormat(JSON)`）：JSON 输出
  - `Debug()` 级别是否启用由外部配置决定
- 实现 `pkg/logger/logger.go`：工厂函数 `New(level, format) *slog.Logger`
- 添加 `pkg/logger/handler_test.go`
  - 测试各级别过滤
  - 测试属性注入
  - 测试 JSON 输出格式
- **对应 C#**: `BBDown.Core/Logger.cs`
- **最佳实践**: 使用 `log/slog`，这是可观测的基石。不要用 `fmt.Println` 或自研 logger。

#### 提交 3: `feat(pkg/httpclient): add observable HTTP client interface and implementation`
- 定义 `pkg/httpclient/client.go`：
  ```go
  type Client interface {
      Get(ctx context.Context, url string, opts ...RequestOption) (*http.Response, error)
      Post(ctx context.Context, url string, body []byte, opts ...RequestOption) (*http.Response, error)
      Head(ctx context.Context, url string, opts ...RequestOption) (*http.Response, error)
      GetRedirectLocation(ctx context.Context, url string) (string, error)
      GetContentLength(ctx context.Context, url string) (int64, error)
  }
  ```
- 实现 `pkg/httpclient/standard.go`：
  - 底层用 `net/http` + 自定义 `Transport`
  - 连接池配置（`MaxIdleConns: 100`）
  - 随机 User-Agent 生成器（复刻 C# 逻辑）
  - 自动重试（3 次，指数退避）
  - 自动 gzip 解压
  - `RequestOption` 函数选项模式注入 Header（Cookie, Referer 等）
  - 所有请求自动记录 `slog.Debug`（URL, Method, Status, Duration）
- 添加 `pkg/httpclient/client_test.go`
  - 使用 `httptest` mock server 测试重试逻辑
  - 测试 UA 随机性
  - 测试 gzip 解压
- **对应 C#**: `BBDown.Core/Util/HTTPUtil.cs`
- **最佳实践**: Client 是 interface，便于后续 mock 测试整个 Fetcher 层

#### 提交 4: `feat(pkg/bvconv): implement BV/AV codec with tests` ✅
- 实现 `pkg/bvconv/bvconv.go`
  - `Encode(aid int64) string`
  - `Decode(bv string) (int64, error)`
  - 复制 C# 算法（base58 + 码表）
- 添加 `pkg/bvconv/bvconv_test.go`
  - 与 C# 版本交叉验证 100 组随机数据
  - 边界值测试（最小/最大 aid）
  - 错误输入测试
- **对应 C#**: `BBDown.Core/Util/BilibiliBvConverter.cs`

#### 提交 5: `feat(internal/core/entity): define core data models` ✅
- 实现 `internal/core/entity/` 下所有 struct：
  - `Page`（含 `BVid()` 方法调用 `pkg/bvconv`）
  - `Video`, `Audio`, `Subtitle`, `ViewPoint`, `Clip`
  - `AudioMaterial`, `AudioMaterialInfo`
  - `VInfo`, `ParsedResult`
- 所有 struct 字段加 `json` tag（为后续解析做准备）
- `Equals` 逻辑：Go 中通过 `reflect.DeepEqual` 或手写 `Equal` 方法
- 添加 `internal/core/entity/entity_test.go`
  - 测试 `Page.Equal()`
  - 测试 `Video.Equal()` / `Audio.Equal()`
- **对应 C#**: `BBDown.Core/Entity/Entity.cs`, `VInfo.cs`, `ParsedResult.cs`

#### 提交 6: `feat(internal/config): add runtime config struct and quality mapping`
- 实现 `internal/config/config.go`：
  - `type Config struct { Cookie, Token, DebugLog, Host, EpHost, TvHost, Area string; ... }`
  - `Qualities map[string]string`（qn → 清晰度名称，只读）
  - 提供 `NewConfig() *Config` 返回默认值
  - **注意**：不再有全局变量 `var COOKIE string`，所有模块通过参数接收 `*Config`
- 添加 `internal/config/config_test.go`
- **对应 C#**: `BBDown.Core/Config.cs`

#### 提交 7: `feat(internal/core/util): add WBI sign and crypto utilities with tests`
- 实现 `internal/core/util/crypto.go`：
  - `WbiSign(api string, wbi string) string`
  - `GetMixinKey(orig string) string`
  - `GetSign(parms string, isBiliPlus bool) string`
  - `GetTimestamp(seconds bool) string`
  - `FormatFileSize(size float64) string`
  - `FormatTime(seconds int, absolute bool) string`
  - `GetValidFileName(input string) string`
- 每个函数都带单元测试，验证与 C# 输出一致性
- **对应 C#**: `BBDownUtil.cs` 工具函数部分

#### 提交 8: `chore: add CI workflow and pre-commit hooks`
- `.github/workflows/ci.yml`：
  - `go test ./...`
  - `go vet ./...`
  - `golangci-lint run`
  - 跨平台编译检查（Linux/macOS/Windows）
- **目标**: 此阶段结束后 `go test ./...` 全绿

---

### Phase 1: 核心解析引擎（提交 9-20）

本阶段的核心目标是实现**视频信息获取**和**流解析**，全部基于接口，100% 可 mock 测试。

#### 提交 9: `feat(internal/core/fetcher): define Fetcher interface and factory`
- 定义 `internal/core/fetcher/fetcher.go`：
  ```go
  type Fetcher interface {
      Fetch(ctx context.Context, id string) (*entity.VInfo, error)
  }
  type Factory func(id string, useIntl bool) (Fetcher, error)
  ```
- 实现工厂函数（目前只返回 error：not implemented）
- 添加 `internal/core/fetcher/fetcher_test.go`
  - 测试工厂对非法 id 返回错误
- **对应 C#**: `BBDown.Core/IFetcher.cs`, `FetcherFactory.cs`

#### 提交 10: `feat(internal/core/fetcher): implement NormalInfoFetcher` ✅
- 实现 `internal/core/fetcher/normal.go`
  - 调用 `x/web-interface/view?aid={id}`
  - 解析分 P 信息
  - 处理互动视频降级提示（`is_stein_gate`）
  - 处理 redirect_url 番剧跳转
- 构造函数签名：`func NewNormalFetcher(client httpclient.Client, cfg *config.Config, logger *slog.Logger) Fetcher`
- 添加 `internal/core/fetcher/normal_test.go`
  - 使用 `httptest` mock Bilibili API 响应
  - 测试正常视频、多 P 视频、互动视频
- **对应 C#**: `NormalInfoFetcher.cs`

#### 提交 11: `feat(internal/core/fetcher): implement Bangumi and Cheese fetchers` ✅
- `bangumi.go`：调用 `pgc/view/web/season?ep_id=`
- `cheese.go`：调用 `pugv/view/web/season?ep_id=`
- 更新 Factory 支持 `ep:`, `cheese:` 前缀
- 每个 fetcher 带 `_test.go`，mock 番剧/课程 JSON
- **对应 C#**: `BangumiInfoFetcher.cs`, `CheeseInfoFetcher.cs`

#### 提交 12: `feat(internal/core/fetcher): implement IntlBangumiInfoFetcher` ✅
- `intl_bangumi.go`：调用 `api.bilibili.tv/intl/gateway/v2/ogv/view/app/season`
- 处理 BiliPlus host 和 sign
- Factory 更新
- mock 测试
- **对应 C#**: `IntlBangumiInfoFetcher.cs`

#### 提交 13: `feat(internal/core/fetcher): implement collection and space fetchers` ✅
- `media_list.go`（合集）
- `series_list.go`（系列）
- `fav_list.go`（收藏夹）
- `space_video.go`（用户空间）
- Factory 完整支持所有前缀
- 集成测试（mock 分页 API）
- **对应 C#**: `MediaListFetcher.cs`, `SeriesListFetcher.cs`, `FavListFetcher.cs`, `SpaceVideoFetcher.cs`

#### 提交 14: `feat(internal/core/parser): add playurl API builder with tests` ✅
- 实现 `internal/core/parser/api.go`：
  - `BuildWebPlayurlAPI(aid, cid, epid, qn string) string`
  - `BuildTVPlayurlAPI(...)`（含 sign 计算）
  - `BuildIntlPlayurlAPI(...)`
  - `BuildAppPlayurlAPI(...)`（供 gRPC 用）
- 纯函数，无外部依赖，极易测试
- 添加 `internal/core/parser/api_test.go`
  - 验证生成的 URL 包含正确参数
  - 验证 WBI sign / TV sign 格式
- **对应 C#**: `Parser.cs` 中 API 构建部分

#### 提交 15: `feat(internal/core/parser): add DASH stream parser` ✅
- 实现 `internal/core/parser/dash.go`：
  - `ParseDashTracks(jsonStr string) ([]entity.Video, []entity.Audio, error)`
  - 解析 video/audio/dolby/flac tracks
  - 处理免二压视频（reParse 逻辑）
  - PCDN URL 过滤（`BaseUrlRegex`）
  - 轨道去重
- 添加 `internal/core/parser/dash_test.go`
  - 使用真实 API 响应的脱敏 JSON 作为 testdata
  - 测试杜比、Hi-Res、多编码共存场景
- **对应 C#**: `Parser.cs` 中 DASH 部分

#### 提交 16: `feat(internal/core/parser): add FLV stream parser` ✅
- 实现 `internal/core/parser/flv.go`：
  - `ParseFlvTracks(jsonStr string) (*ParsedResult, error)`
  - 解析 `durl` clips
  - 解析可用清晰度列表（`accept_quality` / `qn_extras`）
- 添加 `internal/core/parser/flv_test.go`
- **对应 C#**: `Parser.cs` 中 FLV 部分

#### 提交 17: `feat(internal/core/parser): add clip info and viewpoint parser` ✅
- 实现 `internal/core/parser/clip.go`：
  - `ParseClipInfoList(jsonStr string) ([]entity.ViewPoint, error)`
  - 番剧片头片尾转 chapter（正片 → 片头 → 正片 → 片尾）
- 添加 `internal/core/parser/clip_test.go`
- **对应 C#**: `Parser.cs` 中 clip_info_list 部分

#### 提交 18: `feat(internal/core/parser): wire up ExtractTracks orchestrator` ✅
- 实现 `internal/core/parser/parser.go`：
  - `ExtractTracks(ctx, aidOri, aid, cid, epid string, opts ExtractOptions) (*ParsedResult, error)`
  - 整合 API builder → HTTP GET → DASH/FLV/INTL 分支 → clip parser
  - 错误处理：大会员限制检测 → fallback 网页源码解析
- 添加 `internal/core/parser/parser_test.go`
  - mock HTTP client，测试各分支逻辑
- **对应 C#**: `Parser.cs` 整体

#### 提交 19: `feat(internal/core/api): add APP gRPC helper and protobuf definitions` ✅
- 从 `BBDown/BBDown.Core/APP/*.proto` 复制到 `internal/core/api/proto/`
- 配置 `buf` 或 `protoc` 生成 Go 代码
- 实现 `internal/core/api/app.go`：
  - `PackMessage` / `ReadMessage`（5 字节头 + gzip）
  - `BuildGRPCHeaders(token string) map[string]string`
  - `ConvertToDashJson(reply *pb.PlayViewReply) string`
- 添加 `internal/core/api/app_test.go`
  - 测试 Pack/Read 往返
  - 测试 ConvertToDashJson 输出包含预期字段
- **关键技术决策**：选 `protoc-gen-go`，从现有 `.proto` 直接生成。添加 `//go:generate` 指令。
- **对应 C#**: `AppHelper.cs`

#### 提交 20: `feat(internal/core/util): add subtitle utilities` ✅
- 实现 `internal/core/util/subtitle.go`：
  - `GetSubtitles(ctx, aid, cid, epid string, isIntl bool) ([]entity.Subtitle, error)`
  - `ConvertSubFromJSON(jsonStr string) (string, error)`（JSON → SRT）
  - 语言代码映射
- 添加 `internal/core/util/subtitle_test.go`
  - mock 字幕 API
  - 测试 JSON→SRT 转换精度
- **对应 C#**: `SubUtil.cs`

---

### Phase 2: 下载与混流引擎（提交 21-29）

本阶段所有模块都是**接口驱动**，便于后续 mock 和服务器模式复用。

#### 提交 21: `feat(internal/download): define Downloader interface and Config` ✅
- 定义 `internal/download/downloader.go`：
  ```go
  type Downloader interface {
      Download(ctx context.Context, url, path string, opts DownloadOptions) error
      DownloadMultiThread(ctx context.Context, url, path string, opts DownloadOptions) error
  }
  type DownloadOptions struct {
      UseAria2c bool; Aria2cArgs string; ForceHTTP bool; MultiThread bool
  }
  ```
- 添加空实现（返回 `errors.New("not implemented")`）
- 添加 `internal/download/downloader_test.go`（接口契约测试）
- **对应 C#**: `BBDownDownloadUtil.cs` 接口抽象

#### 提交 22: `feat(internal/download): implement single-threaded downloader` ✅
- 实现 `internal/download/single.go`：
  - HTTP Range 请求支持
  - 断点续传（检查已下载大小 + Range header）
  - `If-Range` 支持
  - 进度回调 `func(downloaded, total int64)`
- 使用 `httpclient.Client` 接口，不直接依赖 `net/http`
- 添加 `internal/download/single_test.go`
  - `httptest` mock Range 请求
  - 测试断点续传（先写一半 tmp，再恢复）
  - 测试非 Range 支持的服务器 fallback
- **对应 C#**: `BBDownDownloadUtil.cs` 单线程部分

#### 提交 23: `feat(internal/download): implement multi-threaded range download` ✅
- 实现 `internal/download/multithread.go`：
  - 20MB 分片，使用 `errgroup.Group` 并行下载
  - 临时文件命名：`{index}_{filename}.tmp`
  - 下载完成后合并（io.Copy 顺序拼接）
  - 各片段进度聚合回调
- 添加 `internal/download/multithread_test.go`
  - mock 文件大小，mock 各 Range 响应
  - 测试片段合并后内容完整性（md5 校验）
  - 测试某一片段失败时的整体取消（errgroup 行为）
- **对应 C#**: `BBDownDownloadUtil.cs` 多线程部分

#### 提交 24: `feat(internal/download): add aria2c integration` ✅
- 实现 `internal/download/aria2c.go`：
  - `DownloadWithAria2c(url, path, args string) error`
  - 查找 aria2c 可执行文件（当前目录 → 程序目录 → `$PATH`）
  - 调用 `exec.CommandContext`
- 添加 `internal/download/aria2c_test.go`
  - mock `exec.LookPath`（通过接口抽象）
- **对应 C#**: `BBDownAria2c.cs`

#### 提交 25: `feat(internal/download): add progress bar reporter` ✅
- 实现 `internal/download/progress.go`：
  - 封装 `schollz/progressbar/v3`
  - `type Reporter interface { Report(current, total int64) }`
  - `ConsoleReporter`：进度条
  - `ServerReporter`：写入 channel（供服务器模式消费）
  - `NopReporter`：静默模式
- 添加 `internal/download/progress_test.go`
- **对应 C#**: `ProgressBar.cs`

#### 提交 26: `feat(internal/muxer): define Muxer interface` ✅
- 定义 `internal/muxer/muxer.go`：
  ```go
  type Muxer interface {
      Mux(ctx context.Context, cfg MuxConfig) error
      MergeFLV(ctx context.Context, files []string, outPath string) error
  }
  type MuxConfig struct { ... } // 视频/音频/字幕/封面/章节等路径
  ```
- **对应 C#**: `BBDownMuxer.cs` 抽象

#### 提交 27: `feat(internal/muxer): implement ffmpeg muxer` ✅
- 实现 `internal/muxer/ffmpeg.go`：
  - 构造 ffmpeg 参数（-i 输入、-map 映射、-metadata 元数据）
  - 章节元数据文件生成（FFMETADATA 格式）
  - 字幕嵌入、封面嵌入
  - macOS HEVC `hvc1` tag 修复
  - `MergeFLV`：TS 中转合并
  - 使用 `exec.CommandContext`
- 添加 `internal/muxer/ffmpeg_test.go`
  - 测试参数构造（不实际调用 ffmpeg）
  - 测试 FFMETADATA 格式正确性
- **对应 C#**: `BBDownMuxer.cs` ffmpeg 路径

#### 提交 28: `feat(internal/muxer): implement mp4box muxer` ✅
- 实现 `internal/muxer/mp4box.go`
- 添加参数构造测试
- **对应 C#**: `BBDownMuxer.cs` mp4box 路径

#### 提交 29: `feat(internal/muxer): add external binary finder` ✅
- 实现 `internal/muxer/finder.go`：
  - `FindExecutable(name string) (string, error)`
  - 搜索路径：当前目录 → 程序目录 → `$PATH`
  - `CheckFFmpegDOVI() bool`（检测 libavutil 版本 ≥ 57.17）
- 添加 `internal/muxer/finder_test.go`
- **对应 C#**: `BBDownUtil.FindExecutable`, `CheckFFmpegDOVI`

---

### Phase 3: CLI 与配置（提交 30-36）

#### 提交 30: `feat(internal/cli): add cobra root command and flags` ✅
- 使用 `spf13/cobra` 实现 root command
- 定义所有 flags（复刻 `MyOption`）
- 添加 `internal/cli/cli_test.go`
  - 测试 flag 解析正确性
- **对应 C#**: `CommandLineInvoker.cs`, `MyOption.cs`

#### 提交 31: `feat(internal/cli): add login and serve subcommands` ✅
- `login` subcommand
- `logintv` subcommand
- `serve` subcommand（带 `--listen` flag）
- 添加子命令解析测试

#### 提交 32: `feat(internal/config): add config file parser` ✅
- 实现 `internal/config/parser.go`：
  - 读取 `BBDown.config`
  - `#` 注释支持
  - `--key value` 格式解析
  - 合并到命令行参数（命令行优先级 > 配置文件）
- 添加 `internal/config/parser_test.go`
- **对应 C#**: `BBDownConfigParser.cs`

#### 提交 33: `feat(internal/app): add URL parser` ✅
- 实现 `internal/app/url_parser.go`：
  - `ParseInput(input string) (string, error)`
  - 支持：BV / AV / EP / SS / MD / cheese / mid / favId / listBizId / seriesBizId / 完整 URL
  - b23.tv 短链解析
- 添加 `internal/app/url_parser_test.go`
  - 每种 URL 格式至少一个测试用例
- **对应 C#**: `BBDownUtil.GetAvIdAsync`

#### 提交 34: `feat(internal/app): add save path formatter and track sorter` ✅
- `formatter.go`：`FormatSavePath(format, title, video, audio, page, ...)`
- `sorter.go`：
  - `SortVideoTracks(tracks, dfnPriority, encodingPriority, ascending)`
  - `SortAudioTracks(tracks, encodingPriority, ascending)`
- 添加单元测试
- **对应 C#**: `Program.FormatSavePath`, `Program.SortTracks`

#### 提交 35: `feat(internal/app): add setup and validation logic` ✅
- 实现 `internal/app/setup.go`：
  - `SetupWork(opt *cli.Option, logger *slog.Logger) (*WorkConfig, error)`
  - 废弃选项处理、冲突选项处理
  - 查找外部二进制（ffmpeg, mp4box, aria2c）
  - 切换工作目录
  - 检查登录状态（`CheckLogin`）
  - WBI key 获取
- 返回 `WorkConfig` 结构体，而非元组，避免 Go 多返回值过长
- 添加 `internal/app/setup_test.go`
- **对应 C#**: `Program.SetUpWork`, `BBDownUtil.CheckLogin`

#### 提交 36: `feat(internal/app): add GetVideoInfo workflow`
- 实现 `internal/app/info.go`：
  - `GetVideoInfo(ctx, opt, input string, deps Deps) (fetchedAid string, vInfo *entity.VInfo, apiType string, err error)`
  - `Deps` 结构体注入 fetcher factory、http client、logger（便于测试 mock）
  - 加载认证、检测登录、Factory 路由、EP/SS 回退 cheese、互动视频 TV API 降级
- 添加 `internal/app/info_test.go`
  - mock fetcher factory，测试路由逻辑
  - 测试回退逻辑
- **对应 C#**: `Program.GetVideoInfoAsync`

---

### Phase 4: 主工作流编排（提交 37-42）

#### 提交 37: `feat(internal/app): add DownloadPage orchestrator - part 1 (info & subtitle)` ✅
- 实现 `internal/app/download_page.go` 的**前半部分**：
  - 获取章节信息（`FetchPoints`）
  - 下载封面
  - 获取并下载字幕
  - 仅字幕/仅封面模式的早期返回
- 添加 `internal/app/download_page_test.go`
  - mock HTTP client 测试封面下载逻辑
- **对应 C#**: `Program.DownloadPageAsync` 前半部分

#### 提交 38: `feat(internal/app): add DownloadPage orchestrator - part 2 (stream & selection)`
- 实现**中间部分**：
  - 调用 `parser.ExtractTracks`
  - 下载弹幕 XML → ASS
  - 轨道排序
  - 交互式选择（`--interactive`）
  - 轨道过滤（`--video-only`, `--audio-only`）
- 测试：mock parser，测试排序和过滤逻辑
- **对应 C#**: `Program.DownloadPageAsync` 中段

#### 提交 39: `feat(internal/app): add DownloadPage orchestrator - part 3 (download & mux)`
- 实现**后半部分**：
  - 下载视频/音频/背景音/配音
  - 调用 muxer
  - 清理临时文件
  - 重试逻辑（最多 3 次，用 `for retry := 0; retry < 3; retry++`）
- 测试：mock downloader 和 muxer，验证调用顺序
- **对应 C#**: `Program.DownloadPageAsync` 后半部分

#### 提交 40: `feat(internal/app): add DownloadPages batch workflow`
- 实现 `internal/app/download_pages.go`：
  - 解析 `--select-page`
  - 单页/多页路径选择
  - 逐页下载（支持 `--delay-per-page`）
  - 归档文件去重（`--save-archives-to-file`）
- 测试：mock `DownloadPage` 调用
- **对应 C#**: `Program.DownloadPagesAsync`

#### 提交 41: `feat(cmd/bbdown): wire up main entry point`
- 实现 `cmd/bbdown/main.go`：
  - `main()` → cobra → `RunApp()`
  - `RunApp()` → `CheckUpdate` + `DoWork()`
  - `DoWork()` → `SetupWork` → `GetVideoInfo` → `DownloadPages`
  - Ctrl+C 信号处理（`signal.NotifyContext`）
  - 全局 panic recover + `slog.Error`
- **对应 C#**: `Program.Main`, `Program.RunApp`, `Program.DoWorkAsync`

#### 提交 42: `feat(internal/app): add update checker` ✅ ✅
- 实现 `internal/app/update.go`：
  - `CheckUpdate(ctx context.Context, client httpclient.Client) (string, error)`
  - 异步检查 GitHub releases latest redirect
- 添加 `internal/app/update_test.go`
- **对应 C#**: `BBDownUtil.CheckUpdateAsync`

---

### Phase 5: 高级功能（提交 43-50）

#### 提交 43: `feat(internal/login): add WEB QR code login` ✅
- 实现 `internal/login/web.go`
  - 获取登录 URL
  - 生成二维码（控制台字符画 + 图片文件）
  - 轮询登录状态（带 context 取消）
  - 保存 Cookie 到 `BBDown.data`
- 构造函数注入 `httpclient.Client`
- 添加 `internal/login/web_test.go`（mock 轮询 API）
- **对应 C#**: `BBDownLoginUtil.cs`（WEB 部分）

#### 提交 44: `feat(internal/login): add TV QR code login` ✅
- 实现 `internal/login/tv.go`
  - TV 端扫码登录流程
  - AccessToken 获取
  - 保存到 `BBDownTV.data`
- 添加测试
- **对应 C#**: `BBDownLoginUtil.cs`（TV 部分）

#### 提交 45: `feat(internal/danmaku): add danmaku XML parser and ASS converter` ✅ ✅
- `internal/danmaku/parse.go`：`ParseXML(xmlPath string) ([]DanmakuItem, error)`
- `internal/danmaku/ass.go`：`SaveAsAss(items []DanmakuItem, outputPath string) error`
- `internal/danmaku/position.go`：`PositionController`（碰撞检测）
- 添加 `internal/danmaku/danmaku_test.go`
  - 使用 testdata 中的真实弹幕 XML
  - 测试 ASS 输出格式
- **对应 C#**: `DanmakuUtil.cs`

#### 提交 46: `feat(internal/server): add HTTP API server scaffold`
- 使用 `gin` 或标准库 `net/http`
- **建议用 `gin`**：虽然标准库也能做，但 gin 的中间件、路由、JSON binding 能大幅减少样板代码，且 BBDown 的服务器模式只是辅助功能，不需要极致轻量。
- 实现 `internal/server/server.go`：
  - `type Server struct { ... }`
  - `SetupServer()` 注册路由
  - `Run(listenUrl string)`
- 定义任务结构体 `DownloadTask`
- 添加 `internal/server/server_test.go`
- **对应 C#**: `BBDownApiServer.cs`

#### 提交 47: `feat(internal/server): add task queue and progress tracking`
- 任务队列管理（内存 map + sync.RWMutex）
- 进度追踪接口
- Webhook 回调（`--webhook-url`）
- 添加并发安全测试

#### 提交 48: `feat(internal/server): wire server mode into CLI`
- `serve` subcommand 真正启动服务器
- 服务器模式下复用 `internal/app` 的 workflow（通过接口调用）

#### 提交 49: `feat(internal/login): add console QR code renderer` ✅ ✅
- `internal/login/qrcode.go`：控制台字符二维码渲染
- 添加 `internal/login/qrcode_test.go`
- **对应 C#**: `ConsoleQRCode.cs`

#### 提交 50: `refactor: unify error wrapping and add domain error types` ✅
- 在 `internal/core/entity/errors.go` 中定义：
  ```go
  var (
      ErrLoginRequired = errors.New("login required")
      ErrRegionBlocked = errors.New("content blocked in current region")
      ErrParseFailed   = errors.New("failed to parse video streams")
  )
  ```
- 全量审查所有 `return err`，补 `%w` 包装
- 审查所有 `context.Context` 传递，补全缺失点
- **此提交不改逻辑，只改错误处理**

---

### Phase 6: 集成测试、CI 与文档（提交 51-56）

#### 提交 51: `test: add integration test for complete download workflow`
- `tests/integration/download_test.go`
  - 使用 `httptest` mock 整个 Bilibili API 链路
  - mock ffmpeg/mp4box/aria2c（通过可执行文件替换为 shell 脚本）
  - 验证：输入 URL → 下载完成 → 输出文件存在
- 环境变量 `BBDOWN_TEST_NETWORK=1` 控制是否跑真实网络测试（默认跳过）

#### 提交 52: `test: add benchmark for BV codec and parser`
- `pkg/bvconv/bvconv_bench_test.go`
- `internal/core/parser/parser_bench_test.go`

#### 提交 53: `ci: add GitHub Actions workflow`
- `.github/workflows/ci.yml`：
  - Go 构建（Linux / macOS / Windows）
  - `golangci-lint`（启用 `errcheck`, `govet`, `staticcheck`, ` ineffassign`）
  - 单元测试 + race detector（`go test -race ./...`）
  - 覆盖率报告（`go test -coverprofile`）
  - **Conventional Commits 检查**：
    - 使用 `wagoid/commitlint-github-action@v5` 检查 PR 中所有 commit message
    - 或在 CI 中运行 `npx commitlint --from=HEAD~${{ github.event.pull_request.commits }} --to=HEAD`
- `.github/workflows/release.yml`：
  - goreleaser 自动打包（含 upx 压缩可选）

#### 提交 54: `docs: add README and architecture decision records`
- `README.md`：功能对照表、安装说明、用法示例
- `docs/adr/001-logging.md`：为什么选 slog
- `docs/adr/002-http-client.md`：为什么不用 retryablehttp
- `docs/adr/003-testing-strategy.md`：接口 + mock 的测试策略

#### 提交 55: `perf: add pprof endpoints and connection pool tuning`
- 服务器模式暴露 `/debug/pprof`
- HTTP client 连接池参数调优
- 下载 goroutine 数限制（避免过多并发触发 B站风控）

#### 提交 56: `chore: release v0.1.0`
- 打 tag
- 更新 CHANGELOG

---

## 五、可观测性最佳实践详解

### 5.1 日志分级策略

原 C# 代码只有 `Log()` 和 `LogDebug()` 两级，Go 版本应严格使用四级：

| 级别 | 用途 | 示例 |
|------|------|------|
| `DEBUG` | 开发调试，包含原始 JSON、API URL、header | `slog.Debug("playurl response", "json", resp)` |
| `INFO` | 用户可见的正常流程信息 | `slog.Info("开始下载", "page", p.Index, "url", url)` |
| `WARN` | 非致命异常、降级行为 | `slog.Warn("未登录，解析可能受限")` |
| `ERROR` | 致命错误，操作失败 | `slog.Error("混流失败", "err", err, "path", outPath)` |

**关键规则**：
- 所有 `slog.Error` 必须包含 `err` 字段
- 用 `slog.With("aid", aid).With("cid", cid)` 创建子 logger，避免重复字段
- 服务器模式下自动切换为 JSON Handler，便于 ELK/Loki 采集

### 5.2 测试策略

| 层级 | 技术 | 覆盖率目标 |
|------|------|-----------|
| 单元测试 | `testing` + `httptest` + 接口 mock | ≥ 80% |
| 集成测试 | `tests/integration/` + shell mock 外部二进制 | 核心链路 1 条 |
| 并发测试 | `go test -race` | 所有含 goroutine 的提交 |

**Mock 策略**：
- HTTP：mock `httpclient.Client` 接口
- 外部命令：mock `muxer.Muxer` / `downloader.Downloader` 接口
- 文件系统：对简单场景直接用 `os.MkdirTemp`，复杂场景用 `afero`（可选）

### 5.3 Context 使用规范

```go
// ✅ 正确：context 作为第一参数，贯穿 IO 全链路
func (f *NormalFetcher) Fetch(ctx context.Context, id string) (*VInfo, error) {
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    // ...
}

// ✅ 正确：超时控制
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
result, err := fetcher.Fetch(ctx, id)

// ✅ 正确：取消信号
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
defer stop()
```

---

## 六、与原版 C# 的已知差异

1. **配置管理**：Go 版完全摒弃全局静态变量，所有配置通过参数传递
2. **并发模型**：C# `async/await` + `Parallel.ForEachAsync` → Go `context` + `errgroup`
3. **错误处理**：C# 裸 `throw new Exception(...)` → Go 明确的 `error` 返回值 + 领域错误常量
4. **日志**：C# 自定义静态方法 → Go 标准库 `slog` + 结构化字段
5. **HTTP**：C# 单个全局 `HttpClient` → Go 封装接口 + 连接池调优 + 自动重试
6. **服务器模式**：C# ASP.NET Core Minimal APIs → Go `gin` + 内存任务队列

---

## 七、验证检查点

1. **Phase 0 结束**: `go test ./...` 全绿，`go build ./cmd/bbdown` 成功
2. **Phase 1 结束**: 能用 mock HTTP 测试完整验证：给定 aid → Fetcher 返回 VInfo → Parser 返回 ParsedResult
3. **Phase 2 结束**: mock Downloader 和 Muxer，验证下载和混流调用链正确
4. **Phase 3 结束**: CLI 参数解析与原版行为一致，config file 解析正确
5. **Phase 4 结束**: 集成测试通过：mock API → 输出文件（不调用真实 ffmpeg）
6. **Phase 5 结束**: 登录流程 mock 测试通过，服务器模式启动成功
7. **Phase 6 结束**: CI 通过，`-race` 无竞争，跨平台编译成功
