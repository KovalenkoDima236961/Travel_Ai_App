// Package application defines the use-case layer's ports (interfaces) that
// adapters implement. Concrete use cases live in the service subpackage.
package application

import (
	"context"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budgetoptimization"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/planningconstraints"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/routealternatives"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/templateadaptation"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/triprepair"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/weathercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

// GenerateItineraryInput is the internal generator request. Trip Service owns
// loading trusted user context; frontend callers cannot submit these fields.
type GenerateItineraryInput struct {
	Trip                       entity.Trip
	Instruction                string
	OutputLanguage             string
	UserProfile                *usercontext.UserProfile
	UserPreferences            *usercontext.UserPreferences
	WeatherForecast            *weathercontext.WeatherForecast
	WorkspacePolicyConstraints *workspacepolicies.AIConstraints
	PlanningConstraints        *planningconstraints.PlanningConstraints
}

// RegenerateDayInput is the internal generator request for replacing one day
// in an existing itinerary.
type RegenerateDayInput struct {
	Trip                       entity.Trip
	OutputLanguage             string
	CurrentItinerary           aggregate.Itinerary
	DayNumber                  int
	Instruction                string
	UserProfile                *usercontext.UserProfile
	UserPreferences            *usercontext.UserPreferences
	WeatherForecast            *weathercontext.WeatherForecast
	WorkspacePolicyConstraints *workspacepolicies.AIConstraints
	PlanningConstraints        *planningconstraints.PlanningConstraints
}

// RegenerateItemInput is the internal generator request for replacing one item
// in an existing itinerary day. ItemIndex is zero-based.
type RegenerateItemInput struct {
	Trip                       entity.Trip
	OutputLanguage             string
	CurrentItinerary           aggregate.Itinerary
	DayNumber                  int
	ItemIndex                  int
	Instruction                string
	UserProfile                *usercontext.UserProfile
	UserPreferences            *usercontext.UserPreferences
	WeatherForecast            *weathercontext.WeatherForecast
	WorkspacePolicyConstraints *workspacepolicies.AIConstraints
	PlanningConstraints        *planningconstraints.PlanningConstraints
}

// ItineraryGenerator is the port for turning a trip into a concrete itinerary.
// Implementations (adapters) live under infrastructure.
type ItineraryGenerator interface {
	Generate(ctx context.Context, input GenerateItineraryInput) (*aggregate.Itinerary, error)
	RegenerateDay(ctx context.Context, input RegenerateDayInput) (*aggregate.ItineraryDay, error)
	RegenerateItem(ctx context.Context, input RegenerateItemInput) (*aggregate.ItineraryItem, error)
	OptimizeBudgetDay(ctx context.Context, input budgetoptimization.OptimizeDayInput) (*budgetoptimization.ProposalContent, error)
	AdaptTemplate(ctx context.Context, input templateadaptation.AdaptInput) (*templateadaptation.AdaptResult, error)
	RepairItinerary(ctx context.Context, input triprepair.Input) (*triprepair.ProposalContent, error)
	SuggestRouteAlternatives(ctx context.Context, input routealternatives.AIRequest) (*routealternatives.Response, error)
}
