package copilot

import (
	"strings"
	"testing"
)

func TestFallbackResponseRespectsLanguage(t *testing.T) {
	response := fallbackResponse(
		IntentExplainHealth,
		SafeContext{Health: map[string]any{"score": 58, "level": "needs_attention", "summary": "Transport is missing."}},
		nil,
		"es",
	)
	if !strings.Contains(strings.ToLower(response.Answer), "salud") {
		t.Fatalf("expected Spanish fallback, got %q", response.Answer)
	}
}

