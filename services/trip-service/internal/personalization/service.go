package personalization

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

const maxFeedbackMetadataBytes = 4096

type Repository interface {
	Create(context.Context, Feedback) (Feedback, error)
	ListByUser(context.Context, uuid.UUID, int) ([]Feedback, error)
	ClearByUser(context.Context, uuid.UUID) error
}
type Service struct {
	repo Repository
	log  *zap.Logger
}

func New(repo Repository, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{repo: repo, log: log}
}

type BuildInput struct {
	UserID          uuid.UUID
	WorkspaceID     *uuid.UUID
	Source          Source
	UserContext     usercontext.UserContext
	PreviousTrips   []entity.Trip
	WorkspacePolicy *workspacepolicies.Policy
}

func (s *Service) Build(ctx context.Context, input BuildInput) (Context, error) {
	started := time.Now()
	defer func() {
		contextBuildDuration.WithLabelValues(string(input.Source)).Observe(time.Since(started).Seconds())
	}()
	feedback := []Feedback{}
	if s != nil && s.repo != nil {
		var err error
		feedback, err = s.repo.ListByUser(ctx, input.UserID, 100)
		if err != nil {
			return Context{}, fmt.Errorf("list personalization feedback: %w", err)
		}
	}
	profile := profileFrom(input.UserContext.Profile)
	prefs := preferencesFrom(input.UserContext.Preferences)
	derived := derive(prefs, feedback)
	past := pastTrips(input.PreviousTrips, profile.PreferredCurrency)
	feedbackSignals := summarizeFeedback(feedback)
	policy, warnings := policyFrom(input.WorkspacePolicy, profile.PreferredCurrency, &prefs, &derived)
	context := Context{SchemaVersion: SchemaVersion, UserID: input.UserID, WorkspaceID: cloneID(input.WorkspaceID), Source: input.Source, Profile: profile, Preferences: prefs, DerivedSignals: derived, PastTripSignals: past, FeedbackSignals: feedbackSignals, WorkspacePolicy: policy, Completeness: completeness(profile, prefs), Warnings: warnings, ExplanationInputs: explanations(prefs, derived, feedbackSignals)}
	contextBuiltTotal.WithLabelValues(string(input.Source), context.Completeness.Level).Inc()
	completenessScore.WithLabelValues(context.Completeness.Level).Set(float64(context.Completeness.Score))
	return context, nil
}

func (s *Service) Submit(ctx context.Context, userID uuid.UUID, input SubmitFeedbackInput) (Feedback, error) {
	if s == nil || s.repo == nil {
		return Feedback{}, fmt.Errorf("personalization feedback is not configured")
	}
	if !validEntityType(input.EntityType) {
		return Feedback{}, fmt.Errorf("invalid feedback entityType")
	}
	if !validFeedbackType(input.FeedbackType) {
		return Feedback{}, fmt.Errorf("invalid feedback feedbackType")
	}
	metadata, err := sanitizeMetadata(input.Metadata)
	if err != nil {
		return Feedback{}, err
	}
	created, err := s.repo.Create(ctx, Feedback{ID: uuid.New(), UserID: userID, WorkspaceID: cloneID(input.WorkspaceID), TripID: cloneID(input.TripID), EntityType: strings.TrimSpace(input.EntityType), EntityID: trim(input.EntityID, 160), FeedbackType: input.FeedbackType, FeedbackValue: trim(input.FeedbackValue, 300), Metadata: metadata})
	if err != nil {
		return Feedback{}, fmt.Errorf("create personalization feedback: %w", err)
	}
	s.log.Info("personalization feedback submitted", zap.String("feedback_type", string(input.FeedbackType)), zap.String("entity_type", input.EntityType))
	feedbackSubmittedTotal.WithLabelValues(string(input.FeedbackType)).Inc()
	return created, nil
}

func (s *Service) Summary(ctx context.Context, userID uuid.UUID) (FeedbackSignals, error) {
	if s == nil || s.repo == nil {
		return FeedbackSignals{}, nil
	}
	values, err := s.repo.ListByUser(ctx, userID, 100)
	if err != nil {
		return FeedbackSignals{}, err
	}
	return summarizeFeedback(values), nil
}
func (s *Service) Clear(ctx context.Context, userID uuid.UUID) error {
	if s == nil || s.repo == nil {
		return nil
	}
	return s.repo.ClearByUser(ctx, userID)
}

