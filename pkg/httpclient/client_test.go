package httpclient

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"log/slog"
)

func TestRetrySuccessAfterFailures(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := NewStandardClient(logger)

	start := time.Now()
	resp, err := client.Get(context.Background(), ts.URL)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
	if elapsed < 2*time.Second {
		t.Fatalf("expected backoff delay, got %v", elapsed)
	}
}

func TestRetryExhaustion(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := NewStandardClient(logger)

	resp, err := client.Get(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestUARandomness(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	uas := make(map[string]int)
	for i := 0; i < 20; i++ {
		client := NewStandardClient(logger)
		ua := client.userAgent
		uas[ua]++
	}

	if len(uas) < 2 {
		t.Fatalf("expected different UAs, got %d unique", len(uas))
	}
}

func TestGzipDecompression(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		zw.Write([]byte("hello world"))
		zw.Close()
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(buf.Bytes())
	}))
	defer ts.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := NewStandardClient(logger)

	resp, err := client.Get(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unexpected error reading body: %v", err)
	}
	if string(body) != "hello world" {
		t.Fatalf("expected 'hello world', got %q", string(body))
	}
}

func TestRedirectLocation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := NewStandardClient(logger)

	location, err := client.GetRedirectLocation(context.Background(), ts.URL+"/redirect")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := ts.URL + "/final"
	if location != expected {
		t.Fatalf("expected %q, got %q", expected, location)
	}
}

func TestContentLength(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "42")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := NewStandardClient(logger)

	length, err := client.GetContentLength(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if length != 42 {
		t.Fatalf("expected 42, got %d", length)
	}
}

func TestCustomHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") == "value" && r.Header.Get("Cookie") == "mycookie" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := NewStandardClient(logger)

	resp, err := client.Get(context.Background(), ts.URL, WithHeader("X-Custom", "value"), WithCookie("mycookie"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestBilibiliReferer(t *testing.T) {
	var receivedReferer string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReferer = r.Header.Get("Referer")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := NewStandardClient(logger)

	_, err := client.Get(context.Background(), ts.URL+"/api.bilibili.com/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedReferer != "https://www.bilibili.com/" {
		t.Fatalf("expected bilibili referer, got %q", receivedReferer)
	}
}

func TestEpSsCookie(t *testing.T) {
	var receivedCookie string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedCookie = r.Header.Get("Cookie")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := NewStandardClient(logger)

	_, err := client.Get(context.Background(), ts.URL+"/ep/123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(receivedCookie, "CURRENT_FNVAL=4048") {
		t.Fatalf("expected CURRENT_FNVAL=4048 in cookie, got %q", receivedCookie)
	}
}

func TestTransportConfiguration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := NewStandardClient(logger)

	transport, ok := client.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", client.client.Transport)
	}

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"MaxIdleConns", transport.MaxIdleConns, 100},
		{"MaxIdleConnsPerHost", transport.MaxIdleConnsPerHost, 10},
		{"IdleConnTimeout", transport.IdleConnTimeout, 90 * time.Second},
		{"TLSHandshakeTimeout", transport.TLSHandshakeTimeout, 10 * time.Second},
		{"ExpectContinueTimeout", transport.ExpectContinueTimeout, 1 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.got, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, tt.got)
			}
		})
	}
}
