package personalization

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

func TestBuildUsesSavedPreferencesAndFeedback(t *testing.T) {
	walking := 8.0
	repo := &fakeRepository{items: []Feedback{{
		UserID: uuid.New(), EntityType: "destination_suggestion", FeedbackType: FeedbackPreferTrains,
	}}}
	svc := New(repo, nil)
	contextValue, err := svc.Build(context.Background(), BuildInput{
		UserID: uuid.New(), Source: SourceTripDiscovery,
		UserContext: usercontext.UserContext{Preferences: &usercontext.UserPreferences{
			TravelStyles: []string{"food", "culture"}, PreferredTransport: []string{"public_transport"},
			MaxWalkingKmPerDay: &walking,
		}},
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if contextValue.Completeness.Score == 0 {
		t.Fatalf("expected non-zero completeness: %+v", contextValue.Completeness)
	}
	if contextValue.FeedbackSignals.PreferTrainCount != 1 {
		t.Fatalf("expected train feedback: %+v", contextValue.FeedbackSignals)
	}
	if !contains(contextValue.DerivedSignals.TransportBias, "train") {
		t.Fatalf("expected derived train bias: %+v", contextValue.DerivedSignals)
	}
}

func TestBuildPolicyRemovesBlockedPersonalTransport(t *testing.T) {
	svc := New(&fakeRepository{}, nil)
	contextValue, err := svc.Build(context.Background(), BuildInput{
		UserID: uuid.New(), Source: SourceRouteAlternatives,
		UserContext: usercontext.UserContext{Preferences: &usercontext.UserPreferences{PreferredTransport: []string{"flight", "train"}}},
		WorkspacePolicy: &workspacepolicies.Policy{Rules: workspacepolicies.RulesDocument{Rules: workspacepolicies.Rules{
			DisallowedTransportModes: workspacepolicies.TransportRule{Rule: workspacepolicies.Rule{Enabled: true}, Modes: []string{"flight"}},
		}}},
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if contains(contextValue.Preferences.PreferredTransport, "flight") {
		t.Fatalf("blocked transport leaked into preferences: %+v", contextValue.Preferences)
	}
	if len(contextValue.Warnings) == 0 {
		t.Fatal("expected policy override warning")
	}
}

func TestSanitizeMetadataDropsSensitiveFields(t *testing.T) {
	metadata, err := sanitizeMetadata(map[string]any{"destination": "Vienna", "receiptOcr": "secret", "calendar": "private"})
	if err != nil {
		t.Fatalf("sanitize: %v", err)
	}
	if metadata["destination"] != "Vienna" {
		t.Fatalf("allowed metadata missing: %+v", metadata)
	}
	if _, ok := metadata["receiptOcr"]; ok {
		t.Fatalf("receipt OCR must not be stored: %+v", metadata)
	}
}

type fakeRepository struct{ items []Feedback }

func (f *fakeRepository) Create(_ context.Context, item Feedback) (Feedback, error) {
	f.items = append(f.items, item)
	return item, nil
}
func (f *fakeRepository) ListByUser(_ context.Context, _ uuid.UUID, _ int) ([]Feedback, error) {
	return append([]Feedback(nil), f.items...), nil
}
func (f *fakeRepository) ClearByUser(_ context.Context, _ uuid.UUID) error { f.items = nil; return nil }
func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
