package service

import (
	"context"
	"encoding/json"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/aivalidation"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/planningconstraints"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/weathercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

func (s *Service) EvaluatePolicyForItinerary(
	ctx context.Context,
	trip entity.Trip,
	itinerary aggregate.Itinerary,
) (*workspacepolicies.Evaluation, error) {
	raw, err := json.Marshal(itinerary)
	if err != nil {
		return nil, err
	}
	proposed := trip
	proposed.Itinerary = raw
	evaluation, err := s.evaluateTripPolicyForTrip(ctx, &proposed)
	if err != nil {
		return nil, err
	}
	return &evaluation, nil
}

func generationQualityMetadata(quality aivalidation.GenerationQualityMetadata) map[string]any {
	return aivalidation.MetadataEnvelope(quality)
}

func mergeGenerationQualityMetadata(base map[string]any, quality aivalidation.GenerationQualityMetadata) map[string]any {
	if base == nil {
		base = map[string]any{}
	}
	for key, value := range generationQualityMetadata(quality) {
		base[key] = value
	}
	return base
}

func generationTypeForVersionSource(source entity.ItineraryVersionSource) aivalidation.GenerationType {
	switch source {
	case entity.ItineraryVersionSourceRegenerateDay:
		return aivalidation.GenerationTypeDayRegeneration
	case entity.ItineraryVersionSourceRegenerateItem:
		return aivalidation.GenerationTypeItemRegeneration
	case entity.ItineraryVersionSourceCreatedFromTemplateAI:
		return aivalidation.GenerationTypeTemplateAdaptation
	case entity.ItineraryVersionSourceAIPolicyRepairApplied:
		return aivalidation.GenerationTypePolicyRepair
	case entity.ItineraryVersionSourceBudgetOptimizationApplied:
		return aivalidation.GenerationTypeBudgetOptimizationDay
	default:
		return aivalidation.GenerationTypeFullItinerary
	}
}

func (s *Service) validateGeneratedItinerary(
	ctx context.Context,
	trip entity.Trip,
	itinerary aggregate.Itinerary,
	source entity.ItineraryVersionSource,
	metadata map[string]any,
	planning *planningconstraints.PlanningConstraints,
	weather *weathercontext.WeatherForecast,
	outputLanguage string,
) (aggregate.Itinerary, map[string]any, *aivalidation.PipelineResult, error) {
	if s.generationReliability == nil {
		itinerary, metadata = applyGroundingValidation(itinerary, metadata)
		return itinerary, metadata, nil, nil
	}
	result, err := s.generationReliability.ValidateAndRepair(ctx, aivalidation.PipelineInput{
		GenerationType:      generationTypeForVersionSource(source),
		AIOutput:            itinerary,
		Trip:                trip,
		PlanningConstraints: planning,
		WeatherForecast:     weather,
		RepairAllowed:       true,
		MinimumSaveLevel:    aivalidation.MinimumSaveLevelNoBlockingIssues,
		OutputLanguage:      outputLanguage,
	})
	if err != nil {
		return itinerary, metadata, nil, err
	}
	if !result.SaveAllowed {
		itinerary, metadata = applyGroundingValidation(itinerary, mergeGenerationQualityMetadata(metadata, result.GenerationQuality))
		return itinerary, metadata, &result, aivalidation.NewValidationError(result)
	}
	finalOutput, finalMetadata := applyGroundingValidation(result.FinalOutput, mergeGenerationQualityMetadata(metadata, result.GenerationQuality))
	return finalOutput, finalMetadata, &result, nil
}
