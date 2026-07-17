package search

import (
	"time"

	"github.com/google/uuid"
)

type Result struct {
	ID            string         `json:"id"`
	Type          ResultType     `json:"type"`
	Title         string         `json:"title"`
	Description   string         `json:"description,omitempty"`
	Context       string         `json:"context,omitempty"`
	WorkspaceName string         `json:"workspaceName,omitempty"`
	Href          string         `json:"href"`
	Icon          string         `json:"icon"`
	Category      string         `json:"category"`
	Score         float64        `json:"score"`
	Metadata      map[string]any `json:"metadata,omitempty"`

	TripID      *uuid.UUID `json:"-"`
	WorkspaceID *uuid.UUID `json:"-"`
	UpdatedAt   time.Time  `json:"-"`
}

type Group struct {
	Title string   `json:"title"`
	Items []Result `json:"items"`
}

type Response struct {
	Query   string   `json:"query"`
	Items   []Result `json:"items"`
	Groups  []Group  `json:"groups"`
	HasMore bool     `json:"hasMore"`
}
