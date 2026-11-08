// Package main
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/ashik-info/figma-frame-extractor/internal/config"
	"github.com/ashik-info/figma-frame-extractor/internal/handlers"
)

func main() {
	cfg := config.MustLoad()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := handlers.ExtractFrameContext(ctx, http.DefaultClient, cfg.RawURL, cfg.FigmaToken, os.Stdout); err != nil {
		log.Fatal(err)
	}
}
