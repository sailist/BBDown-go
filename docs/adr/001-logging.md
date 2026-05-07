# ADR 001: 为什么选 slog

## Status

Accepted

## Context

C# 版本的 BBDown 使用了一套自定义的静态日志工具，通过全局方法输出文本到控制台。在 Go 重写过程中，我们需要一个既能满足控制台可读性、又能在服务器模式 (`serve`) 下输出结构化 JSON 日志的方案。

社区中有多个成熟的日志库（如 `zap`、`logrus`），但 Go 1.21 起标准库引入了 `log/slog`，提供了结构化日志的原生支持，包括 JSON Handler 和 Level 控制。

## Decision

使用 Go 标准库 **`log/slog`** 作为项目唯一的日志框架，通过 `pkg/logger` 封装统一的 Handler 初始化逻辑。

- 默认文本输出到控制台（开发调试）。
- `serve` 模式或需要时切换为 JSON Handler。
- 所有日志字段使用结构化 `slog.Info("msg", "key", value)` 风格。

## Consequences

### Positive

- **零外部依赖**：标准库自带，减少模块体积和供应链风险。
- **结构化**：原生支持 `slog.Attr` 和 JSON 输出，便于后续对接日志收集系统。
- **统一接口**：`context.Context` 与 `slog` 天然配合，方便后续追加 trace ID 等字段。

### Negative

- **无默认彩色控制台输出**：`slog.TextHandler` 默认不带颜色，如需彩色需自行封装 Handler。
- **生态工具较少**：相比 `zap` 的丰富生态（如 `zaprotate`），标准库周边工具尚在成长。
