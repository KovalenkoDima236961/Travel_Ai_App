// Package application defines the use-case layer's ports (interfaces) that
// adapters implement. Concrete use cases live in the service subpackage.
package application

import (
	"context"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
)

// GenerateItineraryInput is the internal generator request. Trip Service owns
// loading trusted user context; frontend callers cannot submit these fields.
type GenerateItineraryInput struct {
	Trip            entity.Trip
	UserProfile     *usercontext.UserProfile
	UserPreferences *usercontext.UserPreferences
}

// ItineraryGenerator is the port for turning a trip into a concrete itinerary.
// Implementations (adapters) live under infrastructure.
type ItineraryGenerator interface {
	Generate(ctx context.Context, input GenerateItineraryInput) (*aggregate.Itinerary, error)
}
