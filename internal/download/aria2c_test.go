package download

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nilaonai/bbdown-go/internal/config"
)

func TestDownloadWithAria2c_NotFound(t *testing.T) {
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", origPath)

	err := DownloadWithAria2c(context.Background(), "http://example.com/file", filepath.Join(t.TempDir(), "out.mp4"), "", config.NewConfig())
	if err == nil || !strings.Contains(err.Error(), "find aria2c") {
		t.Fatalf("expected aria2c not found, got %v", err)
	}
}

func TestDownloadWithAria2c_CommandArgs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.mp4")

	// Create a dummy aria2c executable so findAria2c succeeds.
	dummyAria2c := filepath.Join(dir, "aria2c")
	if err := os.WriteFile(dummyAria2c, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("create dummy aria2c: %v", err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", dir+string(filepath.ListSeparator)+origPath)
	defer os.Setenv("PATH", origPath)

	var capturedArgs []string
	origExec := execCommandContext
	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		capturedArgs = append([]string{name}, arg...)
		return origExec(ctx, "echo", append([]string{"mock"}, arg...)...)
	}
	defer func() { execCommandContext = origExec }()

	err := DownloadWithAria2c(context.Background(), "http://example.com/file", path, "--max-download-limit=1M", &config.Config{Cookie: "testcookie"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedArgs) == 0 {
		t.Fatal("expected captured args")
	}

	argsStr := strings.Join(capturedArgs, " ")
	if !strings.Contains(argsStr, "http://example.com/file") {
		t.Fatalf("expected url in args, got %s", argsStr)
	}
	if !strings.Contains(argsStr, "Cookie: testcookie") {
		t.Fatalf("expected cookie in args, got %s", argsStr)
	}
	if !strings.Contains(argsStr, "--max-download-limit=1M") {
		t.Fatalf("expected extra args, got %s", argsStr)
	}
	if !strings.Contains(argsStr, "-d "+dir) && !strings.Contains(argsStr, "-d "+filepath.Clean(dir)) {
		t.Fatalf("expected dir in args, got %s", argsStr)
	}
	if !strings.Contains(argsStr, "-o out.mp4") {
		t.Fatalf("expected filename in args, got %s", argsStr)
	}
}

func TestDownloadWithAria2c_AndroidURL_NoReferer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.mp4")

	dummyAria2c := filepath.Join(dir, "aria2c")
	if err := os.WriteFile(dummyAria2c, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("create dummy aria2c: %v", err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", dir+string(filepath.ListSeparator)+origPath)
	defer os.Setenv("PATH", origPath)

	var capturedArgs []string
	origExec := execCommandContext
	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		capturedArgs = append([]string{name}, arg...)
		return origExec(ctx, "echo", append([]string{"mock"}, arg...)...)
	}
	defer func() { execCommandContext = origExec }()

	err := DownloadWithAria2c(context.Background(), "http://example.com/file?platform=android", path, "", config.NewConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	argsStr := strings.Join(capturedArgs, " ")
	if strings.Contains(argsStr, "Referer:") {
		t.Fatalf("expected no referer for android url, got %s", argsStr)
	}
}
