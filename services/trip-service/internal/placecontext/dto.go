package placecontext

import "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"

// SearchPlacesResponse mirrors External Integrations Service /places/search.
type SearchPlacesResponse struct {
	Items []aggregate.PlaceRef `json:"items"`
}
