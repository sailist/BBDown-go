package main

import (
	"context"
	"testing"

	"github.com/sailist/BBDown-go/internal/cli"
)

func TestPanicRecovery(t *testing.T) {
	recovered := false
	fn := func() {
		defer func() {
			if r := recover(); r != nil {
				recovered = true
			}
		}()
		panic("intentional test panic")
	}
	fn()
	if !recovered {
		t.Fatal("expected panic to be recovered")
	}
}

func TestRunApp_ReturnsError_WhenSetupWorkFails(t *testing.T) {
	opt := cli.NewOption()
	opt.FFmpegPath = "/nonexistent/path/to/ffmpeg"
	opt.SkipMux = false

	err := runApp(context.Background(), opt)
	if err == nil {
		t.Fatal("expected error when SetupWork fails, got nil")
	}
}

func TestDoWork_ReturnsError_WhenInputEmpty(t *testing.T) {
	opt := cli.NewOption()
	opt.URL = ""

	err := doWork(context.Background(), opt)
	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
}
