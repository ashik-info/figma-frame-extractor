// Package handlers
package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ashik-info/figma-frame-extractor/internal/models"
	"github.com/ashik-info/figma-frame-extractor/internal/utils"
)

func ExtractFrame(client *http.Client, rawURL, token string, output io.Writer) error {
	target, err := utils.ExtractTarget(rawURL)
	if err != nil {
		return fmt.Errorf("extract Figma parameters: %w", err)
	}

	node, err := fetchNode(client, target, token)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(node); err != nil {
		return fmt.Errorf("write Figma node JSON: %w", err)
	}
	return nil
}

func fetchNode(client *http.Client, target models.Target, token string) (models.FigmaNode, error) {
	url := fmt.Sprintf("https://api.figma.com/v1/files/%s/nodes?ids=%s", target.FileKey, target.NodeID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return models.FigmaNode{}, fmt.Errorf("create Figma request: %w", err)
	}
	req.Header.Set("X-Figma-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return models.FigmaNode{}, fmt.Errorf("request Figma node: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.FigmaNode{}, fmt.Errorf("Figma request failed with HTTP status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.FigmaNode{}, fmt.Errorf("read Figma response: %w", err)
	}

	var data models.FigmaResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return models.FigmaNode{}, fmt.Errorf("parse Figma response JSON: %w", err)
	}
	node, ok := data.Nodes[target.NodeID]
	if !ok || node.Document.ID == "" {
		return models.FigmaNode{}, fmt.Errorf("selected Figma node was not found")
	}
	return node.Document, nil
}
