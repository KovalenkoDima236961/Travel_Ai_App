package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

func adaptationTestTemplate(t *testing.T) *entity.TripTemplate {
	t.Helper()
	body := map[string]any{
		"schemaVersion": 1,
		"durationDays":  2,
		"days": []map[string]any{
			{
				"dayOffset": 0,
				"title":     "Old Town",
				"items": []map[string]any{
					{
						"name":      "Prague Old Town walk",
						"type":      "activity",
						"startTime": "09:00",
						"place": map[string]any{
							"name":            "Old Town Square",
							"category":        "landmark",
							"provider":        "google",
							"providerPlaceId": "SECRET_PLACE_ID",
							"address":         "Private address",
						},
					},
				},
			},
		},
		// Private metadata that must never reach the AI prompt.
		"metadata": map[string]any{
			"createdFromTripId":    "SECRET_TRIP_ID",
			"createdFromTripTitle": "Private trip title",
		},
		"summary": map[string]any{"estimatedTotalAmount": 250, "currency": "EUR"},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal template body: %v", err)
	}
	return &entity.TripTemplate{
		ID:           uuid.New(),
		Title:        "Prague Weekend",
		DurationDays: 2,
		TemplateJSON: raw,
	}
}

func TestBuildAdaptationTemplateStripsPrivateData(t *testing.T) {
	template := adaptationTestTemplate(t)
	built := buildAdaptationTemplate(template)

	if built.DurationDays != 2 || len(built.Days) != 1 {
		t.Fatalf("unexpected structure: duration=%d days=%d", built.DurationDays, len(built.Days))
	}
	if len(built.Days[0].Items) != 1 {
		t.Fatalf("expected one item, got %d", len(built.Days[0].Items))
	}
	place := built.Days[0].Items[0].Place
	if place == nil || place.Name != "Old Town Square" || place.Category != "landmark" {
		t.Fatalf("place name/category not preserved: %+v", place)
	}

	// The mapped structure carries only name/category on places; serialize it and
	// assert no private identifiers, provider ids, addresses, or trip metadata
	// can reach the model prompt.
	encoded, err := json.Marshal(built)
	if err != nil {
		t.Fatalf("marshal built template: %v", err)
	}
	for _, forbidden := range []string{
		"SECRET_TRIP_ID", "Private trip title", "SECRET_PLACE_ID",
		"Private address", "providerPlaceId", "createdFromTripId", "metadata",
	} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("sanitized template leaked %q: %s", forbidden, encoded)
		}
	}
}

func TestNormalizeCreateTemplateAdaptationInputDefaultsAndValidation(t *testing.T) {
	template := adaptationTestTemplate(t)

	// Duration defaults to the template duration when omitted; title defaults from
	// the template; pace defaults to balanced.
	out, err := normalizeCreateTemplateAdaptationInput(appdto.CreateTemplateAdaptationInput{
		Destination: "Vienna",
		StartDate:   "2026-09-10",
	}, template)
	if err != nil {
		t.Fatalf("normalize valid input: %v", err)
	}
	if out.DurationDays != 2 {
		t.Fatalf("expected duration to default to 2, got %d", out.DurationDays)
	}
	if out.Pace != "balanced" {
		t.Fatalf("expected default pace balanced, got %s", out.Pace)
	}
	if out.Travelers == nil || *out.Travelers != 1 {
		t.Fatalf("expected default travelers 1, got %v", out.Travelers)
	}
	if !strings.Contains(out.Title, "Prague Weekend") {
		t.Fatalf("expected title defaulted from template, got %q", out.Title)
	}

	// Invalid duration is rejected.
	if _, err := normalizeCreateTemplateAdaptationInput(appdto.CreateTemplateAdaptationInput{
		Destination:  "Vienna",
		StartDate:    "2026-09-10",
		DurationDays: 40,
	}, template); err == nil {
		t.Fatal("expected invalid duration to be rejected")
	}

	// Invalid pace is rejected.
	if _, err := normalizeCreateTemplateAdaptationInput(appdto.CreateTemplateAdaptationInput{
		Destination: "Vienna",
		StartDate:   "2026-09-10",
		Pace:        "sprint",
	}, template); err == nil {
		t.Fatal("expected invalid pace to be rejected")
	}

	// Missing destination (and no template hint) is rejected.
	if _, err := normalizeCreateTemplateAdaptationInput(appdto.CreateTemplateAdaptationInput{
		StartDate: "2026-09-10",
	}, template); err == nil {
		t.Fatal("expected missing destination to be rejected")
	}
}
