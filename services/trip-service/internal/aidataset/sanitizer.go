package aidataset

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

var (
	emailPattern     = regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`)
	phonePattern     = regexp.MustCompile(`(?m)\b(?:\+?\d{1,3}[\s.-]?)?(?:\(?\d{3}\)?[\s.-]?)\d{3}[\s.-]?\d{4}\b`)
	jwtPattern       = regexp.MustCompile(`\beyJ[a-zA-Z0-9_-]{8,}\.[a-zA-Z0-9_-]{8,}\.[a-zA-Z0-9_-]{8,}\b`)
	longTokenPattern = regexp.MustCompile(`\b[a-zA-Z0-9_-]{32,}\b`)
	uuidPattern      = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\b`)
)

type SanitizationResult struct {
	InputJSON       json.RawMessage `json:"input"`
	GroundingJSON   json.RawMessage `json:"grounding,omitempty"`
	OutputJSON      json.RawMessage `json:"output"`
	RemovedFields   []string        `json:"removedFields"`
	Warnings        []string        `json:"warnings"`
	Status          string          `json:"status"`
	Checksum        string          `json:"checksum"`
	PrivateDetected bool            `json:"privateDetected"`
}

type sensitiveField struct {
	Reason   string
	HighRisk bool
}

var sensitiveFields = map[string]sensitiveField{
	"userid":                {"user identifier removed", false},
	"user_id":               {"user identifier removed", false},
	"owneruserid":           {"user identifier removed", false},
	"createdbyuserid":       {"user identifier removed", false},
	"reviewedbyuserid":      {"user identifier removed", false},
	"email":                 {"email removed", true},
	"useremail":             {"email removed", true},
	"displayname":           {"display name removed", true},
	"fullname":              {"name removed", true},
	"phone":                 {"phone removed", true},
	"phonenumber":           {"phone removed", true},
	"homeaddress":           {"home address removed", true},
	"addresshome":           {"home address removed", true},
	"exacthomeaddress":      {"home address removed", true},
	"streetaddress":         {"street address removed", true},
	"accesstoken":           {"access token removed", true},
	"access_token":          {"access token removed", true},
	"refreshtoken":          {"refresh token removed", true},
	"refresh_token":         {"refresh token removed", true},
	"oauthtoken":            {"oauth token removed", true},
	"apikey":                {"api key removed", true},
	"api_key":               {"api key removed", true},
	"providerapikey":        {"provider key removed", true},
	"publicsharetoken":      {"share token removed", true},
	"sharetoken":            {"share token removed", true},
	"password":              {"password removed", true},
	"passwordhash":          {"password hash removed", true},
	"filepath":              {"file path removed", true},
	"file_path":             {"file path removed", true},
	"receiptmetadata":       {"receipt metadata removed", true},
	"receiptocrtext":        {"receipt OCR text removed", true},
	"ocrtext":               {"OCR text removed", true},
	"calendar":              {"calendar details removed", true},
	"calendarevents":        {"calendar details removed", true},
	"freebusy":              {"calendar availability removed", true},
	"comments":              {"comments removed", true},
	"comment":               {"comment removed", true},
	"privatenotes":          {"private notes removed", true},
	"private_notes":         {"private notes removed", true},
	"collaborationmessage":  {"collaboration message removed", true},
	"collaborationmessages": {"collaboration messages removed", true},
	"messages":              {"messages removed", true},
	"rawlogs":               {"raw logs removed", true},
	"logs":                  {"raw logs removed", true},
	"requestid":             {"request id removed", false},
	"request_id":            {"request id removed", false},
	"traceid":               {"trace id removed", false},
	"jobid":                 {"job id removed", false},
	"tripid":                {"trip id removed", false},
	"workspaceid":           {"workspace id removed", false},
	"systemprompt":          {"system prompt removed", true},
	"system_prompt":         {"system prompt removed", true},
	"hiddenprompt":          {"hidden prompt removed", true},
	"chainofthought":        {"hidden reasoning removed", true},
	"chain_of_thought":      {"hidden reasoning removed", true},
}

var allowedIDFields = map[string]struct{}{
	"placeid":         {},
	"providerplaceid": {},
	"groundingid":     {},
	"groundingids":    {},
	"sourceid":        {},
	"sourcekey":       {},
	"destinationid":   {},
}

func SanitizeExample(input, grounding, output json.RawMessage) (SanitizationResult, error) {
	removed := make([]string, 0)
	warnings := make([]string, 0)
	highRisk := false

	cleanInput, err := sanitizeRaw(input, "input", &removed, &warnings, &highRisk)
	if err != nil {
		return SanitizationResult{}, err
	}
	cleanGrounding, err := sanitizeRaw(grounding, "grounding", &removed, &warnings, &highRisk)
	if err != nil {
		return SanitizationResult{}, err
	}
	cleanOutput, err := sanitizeRaw(output, "output", &removed, &warnings, &highRisk)
	if err != nil {
		return SanitizationResult{}, err
	}

	removed = dedupeSorted(removed)
	warnings = dedupeSorted(warnings)
	checksum, err := checksumJSON(cleanInput, cleanGrounding, cleanOutput)
	if err != nil {
		return SanitizationResult{}, err
	}
	status := SanitizationPassed
	if highRisk {
		status = SanitizationFailed
	} else if len(warnings) > 0 || len(removed) > 0 {
		status = SanitizationNeedsReview
	}
	return SanitizationResult{
		InputJSON:       cleanInput,
		GroundingJSON:   cleanGrounding,
		OutputJSON:      cleanOutput,
		RemovedFields:   removed,
		Warnings:        warnings,
		Status:          status,
		Checksum:        checksum,
		PrivateDetected: highRisk,
	}, nil
}

