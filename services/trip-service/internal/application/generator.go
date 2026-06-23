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

// RegenerateDayInput is the internal generator request for replacing one day
// in an existing itinerary.
type RegenerateDayInput struct {
	Trip             entity.Trip
	CurrentItinerary aggregate.Itinerary
	DayNumber        int
	Instruction      string
	UserProfile      *usercontext.UserProfile
	UserPreferences  *usercontext.UserPreferences
}

// RegenerateItemInput is the internal generator request for replacing one item
// in an existing itinerary day. ItemIndex is zero-based.
type RegenerateItemInput struct {
	Trip             entity.Trip
	CurrentItinerary aggregate.Itinerary
	DayNumber        int
	ItemIndex        int
	Instruction      string
	UserProfile      *usercontext.UserProfile
	UserPreferences  *usercontext.UserPreferences
}

// ItineraryGenerator is the port for turning a trip into a concrete itinerary.
// Implementations (adapters) live under infrastructure.
type ItineraryGenerator interface {
	Generate(ctx context.Context, input GenerateItineraryInput) (*aggregate.Itinerary, error)
	RegenerateDay(ctx context.Context, input RegenerateDayInput) (*aggregate.ItineraryDay, error)
	RegenerateItem(ctx context.Context, input RegenerateItemInput) (*aggregate.ItineraryItem, error)
}
