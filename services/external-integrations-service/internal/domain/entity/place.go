package entity

// Place is the canonical place shape returned by provider adapters and exposed
// by the HTTP API.
type Place struct {
	Provider        string   `json:"provider"`
	ProviderPlaceID string   `json:"providerPlaceId"`
	Name            string   `json:"name"`
	Address         string   `json:"address"`
	Latitude        *float64 `json:"latitude,omitempty"`
	Longitude       *float64 `json:"longitude,omitempty"`
	Rating          *float64 `json:"rating,omitempty"`
	RatingCount     *int     `json:"ratingCount,omitempty"`
	MapURL          string   `json:"mapUrl,omitempty"`
	Category        string   `json:"category,omitempty"`
	Website         string   `json:"website,omitempty"`
}