func sanitizeRaw(raw json.RawMessage, path string, removed, warnings *[]string, highRisk *bool) (json.RawMessage, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("decode %s json: %w", path, err)
	}
	sanitized := sanitizeValue(value, path, removed, warnings, highRisk)
	out, err := json.Marshal(sanitized)
	if err != nil {
		return nil, fmt.Errorf("marshal sanitized %s: %w", path, err)
	}
	return out, nil
}

func sanitizeValue(value any, path string, removed, warnings *[]string, highRisk *bool) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			childPath := path + "." + key
			normalizedKey := normalizeFieldName(key)
			if sensitive, ok := sensitiveFields[normalizedKey]; ok {
				*removed = append(*removed, childPath)
				*warnings = append(*warnings, sensitive.Reason)
				if sensitive.HighRisk {
					*highRisk = true
				}
				continue
			}
			if looksLikeInternalIDField(normalizedKey) {
				*removed = append(*removed, childPath)
				*warnings = append(*warnings, "internal id removed")
				continue
			}
			out[key] = sanitizeValue(child, childPath, removed, warnings, highRisk)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = sanitizeValue(typed[i], fmt.Sprintf("%s[%d]", path, i), removed, warnings, highRisk)
		}
		return out
	case string:
		return sanitizeString(typed, path, removed, warnings, highRisk)
	default:
		return typed
	}
}

func sanitizeString(value, path string, removed, warnings *[]string, highRisk *bool) string {
	out := value
	if emailPattern.MatchString(out) {
		out = emailPattern.ReplaceAllString(out, "[redacted-email]")
		*warnings = append(*warnings, "email redacted")
		*removed = append(*removed, path)
		*highRisk = true
	}
	if phonePattern.MatchString(out) {
		out = phonePattern.ReplaceAllString(out, "[redacted-phone]")
		*warnings = append(*warnings, "phone redacted")
		*removed = append(*removed, path)
		*highRisk = true
	}
	if jwtPattern.MatchString(out) {
		out = jwtPattern.ReplaceAllString(out, "[redacted-token]")
		*warnings = append(*warnings, "jwt-like token redacted")
		*removed = append(*removed, path)
		*highRisk = true
	}
	if strings.Contains(out, "?") || strings.Contains(out, "&") {
		stripped, changed := stripSecretQueryParams(out)
		if changed {
			out = stripped
			*warnings = append(*warnings, "secret URL query parameter stripped")
			*removed = append(*removed, path)
			*highRisk = true
		}
	}
	if longTokenPattern.MatchString(out) && containsSecretWord(out) {
		out = longTokenPattern.ReplaceAllString(out, "[redacted-secret]")
		*warnings = append(*warnings, "token-like secret redacted")
		*removed = append(*removed, path)
		*highRisk = true
	}
	if uuidPattern.MatchString(out) && strings.Contains(strings.ToLower(path), "internal") {
		out = uuidPattern.ReplaceAllString(out, "[redacted-id]")
		*warnings = append(*warnings, "internal UUID redacted")
		*removed = append(*removed, path)
	}
	return out
}

func stripSecretQueryParams(value string) (string, bool) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.RawQuery == "" {
		return value, false
	}
	query := parsed.Query()
	changed := false
	for key := range query {
		if isSecretQueryKey(key) {
			query.Del(key)
			changed = true
		}
	}
	if !changed {
		return value, false
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), true
}

func isSecretQueryKey(key string) bool {
	switch normalizeFieldName(key) {
	case "token", "accesstoken", "refreshtoken", "apikey", "key", "signature", "password", "sharetoken", "code", "secret":
		return true
	default:
		return false
	}
}

func containsSecretWord(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "token") ||
		strings.Contains(lower, "secret") ||
		strings.Contains(lower, "apikey") ||
		strings.Contains(lower, "api_key")
}

func normalizeFieldName(key string) string {
	replacer := strings.NewReplacer("-", "", "_", "", " ", "", ".", "")
	return replacer.Replace(strings.ToLower(strings.TrimSpace(key)))
}

func looksLikeInternalIDField(normalizedKey string) bool {
	if _, ok := allowedIDFields[normalizedKey]; ok {
		return false
	}
	return normalizedKey == "id" ||
		strings.HasSuffix(normalizedKey, "uuid") ||
		strings.HasSuffix(normalizedKey, "databaseid") ||
		strings.HasSuffix(normalizedKey, "internalid")
}

func dedupeSorted(values []string) []string {
	if len(values) == 0 {
		return values
	}
	set := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func SanitizationMetadata(result SanitizationResult) json.RawMessage {
	sum := sha256.Sum256([]byte(strings.Join(result.RemovedFields, "\n") + "\n" + strings.Join(result.Warnings, "\n")))
	return mustJSON(map[string]any{
		"removedFields":    result.RemovedFields,
		"warnings":         result.Warnings,
		"checksum":         result.Checksum,
		"privateDetected":  result.PrivateDetected,
		"metadataChecksum": hex.EncodeToString(sum[:]),
	})
}