func validEntityType(v string) bool {
	switch strings.TrimSpace(v) {
	case "destination_suggestion", "route_alternative", "itinerary_item", "template", "budget_suggestion", "checklist_item", "general":
		return true
	}
	return false
}
func validFeedbackType(v FeedbackType) bool {
	switch v {
	case FeedbackLike, FeedbackDislike, FeedbackTooExpensive, FeedbackTooMuchWalking, FeedbackTooPacked, FeedbackNotMyVibe, FeedbackMoreNature, FeedbackMoreFood, FeedbackLessMuseums, FeedbackPreferTrains, FeedbackAvoidNightlife, FeedbackPreferRelaxed, FeedbackPreferFastPaced, FeedbackTooFar, FeedbackTooManyTransfers, FeedbackOther:
		return true
	}
	return false
}
func sanitizeMetadata(input map[string]any) (map[string]any, error) {
	if len(input) == 0 {
		return map[string]any{}, nil
	}
	allowed := map[string]struct{}{"source": {}, "destination": {}, "style": {}, "transport": {}, "currency": {}, "category": {}}
	result := map[string]any{}
	for key, raw := range input {
		if _, ok := allowed[key]; !ok {
			continue
		}
		switch value := raw.(type) {
		case string:
			if cleaned := trim(value, 160); cleaned != "" {
				result[key] = cleaned
			}
		case []any:
			values := make([]string, 0, len(value))
			for _, item := range value {
				if text, ok := item.(string); ok && trim(text, 80) != "" {
					values = append(values, trim(text, 80))
					if len(values) == 10 {
						break
					}
				}
			}
			if len(values) > 0 {
				result[key] = values
			}
		}
	}
	raw, _ := json.Marshal(result)
	if len(raw) > maxFeedbackMetadataBytes {
		return nil, fmt.Errorf("feedback metadata is too large")
	}
	return result, nil
}

