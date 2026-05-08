package download

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/sailist/BBDown-go/pkg/httpclient"
)

type mockClient struct {
	mu sync.Mutex

	getFunc              func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error)
	headFunc             func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error)
	getContentLengthFunc func(ctx context.Context, url string) (int64, error)

	getCalls      int
	getCallsSlice []mockGetCall
}

type mockGetCall struct {
	url  string
	opts []httpclient.RequestOption
}

func newMockClient() *mockClient {
	return &mockClient{}
}

func (m *mockClient) Get(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	m.mu.Lock()
	m.getCalls++
	m.getCallsSlice = append(m.getCallsSlice, mockGetCall{url: url, opts: opts})
	m.mu.Unlock()
	if m.getFunc != nil {
		return m.getFunc(ctx, url, opts...)
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
}

func (m *mockClient) Post(ctx context.Context, url string, body []byte, opts ...httpclient.RequestOption) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
}

func (m *mockClient) Head(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	if m.headFunc != nil {
		return m.headFunc(ctx, url, opts...)
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
}

func (m *mockClient) GetRedirectLocation(ctx context.Context, url string) (string, error) {
	return url, nil
}

func (m *mockClient) GetContentLength(ctx context.Context, url string) (int64, error) {
	if m.getContentLengthFunc != nil {
		return m.getContentLengthFunc(ctx, url)
	}
	return 0, nil
}
