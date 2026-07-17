package search

import "strings"

func matchesTokens(query string, tokens []string, values ...string) bool {
	haystack := strings.ToLower(strings.Join(values, " "))
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
	if normalizedQuery != "" && strings.Contains(haystack, normalizedQuery) {
		return true
	}
	for _, token := range tokens {
		if strings.Contains(haystack, token) {
			return true
		}
	}
	return false
}

func includeTripScoped(scope Scope) bool {
	return scope == ScopeAll || scope == ScopeTrips || scope == ScopeCurrentTrip || scope == ScopeWorkspace
}
