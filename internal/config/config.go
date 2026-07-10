// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"os"
)

type Config struct {
	Addr          string // e.g. ":8080"
	DBPath        string
	BaseURL       string // public base URL used to build QR-encoded links, e.g. https://survey.example.com
	QRCacheDir    string
	AdminUser     string
	AdminPassHash string // bcrypt hash
	HashSecret    string // HMAC key for the non-guessable direct-entry (AMOE) URLs
}

func Load() (Config, error) {
	c := Config{
		Addr:          getEnv("ADDR", ":8080"),
		DBPath:        getEnv("DB_PATH", "data/qrsurvey.db"),
		BaseURL:       getEnv("BASE_URL", "http://localhost:8080"),
		QRCacheDir:    getEnv("QR_CACHE_DIR", "data/qrcodes"),
		AdminUser:     os.Getenv("ADMIN_USER"),
		AdminPassHash: os.Getenv("ADMIN_PASS_HASH"),
		HashSecret:    os.Getenv("HASH_SECRET"),
	}
	if c.AdminUser == "" || c.AdminPassHash == "" {
		return c, fmt.Errorf("ADMIN_USER and ADMIN_PASS_HASH must be set")
	}
	if c.HashSecret == "" {
		return c, fmt.Errorf("HASH_SECRET must be set")
	}
	return c, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
