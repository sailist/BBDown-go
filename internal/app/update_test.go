package app

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/nilaonai/bbdown-go/pkg/httpclient"
)

type mockClient struct {
	getFunc                 func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error)
	postFunc                func(ctx context.Context, url string, body []byte, opts ...httpclient.RequestOption) (*http.Response, error)
	headFunc                func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error)
	getRedirectLocationFunc func(ctx context.Context, url string) (string, error)
	getContentLengthFunc    func(ctx context.Context, url string) (int64, error)
}

func (m *mockClient) Get(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, url, opts...)
	}
	return nil, errors.New("not implemented")
}

func (m *mockClient) Post(ctx context.Context, url string, body []byte, opts ...httpclient.RequestOption) (*http.Response, error) {
	if m.postFunc != nil {
		return m.postFunc(ctx, url, body, opts...)
	}
	return nil, errors.New("not implemented")
}

func (m *mockClient) Head(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	if m.headFunc != nil {
		return m.headFunc(ctx, url, opts...)
	}
	return nil, errors.New("not implemented")
}

func (m *mockClient) GetRedirectLocation(ctx context.Context, url string) (string, error) {
	if m.getRedirectLocationFunc != nil {
		return m.getRedirectLocationFunc(ctx, url)
	}
	return "", errors.New("not implemented")
}

func (m *mockClient) GetContentLength(ctx context.Context, url string) (int64, error) {
	if m.getContentLengthFunc != nil {
		return m.getContentLengthFunc(ctx, url)
	}
	return 0, errors.New("not implemented")
}

func TestCheckUpdate_Success(t *testing.T) {
	client := &mockClient{
		getRedirectLocationFunc: func(ctx context.Context, url string) (string, error) {
			return "https://github.com/nilaoda/BBDown/releases/tag/1.6.3", nil
		},
	}

	version, err := CheckUpdate(context.Background(), client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "1.6.3" {
		t.Fatalf("expected version 1.6.3, got %s", version)
	}
}

func TestCheckUpdate_NetworkError(t *testing.T) {
	expectedErr := errors.New("connection refused")
	client := &mockClient{
		getRedirectLocationFunc: func(ctx context.Context, url string) (string, error) {
			return "", expectedErr
		},
	}

	_, err := CheckUpdate(context.Background(), client)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error to wrap %v, got %v", expectedErr, err)
	}
}

func TestCheckUpdate_MalformedURL(t *testing.T) {
	client := &mockClient{
		getRedirectLocationFunc: func(ctx context.Context, url string) (string, error) {
			return "https://example.com/something", nil
		},
	}

	_, err := CheckUpdate(context.Background(), client)
	if err == nil {
		t.Fatal("expected error for malformed URL, got nil")
	}
}
