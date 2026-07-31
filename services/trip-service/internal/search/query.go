package search

import (
	"errors"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var (
	ErrInvalidQuery  = errors.New("search query contains invalid characters")
	ErrQueryTooLong  = errors.New("search query is too long")
	ErrInvalidFilter = errors.New("search filter is invalid")
)

func normalizeQuery(raw string, maxRunes int) (string, error) {
	for _, r := range raw {
		if unicode.IsControl(r) {
			return "", ErrInvalidQuery
		}
	}
	normalized := norm.NFKC.String(raw)
	normalized = strings.TrimSpace(strings.Join(strings.Fields(normalized), " "))
	if maxRunes > 0 && len([]rune(normalized)) > maxRunes {
		return "", ErrQueryTooLong
	}
	return normalized, nil
}

func tokenize(query string) []string {
	parts := strings.Fields(strings.ToLower(query))
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.Trim(part, ".,:;!?()[]{}\"'")
		if len([]rune(part)) < 2 {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		out = append(out, part)
		seen[part] = struct{}{}
	}
	return out
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}

func runeLen(value string) int {
	return len([]rune(strings.TrimSpace(value)))
}
