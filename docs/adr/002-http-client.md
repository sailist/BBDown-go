# ADR 002: 为什么不用 retryablehttp

## Status

Accepted

## Context

下载器极度依赖 HTTP 请求：获取视频信息、解析播放流、下载分片、拉取弹幕与字幕等。网络抖动和 B 站 CDN 偶发的 403/503 要求客户端具备重试、Cookie 透传、超时控制、自定义 Host 替换等能力。

社区常见方案是直接引入 `hashicorp/go-retryablehttp`，它提供了自动重试、退避策略和丰富的拦截器。

## Decision

不引入 `retryablehttp`，而是定义自定义接口 **`httpclient.Client`**，底层基于标准库 **`net/http`** 实现。

```go
type Client interface {
    Do(req *http.Request) (*http.Response, error)
}
```

- 在 `pkg/httpclient/standard.go` 中封装带重试、超时、Cookie Jar 的标准实现。
- 接口对外暴露，所有内部模块（`fetcher`、`download` 等）均通过该接口发起请求。

## Consequences

### Positive

- **完全可控**：重试策略、退避算法、请求拦截、Host 替换均可按需定制，不受第三方库限制。
- **易于测试**：接口化后，单元测试可直接注入 `httpclient.Client` 的 mock 实现，无需真实网络。
- **减少依赖**：避免引入 `go-retryablehttp` 及其传递依赖。

### Negative

- **更多自有代码**：需要自行维护重试逻辑、连接池调优和边缘 case（如 HTTP/2 退避）。
- **重复造轮子风险**：标准库 `net/http` 的默认行为在某些场景下（如 0-length body retry）需要额外处理。
