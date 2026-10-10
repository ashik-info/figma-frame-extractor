// Package main
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/ashik-info/figma-frame-extractor/internal/config"
	"github.com/ashik-info/figma-frame-extractor/internal/handlers"
)

func main() {
	cfg := config.MustLoad()

	if err := handlers.ExtractFrame(http.DefaultClient, cfg.RawURL, cfg.FigmaToken, os.Stdout); err != nil {
		log.Fatal(err)
	}
}