func profileFrom(p *usercontext.UserProfile) Profile {
	if p == nil {
		return Profile{PreferredCurrency: "EUR", PreferredLanguage: "en"}
	}
	return Profile{HomeCity: value(p.HomeCity), HomeCountry: value(p.HomeCountry), PreferredCurrency: fallback(p.PreferredCurrency, "EUR"), PreferredLanguage: fallback(p.PreferredLanguage, "en")}
}
func preferencesFrom(p *usercontext.UserPreferences) Preferences {
	if p == nil {
		return Preferences{TravelStyles: []string{}, FoodPreferences: []string{}, DietaryRestrictions: []string{}, Avoid: []string{}, PreferredTransport: []string{}, AccommodationStyle: []string{}}
	}
	return Preferences{TravelStyles: clean(p.TravelStyles), Pace: fallback(p.Pace, "balanced"), MaxWalkingKmPerDay: p.MaxWalkingKmPerDay, FoodPreferences: clean(p.FoodPreferences), DietaryRestrictions: clean(p.DietaryRestrictions), Avoid: clean(p.Avoid), PreferredTransport: clean(p.PreferredTransport), AccommodationStyle: clean(p.AccommodationStyle)}
}
func derive(p Preferences, feedback []Feedback) DerivedSignals {
	budget := "medium"
	walking := "moderate"
	if p.MaxWalkingKmPerDay != nil {
		if *p.MaxWalkingKmPerDay <= 5 {
			walking = "low"
		} else if *p.MaxWalkingKmPerDay >= 12 {
			walking = "high"
		}
	}
	summary := summarizeFeedback(feedback)
	if summary.TooExpensiveCount >= 2 {
		budget = "low"
	}
	if summary.TooMuchWalkingCount >= 2 {
		walking = "low"
	}
	transport := clean(append(append([]string{}, p.PreferredTransport...), transportFeedback(feedback)...))
	activities := clean(append(append([]string{}, p.TravelStyles...), p.FoodPreferences...))
	if summary.PreferTrainCount > 0 {
		transport = clean(append(transport, "train"))
	}
	return DerivedSignals{BudgetComfort: budget, WalkingTolerance: walking, NoveltyPreference: "medium", TransportBias: transport, ActivityBias: activities, AvoidBias: clean(append(append([]string{}, p.Avoid...), avoidFeedback(feedback)...)), PlanningStyle: fallback(p.Pace, "balanced")}
}
func pastTrips(trips []entity.Trip, currency string) PastTripSignals {
	out := PastTripSignals{RecentDestinations: []string{}, RepeatedStyles: []string{}, PreferredTransportFromHistory: []string{}}
	var days int
	var total float64
	var budgetDays int
	styles := []string{}
	destinations := map[string]int{}
	for _, trip := range trips {
		if strings.TrimSpace(trip.Destination) != "" {
			out.PastDestinationCount++
			if len(out.RecentDestinations) < 5 {
				out.RecentDestinations = append(out.RecentDestinations, trip.Destination)
			}
			destinations[strings.ToLower(strings.TrimSpace(trip.Destination))]++
		}
		if trip.Days > 0 {
			days += int(trip.Days)
		}
		styles = append(styles, trip.Interests...)
		if trip.BudgetAmount != nil && trip.Days > 0 && (trip.BudgetCurrency == "" || strings.EqualFold(trip.BudgetCurrency, currency)) {
			total += *trip.BudgetAmount
			budgetDays += int(trip.Days)
		}
	}
	if len(trips) > 0 {
		out.AverageTripDurationDays = days / len(trips)
	}
	if budgetDays > 0 {
		out.AverageBudgetPerDay = &Money{Amount: total / float64(budgetDays), Currency: currency}
	}
	out.RepeatedStyles = clean(styles)
	return out
}
func summarizeFeedback(items []Feedback) FeedbackSignals {
	out := FeedbackSignals{LikedDestinations: []string{}, DislikedDestinations: []string{}, LikedStyles: []string{}, DislikedStyles: []string{}, BudgetSensitivity: "medium", WalkingSensitivity: "moderate"}
	for _, item := range items {
		out.RecentFeedbackCount++
		destination, _ := item.Metadata["destination"].(string)
		style, _ := item.Metadata["style"].(string)
		switch item.FeedbackType {
		case FeedbackLike:
			out.LikedDestinations = append(out.LikedDestinations, destination)
			out.LikedStyles = append(out.LikedStyles, style)
		case FeedbackDislike, FeedbackNotMyVibe:
			out.DislikedDestinations = append(out.DislikedDestinations, destination)
			out.DislikedStyles = append(out.DislikedStyles, style)
		case FeedbackMoreFood:
			out.LikedStyles = append(out.LikedStyles, "food")
		case FeedbackMoreNature:
			out.LikedStyles = append(out.LikedStyles, "nature")
		case FeedbackAvoidNightlife:
			out.DislikedStyles = append(out.DislikedStyles, "nightlife")
		case FeedbackTooExpensive:
			out.TooExpensiveCount++
		case FeedbackTooMuchWalking:
			out.TooMuchWalkingCount++
		case FeedbackPreferTrains:
			out.PreferTrainCount++
		}
	}
	out.LikedDestinations = clean(out.LikedDestinations)
	out.DislikedDestinations = clean(out.DislikedDestinations)
	out.LikedStyles = clean(out.LikedStyles)
	out.DislikedStyles = clean(out.DislikedStyles)
	if out.TooExpensiveCount >= 2 {
		out.BudgetSensitivity = "high"
	}
	if out.TooMuchWalkingCount >= 2 {
		out.WalkingSensitivity = "high"
	}
	return out
}
func policyFrom(policy *workspacepolicies.Policy, currency string, prefs *Preferences, derived *DerivedSignals) (WorkspacePolicy, []string) {
	out := WorkspacePolicy{DisallowedTransportModes: []string{}}
	warnings := []string{}
	if policy == nil {
		return out, warnings
	}
	out.Enabled = true
	out.PreferredCurrency = currency
	rules := policy.Rules.Rules
	if rules.MaxDailyBudget.Enabled {
		out.MaxDailyBudget = &Money{Amount: rules.MaxDailyBudget.Amount, Currency: rules.MaxDailyBudget.Currency}
		if out.MaxDailyBudget.Currency == "" {
			out.MaxDailyBudget.Currency = currency
		}
		out.BlockingRuleCount++
	}
	if rules.DisallowedTransportModes.Enabled {
		out.DisallowedTransportModes = clean(rules.DisallowedTransportModes.Modes)
		out.BlockingRuleCount++
		disallowed := map[string]struct{}{}
		for _, mode := range out.DisallowedTransportModes {
			disallowed[strings.ToLower(mode)] = struct{}{}
		}
		kept := []string{}
		for _, mode := range prefs.PreferredTransport {
			if _, blocked := disallowed[strings.ToLower(mode)]; blocked {
				warnings = append(warnings, "Personal transport preference "+mode+" ignored because workspace policy disallows it.")
				continue
			}
			kept = append(kept, mode)
		}
		prefs.PreferredTransport = kept
		derived.TransportBias = clean(append([]string{}, kept...))
	}
	return out, warnings
}
func completeness(p Profile, pref Preferences) Completeness {
	out := Completeness{MissingFields: []MissingField{}, RecommendedActions: []RecommendedAction{}}
	add := func(ok bool, score int, field, label, reason string) {
		if ok {
			out.Score += score
		} else {
			out.MissingFields = append(out.MissingFields, MissingField{field, label, reason})
		}
	}
	add(p.HomeCity != "" || p.HomeCountry != "", 10, "homeLocation", "Home city or country", "Helps us suggest practical starting points.")
	add(p.PreferredCurrency != "", 10, "preferredCurrency", "Preferred currency", "Helps us compare costs.")
	add(p.PreferredLanguage != "", 10, "preferredLanguage", "Preferred language", "Helps us localize suggestions.")
	add(len(pref.TravelStyles) > 0, 15, "travelStyles", "Travel styles", "Helps us match trip ideas.")
	add(pref.Pace != "", 10, "pace", "Trip pace", "Helps us balance activity density.")
	add(pref.MaxWalkingKmPerDay != nil, 10, "maxWalkingKmPerDay", "Walking tolerance", "Helps us keep travel comfortable.")
	add(len(pref.PreferredTransport) > 0, 15, "preferredTransport", "Preferred transport", "Helps us suggest better routes.")
	add(len(pref.FoodPreferences) > 0 || len(pref.DietaryRestrictions) > 0, 10, "foodAndDietaryPreferences", "Food and dietary preferences", "Helps us personalize food options.")
	add(len(pref.AccommodationStyle) > 0, 5, "accommodationStyle", "Accommodation style", "Helps us recommend stays.")
	add(len(pref.Avoid) > 0, 5, "avoid", "Avoid list", "Helps us avoid unsuitable activities.")
	if out.Score >= 90 {
		out.Level = "excellent"
	} else if out.Score >= 70 {
		out.Level = "good"
	} else if out.Score >= 40 {
		out.Level = "partial"
	} else {
		out.Level = "poor"
	}
	if len(out.MissingFields) > 0 {
		out.RecommendedActions = []RecommendedAction{{Label: "Review travel preferences", Href: "/settings?section=preferences"}}
	}
	return out
}
func explanations(p Preferences, d DerivedSignals, f FeedbackSignals) []string {
	values := []string{}
	if len(d.TransportBias) > 0 {
		values = append(values, "prefers "+strings.Join(d.TransportBias, "/"))
	}
	if len(d.ActivityBias) > 0 {
		values = append(values, "usually chooses "+strings.Join(d.ActivityBias, " and ")+" trips")
	}
	if d.WalkingTolerance != "" {
		values = append(values, "walking tolerance is "+d.WalkingTolerance)
	}
	if f.TooExpensiveCount > 0 {
		values = append(values, "has flagged expensive suggestions")
	}
	return values
}
func transportFeedback(items []Feedback) []string {
	out := []string{}
	for _, i := range items {
		if i.FeedbackType == FeedbackPreferTrains {
			out = append(out, "train")
		}
	}
	return out
}
func avoidFeedback(items []Feedback) []string {
	out := []string{}
	for _, i := range items {
		if i.FeedbackType == FeedbackAvoidNightlife {
			out = append(out, "nightlife")
		}
	}
	return out
}
func clean(values []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, raw := range values {
		v := trim(raw, 100)
		if v == "" {
			continue
		}
		key := strings.ToLower(v)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
func trim(v string, max int) string {
	v = strings.TrimSpace(v)
	if len(v) > max {
		return v[:max]
	}
	return v
}
func value(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}
func fallback(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return strings.TrimSpace(v)
}
func cloneID(v *uuid.UUID) *uuid.UUID {
	if v == nil {
		return nil
	}
	x := *v
	return &x
}
