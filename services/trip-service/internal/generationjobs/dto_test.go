package generationjobs

import (
	"encoding/json"
	"testing"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

func TestNewJobResponse_ExposesSafeFailureDetails(t *testing.T) {
	code := ErrorAIGeneration
	rawMessage := "provider returned prompt and internal endpoint details"
	job := &entity.GenerationJob{
		JobType:      entity.GenerationJobTypeFullGeneration,
		Status:       entity.GenerationJobStatusFailed,
		ErrorCode:    &code,
		ErrorMessage: &rawMessage,
	}

	encoded, err := json.Marshal(NewJobResponse(job))
	if err != nil {
		t.Fatalf("marshal job response: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(encoded, &body); err != nil {
		t.Fatalf("decode job response: %v", err)
	}
	if _, ok := body["errorMessage"]; ok {
		t.Fatal("raw errorMessage must not be exposed")
	}
	if body["errorMessageSafe"] == rawMessage || body["errorMessageSafe"] == nil {
		t.Fatalf("expected a mapped safe error message, got %#v", body["errorMessageSafe"])
	}
	if body["canRetry"] != true {
		t.Fatalf("expected canRetry=true, got %#v", body["canRetry"])
	}
	if body["retryRecommendedMode"] != "simpler_request" {
		t.Fatalf("expected simpler request retry mode, got %#v", body["retryRecommendedMode"])
	}
}
