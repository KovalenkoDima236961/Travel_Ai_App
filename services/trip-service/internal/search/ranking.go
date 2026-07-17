package search

import (
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

func scoreResult(query string, tokens []string, result Result, currentTripID *uuid.UUID, now time.Time) float64 {
	title := strings.ToLower(strings.TrimSpace(result.Title))
	description := strings.ToLower(strings.TrimSpace(result.Description + " " + result.Context + " " + result.WorkspaceName))
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))

	score := typePriority(result.Type)
	if title == normalizedQuery {
		score += 0.55
	} else if strings.HasPrefix(title, normalizedQuery) {
		score += 0.38
	} else if strings.Contains(title, normalizedQuery) {
		score += 0.28
	}

	if len(tokens) > 0 {
		var titleMatches, descriptionMatches int
		for _, token := range tokens {
			if strings.Contains(title, token) {
				titleMatches++
			}
			if strings.Contains(description, token) {
				descriptionMatches++
			}
		}
		score += 0.26 * (float64(titleMatches) / float64(len(tokens)))
		score += 0.12 * (float64(descriptionMatches) / float64(len(tokens)))
	}

	if currentTripID != nil && result.TripID != nil && *currentTripID == *result.TripID {
		score += 0.25
	}

	if !result.UpdatedAt.IsZero() {
		age := now.Sub(result.UpdatedAt)
		switch {
		case age <= 24*time.Hour:
			score += 0.12
		case age <= 7*24*time.Hour:
			score += 0.08
		case age <= 30*24*time.Hour:
			score += 0.04
		}
	}

	return score
}

func typePriority(resultType ResultType) float64 {
	switch resultType {
	case ResultTypeTrip:
		return 0.68
	case ResultTypeRouteStop, ResultTypeRouteLeg, ResultTypeTransportOption, ResultTypeItineraryItem:
		return 0.6
	case ResultTypeChecklistItem, ResultTypeReminder:
		return 0.52
	case ResultTypeExpense, ResultTypeReceipt:
		return 0.48
	case ResultTypeTemplate, ResultTypeWorkspace:
		return 0.38
	case ResultTypeSetting, ResultTypeCommand:
		return 0.3
	case ResultTypeOpsPage:
		return 0.18
	default:
		return 0.25
	}
}

func sortResults(results []Result) {
	sort.SliceStable(results, func(i, j int) bool {
		left, right := results[i], results[j]
		if left.Score != right.Score {
			return left.Score > right.Score
		}
		if left.Category != right.Category {
			return left.Category < right.Category
		}
		if left.Title != right.Title {
			return left.Title < right.Title
		}
		return left.ID < right.ID
	})
}

func buildResponse(query string, results []Result, limit, perCategoryLimit int) Response {
	sortResults(results)
	items := make([]Result, 0, min(limit, len(results)))
	categoryCounts := map[string]int{}
	hasMore := false

	for _, result := range results {
		if len(items) >= limit {
			hasMore = true
			break
		}
		if categoryCounts[result.Category] >= perCategoryLimit {
			hasMore = true
			continue
		}
		items = append(items, result)
		categoryCounts[result.Category]++
	}

	groupIndex := map[string]int{}
	groups := make([]Group, 0)
	for _, item := range items {
		index, ok := groupIndex[item.Category]
		if !ok {
			index = len(groups)
			groupIndex[item.Category] = index
			groups = append(groups, Group{Title: item.Category})
		}
		groups[index].Items = append(groups[index].Items, item)
	}

	return Response{Query: query, Items: items, Groups: groups, HasMore: hasMore}
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
