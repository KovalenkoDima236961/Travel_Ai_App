package response

import (
	"encoding/json"
	"testing"
)

func TestSanitizePublicItineraryStripsPriceEnrichmentButKeepsEstimatedCost(t *testing.T) {
	raw := json.RawMessage(`{
		"destination":"Rome",
		"totalBudget":700,
		"days":[{
			"day":1,
			"title":"Day 1",
			"items":[{
				"time":"10:00",
				"type":"museum",
				"name":"Museum",
				"estimatedCost":{"amount":18,"currency":"EUR","category":"ticket","source":"provider"},
				"priceEnrichment":{"status":"matched","provider":"mock","matchConfidence":0.82}
			}]
		}]
	}`)

	got := sanitizePublicItinerary(raw).(map[string]any)
	if _, ok := got["totalBudget"]; ok {
		t.Fatalf("expected totalBudget stripped, got %+v", got)
	}
	days := got["days"].([]any)
	item := days[0].(map[string]any)["items"].([]any)[0].(map[string]any)
	if _, ok := item["priceEnrichment"]; ok {
		t.Fatalf("expected priceEnrichment stripped, got %+v", item)
	}
	if _, ok := item["estimatedCost"]; !ok {
		t.Fatalf("expected estimatedCost preserved, got %+v", item)
	}
}
