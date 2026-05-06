package httpclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/exp/slog"
)

var seedCounter int64

var platforms = []string{
	"Windows NT 10.0; Win64",
	"Macintosh; Intel Mac OS X 10_15",
	"X11; Linux x86_64",
}

func randomVersion(min, max int, rnd *rand.Rand) string {
	version := rnd.Float64()*float64(max-min) + float64(min)
	return fmt.Sprintf("%.3f", version)
}

func getRandomUserAgent(rnd *rand.Rand) string {
	browsers := []string{
		fmt.Sprintf("AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", randomVersion(80, 110, rnd)),
		fmt.Sprintf("Gecko/20100101 Firefox/%s", randomVersion(80, 110, rnd)),
	}
	return fmt.Sprintf("Mozilla/5.0 (%s) %s", platforms[rnd.Intn(len(platforms))], browsers[rnd.Intn(len(browsers))])
}

type StandardClient struct {
	client    *http.Client
	logger    *slog.Logger
	userAgent string
}

func NewStandardClient(logger *slog.Logger) *StandardClient {
	seed := time.Now().UnixNano() + atomic.AddInt64(&seedCounter, 1)
	rnd := rand.New(rand.NewSource(seed))

	tr := &http.Transport{
		MaxIdleConns:    100,
		IdleConnTimeout: 90 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return &StandardClient{
		client: &http.Client{
			Transport: tr,
			Timeout:   2 * time.Minute,
		},
		logger:    logger,
		userAgent: getRandomUserAgent(rnd),
	}
}

func (c *StandardClient) Get(ctx context.Context, url string, opts ...RequestOption) (*http.Response, error) {
	return c.request(ctx, http.MethodGet, url, nil, opts...)
}

func (c *StandardClient) Post(ctx context.Context, url string, body []byte, opts ...RequestOption) (*http.Response, error) {
	return c.request(ctx, http.MethodPost, url, body, opts...)
}

func (c *StandardClient) Head(ctx context.Context, url string, opts ...RequestOption) (*http.Response, error) {
	return c.request(ctx, http.MethodHead, url, nil, opts...)
}

func (c *StandardClient) GetRedirectLocation(ctx context.Context, url string) (string, error) {
	resp, err := c.request(ctx, http.MethodHead, url, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.Request != nil && resp.Request.URL != nil {
		return resp.Request.URL.String(), nil
	}
	return url, nil
}

func (c *StandardClient) GetContentLength(ctx context.Context, url string) (int64, error) {
	resp, err := c.request(ctx, http.MethodHead, url, nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.ContentLength, nil
}

func (c *StandardClient) request(ctx context.Context, method, urlStr string, body []byte, opts ...RequestOption) (*http.Response, error) {
	var resp *http.Response
	var lastErr error

	for attempt := 1; attempt <= 3; attempt++ {
		var bodyReader io.Reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
		}

		req, err := http.NewRequestWithContext(ctx, method, urlStr, bodyReader)
		if err != nil {
			return nil, err
		}

		// Default headers
		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Cache-Control", "no-cache")

		if strings.Contains(urlStr, "api.bilibili.com") {
			req.Header.Set("Referer", "https://www.bilibili.com/")
		}
		if strings.Contains(urlStr, "/ep") || strings.Contains(urlStr, "/ss") {
			cookie := req.Header.Get("Cookie")
			if cookie != "" {
				req.Header.Set("Cookie", cookie+";CURRENT_FNVAL=4048;")
			} else {
				req.Header.Set("Cookie", ";CURRENT_FNVAL=4048;")
			}
		}

		// Apply options
		for _, opt := range opts {
			opt(req)
		}

		c.logger.Debug("HTTP request",
			slog.String("method", req.Method),
			slog.String("url", req.URL.String()),
			slog.Int("attempt", attempt),
		)

		resp, lastErr = c.client.Do(req)
		if lastErr == nil && resp.StatusCode < 400 {
			c.logger.Debug("HTTP response",
				slog.Int("status", resp.StatusCode),
			)
			return resp, nil
		}

		if lastErr != nil {
			c.logger.Debug("HTTP request failed",
				slog.String("error", lastErr.Error()),
				slog.Int("attempt", attempt),
			)
		} else {
			c.logger.Debug("HTTP request failed",
				slog.Int("status", resp.StatusCode),
				slog.Int("attempt", attempt),
			)
		}

		if attempt < 3 {
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return resp, nil
}
