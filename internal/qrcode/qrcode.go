// Package qrcode generates and caches the QR code PNG for a poster.
package qrcode

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/skip2/go-qrcode"
)

// PNGForPoster returns the PNG bytes encoding baseURL+"/p/"+posterID,
// generating it on first request and caching it to cacheDir thereafter.
// Posters are created rarely and generation is cheap, so on-demand-with-
// cache avoids a batch job for what's effectively a handful of images.
func PNGForPoster(cacheDir string, posterID int64, baseURL string) ([]byte, error) {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("create qr cache dir: %w", err)
	}

	path := filepath.Join(cacheDir, fmt.Sprintf("%d.png", posterID))
	if b, err := os.ReadFile(path); err == nil {
		return b, nil
	}

	url := fmt.Sprintf("%s/p/%d", baseURL, posterID)
	png, err := qrcode.Encode(url, qrcode.Medium, 512)
	if err != nil {
		return nil, fmt.Errorf("encode qr code: %w", err)
	}

	if err := os.WriteFile(path, png, 0o644); err != nil {
		return nil, fmt.Errorf("cache qr code: %w", err)
	}
	return png, nil
}
