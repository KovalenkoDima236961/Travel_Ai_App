package aiprivacy

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSanitizeJSONRemovesPrivateContextAndPreservesSummary(t *testing.T) {
	raw := []byte(`{
		"email":"traveler@example.com",
		"phoneNumber":"+421 900 123 456",
		"receiptOcrRawText":"CARD 1234",
		"privateExpenseNotes":"client dinner",
		"eventTitle":"Oncology appointment",
		"shareToken":"raw-share-token",
		"apiKey":"super-secret-provider-key",
		"filePath":"/private/receipts/a.pdf",
		"summary":"Two travelers prefer museums. Contact traveler@example.com",
		"budgetTotal":1200
	}`)

	clean, report, err := SanitizeJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if report.RemovedFields < 7 || report.RedactedText == 0 {
		t.Fatalf("unexpected report: %+v", report)
	}
	text := string(clean)
	for _, forbidden := range []string{"traveler@example.com", "CARD 1234", "client dinner", "Oncology", "raw-share-token", "provider-key", "/private/receipts"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("sanitized JSON leaked %q: %s", forbidden, text)
		}
	}
	var decoded map[string]any
	if err := json.Unmarshal(clean, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["budgetTotal"] != float64(1200) || !strings.Contains(decoded["summary"].(string), "Two travelers") {
		t.Fatalf("safe summary was not preserved: %#v", decoded)
	}
}

func TestRedactTextCoversBearerAndAPIKeyLikeValues(t *testing.T) {
	clean, count := RedactText("Bearer abcdefghijklmnopqrstuvwxyz api_key=abcdefghijklmnop")
	if count != 2 || strings.Contains(clean, "abcdefgh") {
		t.Fatalf("unexpected redaction: %q count=%d", clean, count)
	}
}

func TestRedactTextPreservesISODateTime(t *testing.T) {
	const value = "2026-09-10T10:30:00Z"
	clean, count := RedactText(value)
	if clean != value || count != 0 {
		t.Fatalf("date/time was redacted: %q count=%d", clean, count)
	}
}

func TestRedactTextPreservesNumericHeavyUUID(t *testing.T) {
	const value = "b0000000-0000-4000-8000-000000000001"
	clean, count := RedactText(value)
	if clean != value || count != 0 {
		t.Fatalf("UUID was redacted as a phone number: %q count=%d", clean, count)
	}
}
