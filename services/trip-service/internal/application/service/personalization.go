package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/personalization"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

type RecommendedTemplate struct {
	Template       appdto.TripTemplateWithAccess  `json:"template"`
	FitScore       int                            `json:"fitScore"`
	WhyThisFitsYou personalization.WhyThisFitsYou `json:"whyThisFitsYou"`
	FitTags        []string                       `json:"fitTags"`
}
type BudgetSuggestion struct {
	SuggestedRange struct {
		Min personalization.Money `json:"min"`
		Max personalization.Money `json:"max"`
	} `json:"suggestedRange"`
	Confidence          string                     `json:"confidence"`
	Reasons             []string                   `json:"reasons"`
	CategorySuggestions []BudgetCategorySuggestion `json:"categorySuggestions"`
}
type BudgetCategorySuggestion struct {
	Category     string                `json:"category"`
	AmountPerDay personalization.Money `json:"amountPerDay"`
	Reason       string                `json:"reason"`
}

func (s *Service) personalizationForPlanning(ctx context.Context, userID uuid.UUID, source personalization.Source, workspaceID *uuid.UUID, userCtx usercontext.UserContext, previous []entity.Trip, policy *workspacepolicies.Policy) (*personalization.PlanningSummary, error) {
	if s.personalization == nil {
		return nil, nil
	}
	value, err := s.personalization.Build(ctx, personalization.BuildInput{UserID: userID, WorkspaceID: workspaceID, Source: source, UserContext: userCtx, PreviousTrips: previous, WorkspacePolicy: policy})
	if err != nil {
		return nil, err
	}
	summary := value.PlanningSummary()
	return &summary, nil
}

// GetPersonalizationContext returns the caller's own context. A supplied trip
// is first permission checked and is used only to apply workspace policy.
func (s *Service) GetPersonalizationContext(ctx context.Context, source personalization.Source, tripID *uuid.UUID) (personalization.Context, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return personalization.Context{}, err
	}
	var trip *entity.Trip
	if tripID != nil {
		trip, _, err = s.requireViewerEditorOrOwner(ctx, *tripID, user.ID)
		if err != nil {
			return personalization.Context{}, err
		}
	}
	userCtx, err := s.loadUserContext(ctx, user, tripIDForContext(trip))
	if err != nil {
		return personalization.Context{}, err
	}
	previous, err := s.previousTripsForPlanningConstraints(ctx, user.ID, true)
	if err != nil {
		return personalization.Context{}, err
	}
	var workspaceID *uuid.UUID
	if trip != nil {
		workspaceID = trip.WorkspaceID
	}
	policy, err := s.activeWorkspacePolicy(ctx, workspaceID, true)
	if err != nil {
		return personalization.Context{}, err
	}
	if s.personalization == nil {
		return personalization.Context{SchemaVersion: personalization.SchemaVersion, UserID: user.ID, Source: source}, nil
	}
	return s.personalization.Build(ctx, personalization.BuildInput{UserID: user.ID, WorkspaceID: workspaceID, Source: source, UserContext: userCtx, PreviousTrips: previous, WorkspacePolicy: policy})
}

func (s *Service) SubmitPersonalizationFeedback(ctx context.Context, input personalization.SubmitFeedbackInput) (personalization.Feedback, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return personalization.Feedback{}, err
	}
	if input.TripID != nil {
		if _, _, err := s.requireViewerEditorOrOwner(ctx, *input.TripID, user.ID); err != nil {
			return personalization.Feedback{}, err
		}
	}
	if s.personalization == nil {
		return personalization.Feedback{}, fmt.Errorf("personalization feedback is not configured")
	}
	return s.personalization.Submit(ctx, user.ID, input)
}

func (s *Service) PersonalizationFeedbackSummary(ctx context.Context) (personalization.FeedbackSignals, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return personalization.FeedbackSignals{}, err
	}
	if s.personalization == nil {
		return personalization.FeedbackSignals{}, nil
	}
	return s.personalization.Summary(ctx, user.ID)
}

func (s *Service) ClearPersonalizationFeedback(ctx context.Context) error {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return err
	}
	if s.personalization == nil {
		return nil
	}
	return s.personalization.Clear(ctx, user.ID)
}

