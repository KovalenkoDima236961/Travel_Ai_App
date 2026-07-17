// Package aiprivacy is the privacy boundary for payloads sent to the AI
// Planning Service or written to optional prompt logs.
package aiprivacy

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

const Redacted = "[REDACTED]"

var (
	emailPattern  = regexp.MustCompile(`(?i)\b[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}\b`)
	phonePattern  = regexp.MustCompile(`\+?[0-9][0-9 ()\-.]{7,}[0-9]`)
	bearerPattern = regexp.MustCompile(`(?i)\bbearer\s+[a-z0-9._~+/=\-]{12,}`)
	apiKeyPattern = regexp.MustCompile(`(?i)\b(?:sk|pk|api[_-]?key|token|secret)[_:\-= ]+[a-z0-9_./+\-=]{12,}`)

	redactions = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ai_privacy_redactions_total",
		Help: "Sensitive AI context values removed or redacted by Trip Service.",
	}, []string{"reason"})
)

func init() {
	prometheus.MustRegister(redactions)
}

type Report struct {
	RemovedFields int
	RedactedText  int
}

var removedKeys = map[string]string{
	"email": "email", "emailaddress": "email", "phone": "phone", "phonenumber": "phone",
	"homeaddress": "home_address", "exacthomeaddress": "home_address",
	"rawtext": "receipt_ocr", "receiptocr": "receipt_ocr", "receiptocrrawtext": "receipt_ocr",
	"expensenotes": "expense_notes", "privateexpensenotes": "expense_notes",
	"eventtitle": "calendar_detail", "eventdescription": "calendar_detail",
	"eventlocation": "calendar_detail", "attendees": "calendar_detail", "freebusyblocks": "calendar_detail",
	"calendareventtitle": "calendar_detail", "calendareventdescription": "calendar_detail",
	"calendareventlocation": "calendar_detail", "rawcomment": "comment", "rawcomments": "comment", "comments": "comment",
	"sharetoken": "share_secret", "sharepassword": "share_secret", "password": "secret",
	"accesstoken": "token", "refreshtoken": "token", "notificationtoken": "token",
	"apikey": "secret", "secret": "secret", "filepath": "file_path", "storagekey": "file_path",
	"userid": "internal_id", "actoruserid": "internal_id", "createdbyuserid": "internal_id",
	"selectedbyuserid": "internal_id", "workspaceid": "internal_id",
}

// SanitizeJSON removes disallowed fields and redacts PII/secret-like text while
// preserving the allowlisted travel-planning shape. Invalid JSON is rejected so
// callers can fail closed rather than forward an uninspected payload.
func SanitizeJSON(raw []byte) ([]byte, Report, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, Report{}, err
	}
	report := Report{}
	value = sanitizeValue(value, &report)
	clean, err := json.Marshal(value)
	return clean, report, err
}

// RedactText is safe for optional local-only diagnostic logging. Callers should
// still truncate the result and never enable prompt logging in strict envs.
func RedactText(value string) (string, int) {
	redacted := value
	count := 0
	for _, pattern := range []*regexp.Regexp{emailPattern, bearerPattern, apiKeyPattern} {
		matches := len(pattern.FindAllStringIndex(redacted, -1))
		if matches == 0 {
			continue
		}
		redacted = pattern.ReplaceAllString(redacted, Redacted)
		count += matches
	}
	redacted = phonePattern.ReplaceAllStringFunc(redacted, func(candidate string) string {
		if digitCount(candidate) < 10 {
			return candidate
		}
		count++
		return Redacted
	})
	if count > 0 {
		redactions.WithLabelValues("text_pattern").Add(float64(count))
	}
	return redacted, count
}

func digitCount(value string) int {
	count := 0
	for _, char := range value {
		if char >= '0' && char <= '9' {
			count++
		}
	}
	return count
}

func sanitizeValue(value any, report *Report) any {
	switch typed := value.(type) {
	case map[string]any:
		clean := make(map[string]any, len(typed))
		for key, child := range typed {
			normalized := normalizeKey(key)
			if reason, remove := removedKeys[normalized]; remove {
				report.RemovedFields++
				redactions.WithLabelValues(reason).Inc()
				continue
			}
			clean[key] = sanitizeValue(child, report)
		}
		return clean
	case []any:
		clean := make([]any, len(typed))
		for index, child := range typed {
			clean[index] = sanitizeValue(child, report)
		}
		return clean
	case string:
		clean, count := RedactText(typed)
		report.RedactedText += count
		return clean
	default:
		return value
	}
}

func normalizeKey(value string) string {
	replacer := strings.NewReplacer("_", "", "-", "", ".", "", " ", "")
	return strings.ToLower(replacer.Replace(strings.TrimSpace(value)))
}
