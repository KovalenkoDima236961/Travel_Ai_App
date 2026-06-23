package places

import (
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
)

func TestNewUnsupportedProviderReturnsError(t *testing.T) {
	_, err := New(&config.Config{PlaceProvider: config.PlaceProviderConfig{Provider: "google"}}, zap.NewNop())
	if err == nil {
		t.Fatal("expected unsupported provider error")
	}
	if !strings.Contains(err.Error(), "unsupported PLACE_PROVIDER") {
		t.Fatalf("unexpected error: %v", err)
	}
}
