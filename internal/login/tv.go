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
	"strconv"
	"time"

	"github.com/sailist/BBDown-go/internal/core/entity"
	"github.com/sailist/BBDown-go/internal/core/util"
	"github.com/sailist/BBDown-go/pkg/httpclient"
	"github.com/skip2/go-qrcode"
)

const (
	tvAuthCodeURL = "https://passport.snm0516.aisee.tv/x/passport-tv-login/qrcode/auth_code"
	tvPollURL     = "https://passport.bilibili.com/x/passport-tv-login/qrcode/poll"
	tvQRCodeFile  = "qrcode.png"
	tvTokenFile   = "BBDownTV.data"
)



// tvAuthCodeResponse represents the response from the TV auth code API.
type tvAuthCodeResponse struct {
	Data struct {
		URL      string `json:"url"`
		AuthCode string `json:"auth_code"`
	} `json:"data"`
}

// tvPollResponse represents the response from the TV QR code polling API.
type tvPollResponse struct {
	Code int `json:"code"`
	Data struct {
		AccessToken string `json:"access_token"`
	} `json:"data"`
}

// TVLogin performs TV QR code login.
type TVLogin struct {
	client httpclient.Client
	logger *slog.Logger
	appDir string
}

// NewTVLogin creates a new TVLogin instance.
// appDir is the directory where credential files (BBDownTV.data) are saved.
func NewTVLogin(client httpclient.Client, logger *slog.Logger, appDir string) *TVLogin {
	return &TVLogin{client: client, logger: logger, appDir: appDir}
}

// Login performs the TV QR code login flow.
// It gets an auth code, generates a QR code, polls for login status,
// and saves the access token on success.
func (l *TVLogin) Login(ctx context.Context) error {
	authURL, authCode, err := l.getAuthCode(ctx)
	if err != nil {
		return fmt.Errorf("failed to get auth code: %w", err)
	}

	l.logger.Info("generated QR code", "file", tvQRCodeFile)

	if err := l.renderQRCode(authURL); err != nil {
		l.logger.Warn("failed to render console QR code", "err", err)
	}

	if err := l.saveQRCodeImage(authURL); err != nil {
		return fmt.Errorf("failed to save QR code image: %w", err)
	}

	defer l.cleanupQRCode()

	token, err := l.pollLoginStatus(ctx, authCode)
	if err != nil {
		return fmt.Errorf("poll login status: %w", err)
	}

	if err := l.saveToken(token); err != nil {
		return fmt.Errorf("failed to save access token: %w", err)
	}

	l.logger.Info("login successful", "file", tvTokenFile)
	return nil
}

func (l *TVLogin) getAuthCode(ctx context.Context) (string, string, error) {
	parms := GetTVLoginParms()
	body := []byte(parms.Encode())

	resp, err := l.client.Post(ctx, tvAuthCodeURL, body, httpclient.WithHeader("Content-Type", "application/x-www-form-urlencoded"))
	if err != nil {
		return "", "", fmt.Errorf("failed to request auth code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response body: %w", err)
	}

	var result tvAuthCodeResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if result.Data.URL == "" || result.Data.AuthCode == "" {
		return "", "", errors.New("empty URL or auth_code in response")
	}

	return result.Data.URL, result.Data.AuthCode, nil
}

func (l *TVLogin) renderQRCode(authURL string) error {
	qr, err := NewConsoleQRCode(authURL)
	if err != nil {
		return fmt.Errorf("render qr code: %w", err)
	}
	qr.Render()
	return nil
}

func (l *TVLogin) saveQRCodeImage(authURL string) error {
	if err := qrcode.WriteFile(authURL, qrcode.Medium, 5, tvQRCodeFile); err != nil {
		return fmt.Errorf("failed to write QR code file: %w", err)
	}
	return nil
}

func (l *TVLogin) cleanupQRCode() {
	if err := os.Remove(tvQRCodeFile); err != nil && !os.IsNotExist(err) {
		l.logger.Warn("failed to remove QR code file", "file", tvQRCodeFile, "err", err)
	}
}

func (l *TVLogin) pollLoginStatus(ctx context.Context, authCode string) (string, error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
		}

		token, done, err := l.checkLoginStatus(ctx, authCode)
		if err != nil {
			if errors.Is(err, entity.ErrQRExpired) {
				return "", fmt.Errorf("poll login status: %w", err)
			}
			l.logger.Warn("poll login status failed", "err", err)
			continue
		}
		if done {
			return token, nil
		}
	}
}

func (l *TVLogin) checkLoginStatus(ctx context.Context, authCode string) (string, bool, error) {
	parms := GetTVLoginParms()
	parms.Set("auth_code", authCode)
	parms.Set("ts", util.GetTimestamp(true))
	parms.Del("sign")
	parms.Set("sign", util.GetSign(parms.Encode(), false))

	body := []byte(parms.Encode())
	resp, err := l.client.Post(ctx, tvPollURL, body, httpclient.WithHeader("Content-Type", "application/x-www-form-urlencoded"))
	if err != nil {
		return "", false, fmt.Errorf("failed to poll login status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, fmt.Errorf("failed to read response body: %w", err)
	}

	var result tvPollResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", false, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	switch strconv.Itoa(result.Code) {
	case "86038":
		l.logger.Info("QR code expired")
		return "", false, entity.ErrQRExpired
	case "86039":
		l.logger.Debug("waiting for scan")
		return "", false, nil
	case "0":
		l.logger.Info("login successful", "access_token", maskAccessToken(result.Data.AccessToken))
		return result.Data.AccessToken, true, nil
	default:
		return "", false, fmt.Errorf("unknown poll code: %d", result.Code)
	}
}

func (l *TVLogin) saveToken(token string) error {
	path := filepath.Join(l.appDir, tvTokenFile)
	content := "access_token=" + token
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}
	return nil
}

func maskAccessToken(token string) string {
	if len(token) > 4 {
		return token[:2] + "***" + token[len(token)-2:]
	}
	if len(token) > 0 {
		return "***"
	}
	return ""
}

// GetTVLoginParms returns the TV login parameters as url.Values.
func GetTVLoginParms() url.Values {
	deviceID := util.GetRandomString(20)
	buvid := util.GetRandomString(37)
	now := time.Now()
	fingerprint := now.Format("20060102150405") + fmt.Sprintf("%03d", now.Nanosecond()/1e6) + util.GetRandomString(45)

	parms := url.Values{}
	parms.Set("appkey", "4409e2ce8ffd12b8")
	parms.Set("auth_code", "")
	parms.Set("bili_local_id", deviceID)
	parms.Set("build", "102801")
	parms.Set("buvid", buvid)
	parms.Set("channel", "master")
	parms.Set("device", "OnePlus")
	parms.Set("device_id", deviceID)
	parms.Set("device_name", "OnePlus7TPro")
	parms.Set("device_platform", "Android10OnePlusHD1910")
	parms.Set("fingerprint", fingerprint)
	parms.Set("guid", buvid)
	parms.Set("local_fingerprint", fingerprint)
	parms.Set("local_id", buvid)
	parms.Set("mobi_app", "android_tv_yst")
	parms.Set("networkstate", "wifi")
	parms.Set("platform", "android")
	parms.Set("sys_ver", "29")
	parms.Set("ts", util.GetTimestamp(true))

	// Calculate sign
	parms.Set("sign", util.GetSign(parms.Encode(), false))

	return parms
}
