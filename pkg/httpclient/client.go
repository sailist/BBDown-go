package httpclient

import (
	"context"
	"net/http"
)

type Client interface {
	Get(ctx context.Context, url string, opts ...RequestOption) (*http.Response, error)
	Post(ctx context.Context, url string, body []byte, opts ...RequestOption) (*http.Response, error)
	Head(ctx context.Context, url string, opts ...RequestOption) (*http.Response, error)
	GetRedirectLocation(ctx context.Context, url string) (string, error)
	GetContentLength(ctx context.Context, url string) (int64, error)
}

type RequestOption func(*http.Request)

func WithHeader(key, value string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set(key, value)
	}
}

func WithCookie(cookie string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set("Cookie", cookie)
	}
}

func WithReferer(referer string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set("Referer", referer)
	}
}

func WithUserAgent(ua string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set("User-Agent", ua)
	}
}
