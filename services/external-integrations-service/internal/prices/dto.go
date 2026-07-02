package prices

// EstimatedCost mirrors the shared item-level cost shape used by Trip Service
// and the Web App.
type EstimatedCost struct {
	Amount     *float64 `json:"amount,omitempty"`
	Currency   string   `json:"currency,omitempty"`
	Category   string   `json:"category,omitempty"`
	Confidence string   `json:"confidence,omitempty"`
	Source     string   `json:"source,omitempty"`
	Note       string   `json:"note,omitempty"`
}

type PriceEstimateInput struct {
	Destination string            `json:"destination"`
	Currency    string            `json:"currency,omitempty"`
	Date        string            `json:"date,omitempty"`
	Place       *PricePlace       `json:"place"`
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
	EstimatedCost   *EstimatedCost `json:"estimatedCost"`
	Provider        string         `json:"provider"`
	FallbackUsed    bool           `json:"fallbackUsed"`
	PriceType       *string        `json:"priceType"`
	Matched         bool           `json:"matched"`
	MatchConfidence float64        `json:"matchConfidence"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}
