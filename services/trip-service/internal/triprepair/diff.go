package triprepair

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

const maxDiffChanges = 100

func BuildDiff(before, after aggregate.Itinerary) Diff {
	diff := Diff{
		DaysChanged:   []Change{},
		ItemsAdded:    []Change{},
		ItemsRemoved:  []Change{},
		ItemsModified: []Change{},
		ItemsMoved:    []Change{},
	}
	used := 0
	beforeDays := daysByNumber(before)
	afterDays := daysByNumber(after)

	for dayNumber, afterDay := range afterDays {
		beforeDay, ok := beforeDays[dayNumber]
		if !ok {
			if used < maxDiffChanges {
				day := dayNumber
				diff.DaysChanged = append(diff.DaysChanged, Change{
					Type:      "day_added",
					DayNumber: &day,
					After:     map[string]any{"title": afterDay.Title},
				})
			}
			used++
			continue
		}
		if strings.TrimSpace(beforeDay.Title) != strings.TrimSpace(afterDay.Title) {
			if used < maxDiffChanges {
				day := dayNumber
				diff.DaysChanged = append(diff.DaysChanged, Change{
					Type:      "day_modified",
					DayNumber: &day,
					FieldChanges: []FieldChange{{
						Field:  "title",
						Before: beforeDay.Title,
						After:  afterDay.Title,
					}},
				})
			}
			used++
		}
		used = diffItems(beforeDay, afterDay, &diff, used)
	}
	for dayNumber, beforeDay := range beforeDays {
		if _, ok := afterDays[dayNumber]; ok {
			continue
		}
		if used < maxDiffChanges {
			day := dayNumber
			diff.DaysChanged = append(diff.DaysChanged, Change{
				Type:      "day_removed",
				DayNumber: &day,
				Before:    map[string]any{"title": beforeDay.Title},
			})
		}
		used++
	}
	if used > maxDiffChanges {
		diff.Warnings = append(diff.Warnings, "Large repair changed many itinerary items.")
	}
	return diff
}

func diffItems(before, after aggregate.ItineraryDay, diff *Diff, used int) int {
	beforeMatched := map[int]bool{}
	afterMatched := map[int]bool{}
	for afterIndex, afterItem := range after.Items {
		beforeIndex := findMatchingItem(before.Items, afterItem, beforeMatched)
		if beforeIndex < 0 {
			continue
		}
		beforeMatched[beforeIndex] = true
		afterMatched[afterIndex] = true
		if beforeIndex != afterIndex {
			if used < maxDiffChanges {
				day, index := after.Day, afterIndex
				oldIndex := beforeIndex
				diff.ItemsMoved = append(diff.ItemsMoved, Change{
					Type:      "item_moved",
					DayNumber: &day,
					ItemIndex: &index,
					Before:    map[string]any{"itemIndex": oldIndex, "name": before.Items[beforeIndex].Name},
					After:     map[string]any{"itemIndex": afterIndex, "name": afterItem.Name},
				})
			}
			used++
		}
		if fieldChanges := itemFieldChanges(before.Items[beforeIndex], afterItem); len(fieldChanges) > 0 {
			if used < maxDiffChanges {
				day, index := after.Day, afterIndex
				diff.ItemsModified = append(diff.ItemsModified, Change{
					Type:         "item_modified",
					DayNumber:    &day,
					ItemIndex:    &index,
					FieldChanges: fieldChanges,
				})
			}
			used++
		}
	}
	for index, item := range after.Items {
		if afterMatched[index] {
			continue
		}
		if used < maxDiffChanges {
			day, itemIndex := after.Day, index
			diff.ItemsAdded = append(diff.ItemsAdded, Change{
				Type:      "item_added",
				DayNumber: &day,
				ItemIndex: &itemIndex,
				After:     compactItem(item),
			})
		}
		used++
	}
	for index, item := range before.Items {
		if beforeMatched[index] {
			continue
		}
		if used < maxDiffChanges {
			day, itemIndex := before.Day, index
			diff.ItemsRemoved = append(diff.ItemsRemoved, Change{
				Type:      "item_removed",
				DayNumber: &day,
				ItemIndex: &itemIndex,
				Before:    compactItem(item),
			})
		}
		used++
	}
	return used
}

func daysByNumber(it aggregate.Itinerary) map[int]aggregate.ItineraryDay {
	out := make(map[int]aggregate.ItineraryDay, len(it.Days))
	for _, day := range it.Days {
		out[day.Day] = day
	}
	return out
}

func findMatchingItem(items []aggregate.ItineraryItem, target aggregate.ItineraryItem, used map[int]bool) int {
	targetKey := itemIdentity(target)
	for index, item := range items {
		if used[index] {
			continue
		}
		if itemIdentity(item) == targetKey {
			return index
		}
	}
	return -1
}

func itemIdentity(item aggregate.ItineraryItem) string {
	name := strings.ToLower(strings.TrimSpace(item.Name))
	timeValue := strings.TrimSpace(item.Time)
	return fmt.Sprintf("%s|%s", name, timeValue)
}

func itemFieldChanges(before, after aggregate.ItineraryItem) []FieldChange {
	fields := []FieldChange{}
	appendChange := func(field string, oldValue, newValue any) {
		if reflect.DeepEqual(oldValue, newValue) {
			return
		}
		fields = append(fields, FieldChange{Field: field, Before: oldValue, After: newValue})
	}
	appendChange("time", before.Time, after.Time)
	appendChange("endTime", before.EndTime, after.EndTime)
	appendChange("type", before.Type, after.Type)
	appendChange("category", before.Category, after.Category)
	appendChange("transportMode", before.TransportMode, after.TransportMode)
	appendChange("durationMinutes", before.DurationMinutes, after.DurationMinutes)
	appendChange("walkingDistanceKm", before.WalkingDistanceKm, after.WalkingDistanceKm)
	appendChange("name", before.Name, after.Name)
	appendChange("note", before.Note, after.Note)
	appendChange("estimatedCost", before.EstimatedCost, after.EstimatedCost)
	return fields
}

func compactItem(item aggregate.ItineraryItem) map[string]any {
	return map[string]any{
		"time":          item.Time,
		"endTime":       item.EndTime,
		"type":          item.Type,
		"name":          item.Name,
		"estimatedCost": item.EstimatedCost,
	}
}
