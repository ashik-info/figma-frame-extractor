// Package models
package models

type FigmaNode struct {
	ID                    string      `json:"id"`
	Name                  string      `json:"name"`
	Type                  string      `json:"type"`
	LayoutMode            string      `json:"layoutMode,omitempty"`
	PrimaryAxisAlignItems string      `json:"primaryAxisAlignItems,omitempty"`
	CounterAxisAlignItems string      `json:"counterAxisAlignItems,omitempty"`
	ItemSpacing           float64     `json:"itemSpacing,omitempty"`
	Children              []FigmaNode `json:"children,omitempty"`
}
