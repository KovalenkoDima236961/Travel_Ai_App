package entity

// OpeningHoursInterval is one local-time opening interval for a place. DayOfWeek
// uses the app-wide convention: 1 = Monday, 7 = Sunday.
type OpeningHoursInterval struct {
	DayOfWeek int    `json:"dayOfWeek"`
	Open      string `json:"open"`
	Close     string `json:"close"`
}

// Place is the canonical place shape returned by provider adapters and exposed
// by the HTTP API.
type Place struct {
	Provider        string                 `json:"provider"`
	ProviderPlaceID string                 `json:"providerPlaceId"`
	Name            string                 `json:"name"`
	Address         string                 `json:"address"`
	Latitude        *float64               `json:"latitude,omitempty"`
	Longitude       *float64               `json:"longitude,omitempty"`
	Rating          *float64               `json:"rating,omitempty"`
	RatingCount     *int                   `json:"ratingCount,omitempty"`
	MapURL          string                 `json:"mapUrl,omitempty"`
	Category        string                 `json:"category,omitempty"`
	Website         string                 `json:"website,omitempty"`
	OpeningHours    []OpeningHoursInterval `json:"openingHours,omitempty"`
}
