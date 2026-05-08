package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/sailist/BBDown-go/pkg/httpclient"
)

const (
	latestReleaseURL = "https://github.com/nilaoda/BBDown/releases/latest"
	releaseTagPrefix = "https://github.com/nilaoda/BBDown/releases/tag/"
)

// CheckUpdate checks the latest BBDown release version from GitHub.
// It returns the version tag string (e.g., "1.6.3") or an error.
func CheckUpdate(ctx context.Context, client httpclient.Client) (string, error) {
	slog.Debug("checking for update", slog.String("url", latestReleaseURL))

	redirectURL, err := client.GetRedirectLocation(ctx, latestReleaseURL)
	if err != nil {
		return "", fmt.Errorf("failed to get redirect location: %w", err)
	}

	slog.Debug("got redirect location", slog.String("url", redirectURL))

	if !strings.HasPrefix(redirectURL, releaseTagPrefix) {
		return "", fmt.Errorf("unexpected redirect URL: %s", redirectURL)
	}

	version := strings.TrimPrefix(redirectURL, releaseTagPrefix)
	return version, nil
}
