# ADR 003: 接口 + Mock 的测试策略

## Status

Accepted

## Context

BBDown 是一个重度依赖网络 IO 的命令行下载器：
- 请求 Bilibili API（会变化、有速率限制）。
- 下载大文件到本地磁盘。
- 调用外部二进制（FFmpeg、aria2c、MP4Box）。

如果在单元测试中直接使用真实网络或外部进程，测试会变得缓慢、不稳定且难以在 CI 中复现。

## Decision

**所有 IO 依赖均通过接口注入**，绝不直接在业务代码中调用具体实现。

关键接口包括：

| 模块 | 接口 | 职责 |
|---|---|---|
| `pkg/httpclient` | `Client` | HTTP 请求（含重试、Cookie） |
| `internal/core/fetcher` | `Fetcher` | 业务数据获取（视频信息、播放 URL 等） |
| `internal/download` | `Downloader` | 文件下载（单线程、多线程、aria2c） |
| `internal/muxer` | `Muxer` | 音视频封装（FFmpeg、MP4Box） |

测试策略：
- 单元测试：100% 使用 mock 实现，覆盖正常路径、错误路径和边界条件。
- 集成测试：保留在 `tests/integration/`，仅在需要时手动触发，不阻塞 CI。
- 外部命令调用（FFmpeg、aria2c）统一通过接口抽象，测试中注入 fake 进程行为。

## Consequences

### Positive

- **80%+ 单元测试覆盖率**：所有核心 fetcher、parser、sorter、formatter、muxer 均可在毫秒级完成测试。
- **无真实网络**：CI 完全离线运行，不受 B 站 API 变更或网络波动影响。
- **快速反馈**：开发者本地运行 `go test ./...` 即可在数秒内获得完整反馈。

### Negative

- **接口开销**：需要为每个外部依赖定义接口和 mock，增加了少量 boilerplate。
- **mock 与真实行为漂移**：若 mock 未准确模拟外部系统行为，可能出现“测试通过但线上失败”的情况。需要定期通过集成测试或契约测试进行校准。
