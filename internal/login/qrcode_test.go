package login

import (
	"os"
	"testing"
)

func TestNewConsoleQRCode_ValidText(t *testing.T) {
	qr, err := NewConsoleQRCode("https://example.com")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if qr == nil {
		t.Fatal("expected QR code, got nil")
	}
}

func TestNewConsoleQRCode_EmptyText(t *testing.T) {
	_, err := NewConsoleQRCode("")
	if err == nil {
		t.Fatal("expected error for empty text, got nil")
	}
}

func TestConsoleQRCode_Render(t *testing.T) {
	qr, err := NewConsoleQRCode("test")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Render panicked: %v", r)
		}
	}()

	qr.Render()
}

func TestConsoleQRCode_RenderWithSymbols(t *testing.T) {
	qr, err := NewConsoleQRCode("test")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RenderWithSymbols panicked: %v", r)
		}
	}()

	qr.RenderWithSymbols("##", "..")
}

func TestSaveQRCodeImage(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "qrcode-*.png")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	if err := SaveQRCodeImage("https://example.com", tmpPath, 256); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	info, err := os.Stat(tmpPath)
	if err != nil {
		t.Fatalf("expected file to exist, got error: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("expected non-empty file")
	}
}
