package priceclient

import "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"

type PriceEstimateInput struct {
	Destination string            `json:"destination"`
	Currency    string            `json:"currency,omitempty"`
	Date        string            `json:"date,omitempty"`
	Place       PricePlace        `json:"place"`
	ItemContext *PriceItemContext `json:"itemContext,omitempty"`
}

type PricePlace struct {
	Provider        string   `json:"provider,omitempty"`
	ProviderPlaceID string   `json:"providerPlaceId,omitempty"`
	Name            string   `json:"name"`
	Address         string   `json:"address,omitempty"`
	Category        string   `json:"category,omitempty"`
	Latitude        *float64 `json:"lat,omitempty"`
	Longitude       *float64 `json:"lng,omitempty"`
	Rating          *float64 `json:"rating,omitempty"`
	PriceLevel      *int     `json:"priceLevel,omitempty"`
}

type PriceItemContext struct {
	Name        string `json:"name,omitempty"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

type PriceEstimateResult struct {
	EstimatedCost   *aggregate.EstimatedCost `json:"estimatedCost"`
	Provider        string                   `json:"provider"`
	FallbackUsed    bool                     `json:"fallbackUsed"`
	PriceType       *string                  `json:"priceType"`
	Matched         bool                     `json:"matched"`
	MatchConfidence float64                  `json:"matchConfidence"`
	Metadata        map[string]any           `json:"metadata,omitempty"`
}
