package login

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nilaonai/bbdown-go/pkg/httpclient"
)

type mockClient struct {
	getFunc func(url string) (*http.Response, error)
	calls   []string
}

func (m *mockClient) Get(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	m.calls = append(m.calls, url)
	if m.getFunc != nil {
		return m.getFunc(url)
	}
	return nil, fmt.Errorf("no getFunc set")
}

func (m *mockClient) Post(ctx context.Context, url string, body []byte, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, fmt.Errorf("Post not implemented")
}

func (m *mockClient) Head(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, fmt.Errorf("Head not implemented")
}

func (m *mockClient) GetRedirectLocation(ctx context.Context, url string) (string, error) {
	return "", fmt.Errorf("GetRedirectLocation not implemented")
}

func (m *mockClient) GetContentLength(ctx context.Context, url string) (int64, error) {
	return 0, fmt.Errorf("GetContentLength not implemented")
}

func newMockClientWithResponse(body string) *mockClient {
	return &mockClient{
		getFunc: func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		},
	}
}

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestWebLogin_generateQRCode(t *testing.T) {
	generateResp := `{"data":{"url":"https://passport.bilibili.com/x/passport-login/web/qrcode/generate?source=main-fe-header&qrcode_key=abc123","qrcode_key":"abc123"}}`
	client := newMockClientWithResponse(generateResp)
	wl := NewWebLogin(client, noopLogger())

	loginURL, key, err := wl.generateQRCode(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if loginURL == "" {
		t.Fatal("expected non-empty login URL")
	}
	if key != "abc123" {
		t.Fatalf("expected qrcode_key=abc123, got: %s", key)
	}
}

func TestWebLogin_generateQRCode_EmptyResponse(t *testing.T) {
	generateResp := `{"data":{"url":"","qrcode_key":""}}`
	client := newMockClientWithResponse(generateResp)
	wl := NewWebLogin(client, noopLogger())

	_, _, err := wl.generateQRCode(context.Background())
	if err == nil {
		t.Fatal("expected error for empty response, got nil")
	}
}

func TestWebLogin_extractCookie(t *testing.T) {
	wl := NewWebLogin(nil, noopLogger())

	tests := []struct {
		name        string
		redirectURL string
		want        string
		wantErr     bool
	}{
		{
			name:        "standard URL",
			redirectURL: "https://www.bilibili.com?SESSDATA=abc123&bili_jct=xyz",
			want:        "SESSDATA=abc123;bili_jct=xyz",
		},
		{
			name:        "URL with comma",
			redirectURL: "https://www.bilibili.com?SESSDATA=a,b,c&bili_jct=xyz",
			want:        "SESSDATA=a%2Cb%2Cc;bili_jct=xyz",
		},
		{
			name:        "no query string",
			redirectURL: "https://www.bilibili.com",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := wl.extractCookie(tt.redirectURL)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestWebLogin_saveCookie(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(origWd)

	wl := NewWebLogin(nil, noopLogger())
	cookie := "SESSDATA=abc123;bili_jct=xyz"
	if err := wl.saveCookie(cookie); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	path := filepath.Join(tmpDir, cookieFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read cookie file: %v", err)
	}
	if string(data) != cookie {
		t.Fatalf("expected %q, got %q", cookie, string(data))
	}
}

func TestWebLogin_pollLoginStatus_Success(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(origWd)

	callCount := 0
	client := &mockClient{
		getFunc: func(url string) (*http.Response, error) {
			if strings.Contains(url, "generate") {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"url":"https://passport.bilibili.com/x/passport-login/web/qrcode/generate?source=main-fe-header&qrcode_key=testkey","qrcode_key":"testkey"}}`)),
				}, nil
			}
			callCount++
			switch callCount {
			case 1:
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"code":86101}}`)),
				}, nil
			case 2:
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"code":86090}}`)),
				}, nil
			case 3:
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"code":0,"url":"https://www.bilibili.com?SESSDATA=abc123&bili_jct=xyz"}}`)),
				}, nil
			default:
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"code":0,"url":"https://www.bilibili.com?SESSDATA=abc123&bili_jct=xyz"}}`)),
				}, nil
			}
		},
	}

	wl := NewWebLogin(client, noopLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = wl.Login(ctx)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	path := filepath.Join(tmpDir, cookieFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read cookie file: %v", err)
	}
	want := "SESSDATA=abc123;bili_jct=xyz"
	if string(data) != want {
		t.Fatalf("expected cookie %q, got %q", want, string(data))
	}
}

func TestWebLogin_pollLoginStatus_Expired(t *testing.T) {
	client := &mockClient{
		getFunc: func(url string) (*http.Response, error) {
			if strings.Contains(url, "generate") {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"url":"https://passport.bilibili.com/x/passport-login/web/qrcode/generate?source=main-fe-header&qrcode_key=testkey","qrcode_key":"testkey"}}`)),
				}, nil
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"data":{"code":86038}}`)),
			}, nil
		},
	}

	wl := NewWebLogin(client, noopLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := wl.Login(ctx)
	if !errors.Is(err, ErrQRExpired) {
		t.Fatalf("expected ErrQRExpired, got: %v", err)
	}
}

func TestWebLogin_pollLoginStatus_ContextCancel(t *testing.T) {
	client := &mockClient{
		getFunc: func(url string) (*http.Response, error) {
			if strings.Contains(url, "generate") {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"url":"https://passport.bilibili.com/x/passport-login/web/qrcode/generate?source=main-fe-header&qrcode_key=testkey","qrcode_key":"testkey"}}`)),
				}, nil
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"data":{"code":86101}}`)),
			}, nil
		},
	}

	wl := NewWebLogin(client, noopLogger())

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(1500 * time.Millisecond)
		cancel()
	}()

	err := wl.Login(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestWebLogin_maskSESSDATA(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"SESSDATA=abcdef;bili_jct=xyz", "ab***ef"},
		{"SESSDATA=ab;bili_jct=xyz", "***"},
		{"SESSDATA=abc;bili_jct=xyz", "***"},
		{"bili_jct=xyz", ""},
	}

	for _, tt := range tests {
		got := maskSESSDATA(tt.input)
		if got != tt.want {
			t.Fatalf("maskSESSDATA(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
