package muxer

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFindExecutable_CurrentDir(t *testing.T) {
	dir := t.TempDir()
	name := "testbin_currentdir"
	if runtime.GOOS == "windows" {
		name = "testbin_currentdir.exe"
	}
	binPath := filepath.Join(dir, name)
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\necho ok\n"), 0o755); err != nil {
		t.Fatalf("create test binary: %v", err)
	}

	// Change to the temp directory
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	found, err := FindExecutable("testbin_currentdir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(found) != name {
		t.Errorf("expected %q, got %q", name, found)
	}
}

func TestFindExecutable_PATH(t *testing.T) {
	dir := t.TempDir()
	name := "testbin_path"
	if runtime.GOOS == "windows" {
		name = "testbin_path.exe"
	}
	binPath := filepath.Join(dir, name)
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\necho ok\n"), 0o755); err != nil {
		t.Fatalf("create test binary: %v", err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", dir+string(filepath.ListSeparator)+origPath)
	defer os.Setenv("PATH", origPath)

	found, err := FindExecutable("testbin_path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != binPath {
		t.Errorf("expected %q, got %q", binPath, found)
	}
}

func TestFindExecutable_NotFound(t *testing.T) {
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", origPath)

	_, err := FindExecutable("nonexistent_binary_12345")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got: %v", err)
	}
}

func TestCheckFFmpegDOVI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows: mock scripts are not valid PE executables")
	}
	dir := t.TempDir()
	name := "ffmpeg"
	if runtime.GOOS == "windows" {
		name = "ffmpeg.exe"
	}
	binPath := filepath.Join(dir, name)

	// Create a mock ffmpeg that prints a known libavutil version
	script := `#!/bin/sh
echo "ffmpeg version 5.1"
echo "built with ..."
echo "libavutil      57. 28.100"
echo "libavcodec     59. 37.100"
`
	if runtime.GOOS == "windows" {
		script = `@echo off
echo ffmpeg version 5.1
echo libavutil      57. 28.100
`
	}
	if err := os.WriteFile(binPath, []byte(script), 0o755); err != nil {
		t.Fatalf("create mock ffmpeg: %v", err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", dir+string(filepath.ListSeparator)+origPath)
	defer os.Setenv("PATH", origPath)

	ok, err := CheckFFmpegDOVI()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected DOVI support for libavutil 57.28")
	}
}

func TestCheckFFmpegDOVI_OldVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows: mock scripts are not valid PE executables")
	}
	dir := t.TempDir()
	name := "ffmpeg"
	if runtime.GOOS == "windows" {
		name = "ffmpeg.exe"
	}
	binPath := filepath.Join(dir, name)

	script := `#!/bin/sh
echo "ffmpeg version 4.4"
echo "libavutil      56. 70.100"
`
	if runtime.GOOS == "windows" {
		script = `@echo off
echo ffmpeg version 4.4
echo libavutil      56. 70.100
`
	}
	if err := os.WriteFile(binPath, []byte(script), 0o755); err != nil {
		t.Fatalf("create mock ffmpeg: %v", err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", dir+string(filepath.ListSeparator)+origPath)
	defer os.Setenv("PATH", origPath)

	ok, err := CheckFFmpegDOVI()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected no DOVI support for libavutil 56.70")
	}
}

func TestCheckFFmpegDOVI_BoundaryVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows: mock scripts are not valid PE executables")
	}
	dir := t.TempDir()
	name := "ffmpeg"
	if runtime.GOOS == "windows" {
		name = "ffmpeg.exe"
	}
	binPath := filepath.Join(dir, name)

	script := `#!/bin/sh
echo "ffmpeg version 5.0"
echo "libavutil      57. 17.100"
`
	if runtime.GOOS == "windows" {
		script = `@echo off
echo ffmpeg version 5.0
echo libavutil      57. 17.100
`
	}
	if err := os.WriteFile(binPath, []byte(script), 0o755); err != nil {
		t.Fatalf("create mock ffmpeg: %v", err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", dir+string(filepath.ListSeparator)+origPath)
	defer os.Setenv("PATH", origPath)

	ok, err := CheckFFmpegDOVI()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected DOVI support for libavutil 57.17 (boundary)")
	}
}

func TestCheckFFmpegDOVI_NotFound(t *testing.T) {
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", origPath)

	_, err := CheckFFmpegDOVI()
	if err == nil || !strings.Contains(err.Error(), "find ffmpeg") {
		t.Fatalf("expected find ffmpeg error, got: %v", err)
	}
}

func TestCheckFFmpegDOVI_NoLibavutil(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows: mock scripts are not valid PE executables")
	}
	dir := t.TempDir()
	name := "ffmpeg"
	if runtime.GOOS == "windows" {
		name = "ffmpeg.exe"
	}
	binPath := filepath.Join(dir, name)

	script := `#!/bin/sh
echo "ffmpeg version 5.1"
echo "no libavutil here"
`
	if runtime.GOOS == "windows" {
		script = `@echo off
echo ffmpeg version 5.1
echo no libavutil here
`
	}
	if err := os.WriteFile(binPath, []byte(script), 0o755); err != nil {
		t.Fatalf("create mock ffmpeg: %v", err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", dir+string(filepath.ListSeparator)+origPath)
	defer os.Setenv("PATH", origPath)

	ok, err := CheckFFmpegDOVI()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected no DOVI support when libavutil not found")
	}
}
