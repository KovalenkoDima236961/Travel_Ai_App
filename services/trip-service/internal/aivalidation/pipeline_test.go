package aivalidation

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

func TestPipelineRepairsCriticalDayCountIssue(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxRepairAttempts = 1
	client := &pipelineRepairClient{}
	pipeline := NewPipeline(NewValidator(cfg), client, fakePolicyEvaluator{}, cfg, zap.NewNop())
	trip := entity.Trip{Destination: "Vienna", Days: 2, BudgetCurrency: "EUR"}
	output := aggregate.Itinerary{
		Destination: "Vienna",
		Currency:    "EUR",
		Days: []aggregate.ItineraryDay{
			{
				Day:   1,
				Title: "Arrival",
				Items: []aggregate.ItineraryItem{
					{Time: "10:00", Type: "activity", Name: "Old town walk"},
				},
			},
		},
	}

	result, err := pipeline.ValidateAndRepair(context.Background(), PipelineInput{
		GenerationType:    GenerationTypeFullItinerary,
		AIOutput:          output,
		Trip:              trip,
		RepairAllowed:     true,
		MaxRepairAttempts: 1,
		MinimumSaveLevel:  MinimumSaveLevelNoBlockingIssues,
	})
	if err != nil {
		t.Fatalf("ValidateAndRepair returned error: %v", err)
	}
	if !result.SaveAllowed {
		t.Fatalf("expected repaired output to be saveable, got quality %#v", result.GenerationQuality)
	}
	if client.calls != 1 {
		t.Fatalf("expected one repair call, got %d", client.calls)
	}
	if len(result.FinalOutput.Days) != 2 {
		t.Fatalf("expected repaired output to have 2 days, got %d", len(result.FinalOutput.Days))
	}
	if result.GenerationQuality.Status != StatusRepairedAndValidated &&
		result.GenerationQuality.Status != StatusRepairedWithWarnings {
		t.Fatalf("expected repaired status, got %s", result.GenerationQuality.Status)
	}
	if result.GenerationQuality.RepairAttempts != 1 {
		t.Fatalf("expected one repair attempt in metadata, got %d", result.GenerationQuality.RepairAttempts)
	}
}

type pipelineRepairClient struct {
	calls int
}

func (c *pipelineRepairClient) RepairGenerationOutput(_ context.Context, request RepairGenerationOutputRequest) (*RepairGenerationOutputResponse, error) {
	c.calls++
	out := request.CurrentOutput
	out.Days = append(out.Days, aggregate.ItineraryDay{
		Day:   2,
		Title: "Flexible repaired day",
		Items: []aggregate.ItineraryItem{
			{Time: "10:00", Type: "activity", Name: "Flexible repaired item"},
		},
	})
	return &RepairGenerationOutputResponse{
		RepairedOutput: out,
		ChangesMade: []RepairChange{
			{Type: "day_added", Description: "Added missing day"},
		},
	}, nil
}

func (c *pipelineRepairClient) ProviderMode() string {
	return "test"
}

type fakePolicyEvaluator struct{}

func (fakePolicyEvaluator) EvaluatePolicyForItinerary(context.Context, entity.Trip, aggregate.Itinerary) (*workspacepolicies.Evaluation, error) {
	return nil, nil
}
