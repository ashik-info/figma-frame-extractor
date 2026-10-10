// Package utils
package utils

import (
	"errors"
	"net/url"
	"strings"

	"github.com/ashik-info/figma-frame-extractor/internal/models"
)

func ExtractTarget(rawURL string) (models.Target, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return models.Target{}, err
	}

	if u.Host != "www.figma.com" && u.Host != "figma.com" {
		return models.Target{}, errors.New("invalid host: not a figma URL")
	}

	pathSegments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(pathSegments) < 2 {
		return models.Target{}, errors.New("invalid path: missing file key")
	}

	if pathSegments[0] != "design" && pathSegments[0] != "file" {
		return models.Target{}, errors.New("invalid path: missing design or file route")
	}

	fileKey := pathSegments[1]

	rawNodeID := u.Query().Get("node-id")
	if rawNodeID == "" {
		return models.Target{}, errors.New("missing node-id query parameter")
	}

	apiNodeID := strings.ReplaceAll(rawNodeID, "-", ":")

	return models.Target{
		FileKey: fileKey,
		NodeID:  apiNodeID,
	}, nil
}