func (s *Service) RecommendedTemplates(ctx context.Context, workspaceID *uuid.UUID, limit int) ([]RecommendedTemplate, error) {
	if limit <= 0 || limit > 10 {
		limit = 10
	}
	contextValue, err := s.GetPersonalizationContext(ctx, personalization.SourceTemplateRanking, nil)
	if err != nil {
		return nil, err
	}
	items, _, _, err := s.ListTripTemplates(ctx, appdto.ListTripTemplatesInput{Limit: limit, WorkspaceID: workspaceID})
	if err != nil {
		return nil, err
	}
	result := make([]RecommendedTemplate, 0, len(items))
	for _, item := range items {
		template := item.Template
		score := 55
		reasons := []string{}
		tags := []string{}
		if overlap(template.Tags, contextValue.Preferences.TravelStyles) {
			score += 16
			reasons = append(reasons, "Matches your saved travel styles.")
			tags = append(tags, "style-match")
		}
		if overlap(template.Tags, contextValue.DerivedSignals.TransportBias) {
			score += 12
			reasons = append(reasons, "Fits your preferred transport style.")
			tags = append(tags, "transport-match")
		}
		if overlap(template.Tags, contextValue.DerivedSignals.ActivityBias) {
			score += 8
			reasons = append(reasons, "Includes activities you often choose.")
			tags = append(tags, "activity-match")
		}
		if contextValue.PastTripSignals.AverageTripDurationDays > 0 && abs(int(template.DurationDays)-contextValue.PastTripSignals.AverageTripDurationDays) <= 1 {
			score += 9
			reasons = append(reasons, "Fits your typical trip length.")
			tags = append(tags, "typical-duration")
		}
		if len(reasons) == 0 {
			reasons = append(reasons, "A reusable starting point you can still review and adapt.")
		}
		if score > 100 {
			score = 100
		}
		result = append(result, RecommendedTemplate{Template: item, FitScore: score, WhyThisFitsYou: personalization.WhyThisFitsYou{Score: score, Reasons: reasons, SignalsUsed: []string{"travelStyles", "preferredTransport", "pastTripSignals"}}, FitTags: tags})
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].FitScore > result[j].FitScore })
	if len(result) > 0 {
		personalization.RecordRecommendation(personalization.SourceTemplateRanking, contextValue.Completeness.Level, len(result))
	}
	return result, nil
}

func (s *Service) GetPersonalizedBudgetSuggestion(ctx context.Context, tripID uuid.UUID) (BudgetSuggestion, error) {
	trip, _, err := s.GetWithAccess(ctx, tripID)
	if err != nil {
		return BudgetSuggestion{}, err
	}
	contextValue, err := s.GetPersonalizationContext(ctx, personalization.SourceBudgetSuggestion, &tripID)
	if err != nil {
		return BudgetSuggestion{}, err
	}
	currency := contextValue.Profile.PreferredCurrency
	if strings.TrimSpace(trip.BudgetCurrency) != "" {
		currency = trip.BudgetCurrency
	}
	if currency == "" {
		currency = "EUR"
	}
	days := int(trip.Days)
	if days < 1 {
		days = 1
	}
	travelers := int(trip.Travelers)
	if travelers < 1 {
		travelers = 1
	}
	daily := 120.0
	if contextValue.PastTripSignals.AverageBudgetPerDay != nil && strings.EqualFold(contextValue.PastTripSignals.AverageBudgetPerDay.Currency, currency) {
		daily = contextValue.PastTripSignals.AverageBudgetPerDay.Amount
	}
	if trip.BudgetAmount != nil && *trip.BudgetAmount > 0 {
		daily = *trip.BudgetAmount / float64(days*travelers)
	}
	if contextValue.DerivedSignals.BudgetComfort == "low" {
		daily *= 0.88
	}
	if contextValue.WorkspacePolicy.MaxDailyBudget != nil && strings.EqualFold(contextValue.WorkspacePolicy.MaxDailyBudget.Currency, currency) && daily > contextValue.WorkspacePolicy.MaxDailyBudget.Amount {
		daily = contextValue.WorkspacePolicy.MaxDailyBudget.Amount
	}
	total := daily * float64(days*travelers)
	out := BudgetSuggestion{Confidence: "medium", Reasons: []string{}, CategorySuggestions: []BudgetCategorySuggestion{}}
	out.SuggestedRange.Min = personalization.Money{Amount: roundPersonalizationMoney(total * .9), Currency: currency}
	out.SuggestedRange.Max = personalization.Money{Amount: roundPersonalizationMoney(total * 1.1), Currency: currency}
	if contextValue.PastTripSignals.AverageBudgetPerDay != nil {
		out.Reasons = append(out.Reasons, "Similar past trips averaged "+fmt.Sprintf("%.0f", contextValue.PastTripSignals.AverageBudgetPerDay.Amount)+" "+currency+" per day.")
	}
	if len(contextValue.DerivedSignals.TransportBias) > 0 {
		out.Reasons = append(out.Reasons, "Uses your preference for "+strings.Join(contextValue.DerivedSignals.TransportBias, "/")+".")
	}
	if contextValue.WorkspacePolicy.MaxDailyBudget != nil {
		out.Reasons = append(out.Reasons, "Respects the workspace daily budget policy.")
	}
	if len(out.Reasons) == 0 {
		out.Reasons = append(out.Reasons, "Based on your trip duration, travelers, and saved preferences.")
	}
	food := daily * .3
	if overlap(contextValue.DerivedSignals.ActivityBias, []string{"food", "local_food", "markets"}) {
		food = daily * .35
	}
	out.CategorySuggestions = append(out.CategorySuggestions, BudgetCategorySuggestion{Category: "food", AmountPerDay: personalization.Money{Amount: roundPersonalizationMoney(food), Currency: currency}, Reason: "Adjusted for your food and activity preferences."})
	return out, nil
}

func overlap(left, right []string) bool {
	seen := map[string]struct{}{}
	for _, v := range left {
		seen[strings.ToLower(strings.TrimSpace(v))] = struct{}{}
	}
	for _, v := range right {
		if _, ok := seen[strings.ToLower(strings.TrimSpace(v))]; ok {
			return true
		}
	}
	return false
}
func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
func roundPersonalizationMoney(value float64) float64 { return float64(int(value*100+0.5)) / 100 }
