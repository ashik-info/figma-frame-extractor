// Package config
package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName    string
	Env        string
	RawURL     string
	FigmaToken string
}

func MustLoad() Config {
	_ = godotenv.Load()

	cfg := Config{
		AppName:    env("APP_NAME", "figma-frame-extractor-service"),
		Env:        env("ENV", "dev"),
		RawURL:     env("RAW_URL", ""),
		FigmaToken: env("FIGMA_TOKEN", ""),
	}
	return cfg
}

func env(k, d string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return d
}
