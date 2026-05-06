package login

import (
	"fmt"

	"github.com/skip2/go-qrcode"
)

// ConsoleQRCode renders a QR code to the console using block characters.
type ConsoleQRCode struct {
	data *qrcode.QRCode
}

// NewConsoleQRCode creates a new console QR code renderer from text.
func NewConsoleQRCode(text string) (*ConsoleQRCode, error) {
	data, err := qrcode.New(text, qrcode.Medium)
	if err != nil {
		return nil, fmt.Errorf("failed to create QR code: %w", err)
	}
	return &ConsoleQRCode{data: data}, nil
}

// Render prints the QR code to stdout using Unicode block characters.
func (c *ConsoleQRCode) Render() {
	c.RenderWithSymbols("██", "  ")
}

// RenderWithSymbols prints the QR code with custom dark and light symbols.
func (c *ConsoleQRCode) RenderWithSymbols(dark, light string) {
	bitmap := c.data.Bitmap()
	for y := range bitmap {
		for x := range bitmap[y] {
			if bitmap[y][x] {
				print(dark)
			} else {
				print(light)
			}
		}
		println()
	}
}

// SaveQRCodeImage saves the QR code as a PNG image file.
func SaveQRCodeImage(text, path string, size int) error {
	if err := qrcode.WriteFile(text, qrcode.Medium, size, path); err != nil {
		return fmt.Errorf("failed to save QR code image: %w", err)
	}
	return nil
}
