package aidataset

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

func rawOrEmptyObject(raw json.RawMessage) json.RawMessage {
	if len(bytes.TrimSpace(raw)) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}

func rawOrNull(raw json.RawMessage) json.RawMessage {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	return raw
}

func normalizeJSON(raw json.RawMessage) ([]byte, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return []byte("null"), nil
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	normalized := normalizeValue(value)
	return json.Marshal(normalized)
}

func normalizeValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			out[key] = normalizeValue(typed[key])
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = normalizeValue(typed[i])
		}
		return out
	default:
		return typed
	}
}

func checksumJSON(parts ...json.RawMessage) (string, error) {
	hash := sha256.New()
	for _, part := range parts {
		normalized, err := normalizeJSON(part)
		if err != nil {
			return "", fmt.Errorf("normalize json: %w", err)
		}
		_, _ = hash.Write(normalized)
		_, _ = hash.Write([]byte{'\n'})
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func jsonObject(raw json.RawMessage) (map[string]any, bool) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]any{}, true
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, false
	}
	obj, ok := value.(map[string]any)
	return obj, ok
}

func jsonMap(raw json.RawMessage) map[string]any {
	obj, ok := jsonObject(raw)
	if !ok {
		return map[string]any{}
	}
	return obj
}

func mustJSON(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return raw
}
