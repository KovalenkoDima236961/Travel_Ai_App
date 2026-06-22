// Package application defines the use-case layer's ports (interfaces) that
// adapters implement. Concrete use cases live in the service subpackage.
package application

import (
	"context"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

// ItineraryGenerator is the port for turning a trip into a concrete itinerary.
// It is the seam where the local mock is swapped for the future async AI
// Planning Service. Implementations (adapters) live under infrastructure.
type ItineraryGenerator interface {
	Generate(ctx context.Context, trip entity.Trip) (*aggregate.Itinerary, error)
}
