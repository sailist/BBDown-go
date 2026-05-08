package login

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sailist/BBDown-go/internal/core/entity"
	"github.com/sailist/BBDown-go/pkg/httpclient"
)

type mockTVHTTPClient struct {
	responses []struct {
		resp *http.Response
		err  error
	}
	index int
}

func (m *mockTVHTTPClient) Get(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, errors.New("unexpected GET")
}

func (m *mockTVHTTPClient) Post(ctx context.Context, url string, body []byte, opts ...httpclient.RequestOption) (*http.Response, error) {
	if m.index >= len(m.responses) {
		return nil, errors.New("no more mock responses")
	}
	r := m.responses[m.index]
	m.index++
	return r.resp, r.err
}

func (m *mockTVHTTPClient) Head(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, errors.New("unexpected HEAD")
}

func (m *mockTVHTTPClient) GetRedirectLocation(ctx context.Context, url string) (string, error) {
	return "", errors.New("unexpected GetRedirectLocation")
}

func (m *mockTVHTTPClient) GetContentLength(ctx context.Context, url string) (int64, error) {
	return 0, errors.New("unexpected GetContentLength")
}

func newMockTVResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestGetTVLoginParms(t *testing.T) {
	parms := GetTVLoginParms()

	required := []string{"appkey", "bili_local_id", "build", "buvid", "channel",
		"device", "device_id", "device_name", "device_platform", "fingerprint",
		"guid", "local_fingerprint", "local_id", "mobi_app", "networkstate",
		"platform", "sys_ver", "ts", "sign"}

	for _, key := range required {
		if parms.Get(key) == "" {
			t.Errorf("missing required param: %s", key)
		}
	}

	if parms.Get("appkey") != "4409e2ce8ffd12b8" {
		t.Errorf("appkey = %q, want %q", parms.Get("appkey"), "4409e2ce8ffd12b8")
	}

	if parms.Get("sign") == "" {
		t.Error("sign is empty")
	}

	if parms.Get("auth_code") != "" {
		t.Errorf("auth_code should be empty initially, got %q", parms.Get("auth_code"))
	}
}

func TestTVLogin_getAuthCode(t *testing.T) {
	mockBody := `{"data":{"url":"https://example.com/qr","auth_code":"abc123"}}`
	client := &mockTVHTTPClient{
		responses: []struct {
			resp *http.Response
			err  error
		}{{resp: newMockTVResponse(mockBody)}},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tv := NewTVLogin(client, logger, t.TempDir())

	url, authCode, err := tv.getAuthCode(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if url != "https://example.com/qr" {
		t.Errorf("url = %q, want %q", url, "https://example.com/qr")
	}

	if authCode != "abc123" {
		t.Errorf("authCode = %q, want %q", authCode, "abc123")
	}
}

func TestTVLogin_getAuthCode_EmptyResponse(t *testing.T) {
	mockBody := `{"data":{"url":"","auth_code":""}}`
	client := &mockTVHTTPClient{
		responses: []struct {
			resp *http.Response
			err  error
		}{{resp: newMockTVResponse(mockBody)}},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tv := NewTVLogin(client, logger, t.TempDir())

	_, _, err := tv.getAuthCode(context.Background())
	if err == nil {
		t.Fatal("expected error for empty response, got nil")
	}
}

func TestTVLogin_checkLoginStatus(t *testing.T) {
	tests := []struct {
		name      string
		response  string
		wantToken string
		wantDone  bool
		wantErr   error
	}{
		{
			name:      "waiting",
			response:  `{"code":86039,"data":{}}`,
			wantToken: "",
			wantDone:  false,
			wantErr:   nil,
		},
		{
			name:      "expired",
			response:  `{"code":86038,"data":{}}`,
			wantToken: "",
			wantDone:  false,
			wantErr:   entity.ErrQRExpired,
		},
		{
			name:      "success",
			response:  `{"code":0,"data":{"access_token":"token123"}}`,
			wantToken: "token123",
			wantDone:  true,
			wantErr:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockTVHTTPClient{
				responses: []struct {
					resp *http.Response
					err  error
				}{{resp: newMockTVResponse(tt.response)}},
			}

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			tv := NewTVLogin(client, logger, t.TempDir())

			token, done, err := tv.checkLoginStatus(context.Background(), "auth123")

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("err = %v, want %v", err, tt.wantErr)
			}
			if token != tt.wantToken {
				t.Errorf("token = %q, want %q", token, tt.wantToken)
			}
			if done != tt.wantDone {
				t.Errorf("done = %v, want %v", done, tt.wantDone)
			}
		})
	}
}

func TestTVLogin_pollLoginStatus(t *testing.T) {
	client := &mockTVHTTPClient{
		responses: []struct {
			resp *http.Response
			err  error
		}{
			{resp: newMockTVResponse(`{"code":86039,"data":{}}`)},
			{resp: newMockTVResponse(`{"code":0,"data":{"access_token":"mytoken"}}`)},
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tv := NewTVLogin(client, logger, t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	token, err := tv.pollLoginStatus(ctx, "auth123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token != "mytoken" {
		t.Errorf("token = %q, want %q", token, "mytoken")
	}
}

func TestTVLogin_pollLoginStatus_Expired(t *testing.T) {
	client := &mockTVHTTPClient{
		responses: []struct {
			resp *http.Response
			err  error
		}{
			{resp: newMockTVResponse(`{"code":86038,"data":{}}`)},
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tv := NewTVLogin(client, logger, t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := tv.pollLoginStatus(ctx, "auth123")
	if !errors.Is(err, entity.ErrQRExpired) {
		t.Errorf("err = %v, want ErrTVQRExpired", err)
	}
}

func TestTVLogin_pollLoginStatus_ContextCancelled(t *testing.T) {
	client := &mockTVHTTPClient{
		responses: []struct {
			resp *http.Response
			err  error
		}{
			{resp: newMockTVResponse(`{"code":86039,"data":{}}`)},
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tv := NewTVLogin(client, logger, t.TempDir())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := tv.pollLoginStatus(ctx, "auth123")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestTVLogin_saveToken(t *testing.T) {
	tmpDir := t.TempDir()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tv := NewTVLogin(nil, logger, tmpDir)

	err := tv.saveToken("testtoken")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, tvTokenFile))
	if err != nil {
		t.Fatalf("failed to read token file: %v", err)
	}

	want := "access_token=testtoken"
	if string(content) != want {
		t.Errorf("content = %q, want %q", string(content), want)
	}
}

func TestMaskAccessToken(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"abcdef", "ab***ef"},
		{"abc", "***"},
		{"", ""},
		{"ab", "***"},
		{"a", "***"},
	}

	for _, tt := range tests {
		got := maskAccessToken(tt.input)
		if got != tt.want {
			t.Errorf("maskAccessToken(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
