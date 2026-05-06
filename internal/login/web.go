package login

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nilaonai/bbdown-go/internal/core/entity"
	"github.com/nilaonai/bbdown-go/pkg/httpclient"
	"github.com/skip2/go-qrcode"
)

const (
	qrGenerateURL = "https://passport.bilibili.com/x/passport-login/web/qrcode/generate?source=main-fe-header"
	qrPollURLFmt  = "https://passport.bilibili.com/x/passport-login/web/qrcode/poll?qrcode_key=%s&source=main-fe-header"
	qrCodeFile    = "qrcode.png"
	cookieFile    = "BBDown.data"
)



// qrGenerateResponse represents the response from the QR code generation API.
type qrGenerateResponse struct {
	Data struct {
		URL       string `json:"url"`
		QrcodeKey string `json:"qrcode_key"`
	} `json:"data"`
}

// qrPollResponse represents the response from the QR code polling API.
type qrPollResponse struct {
	Data struct {
		Code int    `json:"code"`
		URL  string `json:"url"`
	} `json:"data"`
}

// WebLogin performs WEB QR code login.
type WebLogin struct {
	client httpclient.Client
	logger *slog.Logger
}

// NewWebLogin creates a new WebLogin instance.
func NewWebLogin(client httpclient.Client, logger *slog.Logger) *WebLogin {
	return &WebLogin{client: client, logger: logger}
}

// Login performs the WEB QR code login flow.
// It generates a QR code, polls for login status, and saves the cookie on success.
func (l *WebLogin) Login(ctx context.Context) error {
	loginURL, qrcodeKey, err := l.generateQRCode(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate QR code: %w", err)
	}

	l.logger.Info("generated QR code", "file", qrCodeFile)

	if err := l.renderQRCode(loginURL); err != nil {
		l.logger.Warn("failed to render console QR code", "err", err)
	}

	if err := l.saveQRCodeImage(loginURL); err != nil {
		return fmt.Errorf("failed to save QR code image: %w", err)
	}

	defer l.cleanupQRCode()

	cookie, err := l.pollLoginStatus(ctx, qrcodeKey)
	if err != nil {
		return fmt.Errorf("poll login status: %w", err)
	}

	if err := l.saveCookie(cookie); err != nil {
		return fmt.Errorf("failed to save cookie: %w", err)
	}

	l.logger.Info("login successful", "file", cookieFile)
	return nil
}

func (l *WebLogin) generateQRCode(ctx context.Context) (string, string, error) {
	resp, err := l.client.Get(ctx, qrGenerateURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to get QR code URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response body: %w", err)
	}

	var result qrGenerateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if result.Data.URL == "" || result.Data.QrcodeKey == "" {
		return "", "", errors.New("empty URL or qrcode_key in response")
	}

	return result.Data.URL, result.Data.QrcodeKey, nil
}

func (l *WebLogin) renderQRCode(loginURL string) error {
	qr, err := NewConsoleQRCode(loginURL)
	if err != nil {
		return fmt.Errorf("render qr code: %w", err)
	}
	qr.Render()
	return nil
}

func (l *WebLogin) saveQRCodeImage(loginURL string) error {
	if err := qrcode.WriteFile(loginURL, qrcode.Medium, 7, qrCodeFile); err != nil {
		return fmt.Errorf("failed to write QR code file: %w", err)
	}
	return nil
}

func (l *WebLogin) cleanupQRCode() {
	if err := os.Remove(qrCodeFile); err != nil && !os.IsNotExist(err) {
		l.logger.Warn("failed to remove QR code file", "file", qrCodeFile, "err", err)
	}
}

func (l *WebLogin) pollLoginStatus(ctx context.Context, qrcodeKey string) (string, error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	scanned := false

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
		}

		cookie, done, err := l.checkLoginStatus(ctx, qrcodeKey, &scanned)
		if err != nil {
			if errors.Is(err, entity.ErrQRExpired) {
				return "", fmt.Errorf("poll login status: %w", err)
			}
			l.logger.Warn("poll login status failed", "err", err)
			continue
		}
		if done {
			return cookie, nil
		}
	}
}

func (l *WebLogin) checkLoginStatus(ctx context.Context, qrcodeKey string, scanned *bool) (string, bool, error) {
	pollURL := fmt.Sprintf(qrPollURLFmt, qrcodeKey)
	resp, err := l.client.Get(ctx, pollURL)
	if err != nil {
		return "", false, fmt.Errorf("failed to poll login status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, fmt.Errorf("failed to read response body: %w", err)
	}

	var result qrPollResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", false, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	switch result.Data.Code {
	case 86038:
		l.logger.Info("QR code expired")
		return "", false, entity.ErrQRExpired
	case 86101:
		l.logger.Debug("waiting for scan")
		return "", false, nil
	case 86090:
		if !*scanned {
			l.logger.Info("scanned, waiting for confirmation")
			*scanned = true
		}
		return "", false, nil
	case 0:
		cookie, err := l.extractCookie(result.Data.URL)
		if err != nil {
			return "", false, fmt.Errorf("failed to extract cookie: %w", err)
		}
		l.logger.Info("login successful", "SESSDATA", maskSESSDATA(cookie))
		return cookie, true, nil
	default:
		return "", false, fmt.Errorf("unknown poll code: %d", result.Data.Code)
	}
}

func (l *WebLogin) extractCookie(redirectURL string) (string, error) {
	u, err := url.Parse(redirectURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse redirect URL: %w", err)
	}

	query := u.RawQuery
	if query == "" {
		return "", errors.New("no query string in redirect URL")
	}

	cookie := strings.ReplaceAll(query, "&", ";")
	cookie = strings.ReplaceAll(cookie, ",", "%2C")
	return cookie, nil
}

func (l *WebLogin) saveCookie(cookie string) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	path := filepath.Join(wd, cookieFile)
	if err := os.WriteFile(path, []byte(cookie), 0o600); err != nil {
		return fmt.Errorf("failed to write cookie file: %w", err)
	}
	return nil
}

func maskSESSDATA(cookie string) string {
	// Try to extract SESSDATA value for logging (masked)
	parts := strings.Split(cookie, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "SESSDATA=") {
			val := strings.TrimPrefix(part, "SESSDATA=")
			if len(val) > 4 {
				return val[:2] + "***" + val[len(val)-2:]
			}
			return "***"
		}
	}
	return ""
}
