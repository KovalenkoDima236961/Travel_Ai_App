package handler

import "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"

// SearchPlacesResponse is the response envelope for GET /places/search.
type SearchPlacesResponse struct {
	Items []entity.Place `json:"items"`
}
