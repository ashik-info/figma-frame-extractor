// Package models
package models

type FigmaResponse struct {
	Nodes map[string]struct {
		Document FigmaNode `json:"document"`
	} `json:"nodes"`
}
