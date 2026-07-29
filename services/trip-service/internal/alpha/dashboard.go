package alpha

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

var trackedFeatures = []string{
	"authentication",
	"profile",
	"trips",
	"ai",
	"budget",
	"routes",
	"sharing",
	"notifications",
	"feedback",
	"settings",
	"onboarding",
}

type dashboardWindow struct {
	Since30 time.Time
	Since7  time.Time
	Since1  time.Time
}

func (s *Service) Dashboard(ctx context.Context) (*Dashboard, error) {
	now := s.now()
	window := dashboardWindow{
		Since30: now.AddDate(0, 0, -30),
		Since7:  now.AddDate(0, 0, -7),
		Since1:  now.AddDate(0, 0, -1),
	}
	users, err := s.userMetrics(ctx, window)
	if err != nil {
		return nil, err
	}
	trips, err := s.tripMetrics(ctx, window)
	if err != nil {
		return nil, err
	}
	ai, err := s.aiMetrics(ctx, window)
	if err != nil {
		return nil, err
	}
	feedback, err := s.feedbackMetrics(ctx, window)
	if err != nil {
		return nil, err
	}
	usage, err := s.usageMetrics(ctx, window)
	if err != nil {
		return nil, err
	}
	health, err := s.healthMetrics(ctx, window)
	if err != nil {
		return nil, err
	}
	funnel, err := s.funnel(ctx, window)
	if err != nil {
		return nil, err
	}
	features, err := s.featureUsage(ctx, window)
	if err != nil {
		return nil, err
	}
	costs := CostMetrics{
		OpenAITokens:        ai.TokenUsage,
		EstimatedOpenAICost: estimateOpenAICost(ai.TokenUsage),
	}
	dashboard := &Dashboard{
		GeneratedAt:  now,
		Users:        users,
		Trips:        trips,
		AI:           ai,
		Feedback:     feedback,
		Usage:        usage,
		Costs:        costs,
		Health:       health,
		Funnel:       funnel,
		FeatureUsage: features,
	}
	dashboard.Alerts = deriveAlerts(*dashboard)
	alphaActiveUsers.Set(float64(users.Active))
	alphaRetentionRate.Set(ratio(users.Retained, max(users.Invited, 1)))
	alphaOpenAICostEstimate.Set(costs.EstimatedOpenAICost)
	if ai.AverageLatencyMS > 0 {
		alphaGenerationLatency.Observe(float64(ai.AverageLatencyMS) / 1000)
	}
	return dashboard, nil
}

func (s *Service) userMetrics(ctx context.Context, window dashboardWindow) (UserMetrics, error) {
	invited, err := s.count(ctx, `SELECT COUNT(*) FROM alpha_invite_activations`)
	if err != nil {
		return UserMetrics{}, err
	}
	active, err := s.count(ctx, `SELECT COUNT(*) FROM alpha_participants WHERE active=true AND last_activity_at >= $1`, window.Since7)
	if err != nil {
		return UserMetrics{}, err
	}
	inactive, err := s.count(ctx, `SELECT COUNT(*) FROM alpha_participants WHERE active=true AND (last_activity_at IS NULL OR last_activity_at < $1)`, window.Since7)
	if err != nil {
		return UserMetrics{}, err
	}
	retained, err := s.count(ctx, `SELECT COUNT(*) FROM alpha_participants WHERE first_login_at IS NOT NULL AND last_activity_at >= first_login_at + interval '1 day'`)
	if err != nil {
		return UserMetrics{}, err
	}
	return UserMetrics{Invited: invited, Active: active, Inactive: inactive, Retained: retained}, nil
}

func (s *Service) tripMetrics(ctx context.Context, window dashboardWindow) (TripMetrics, error) {
	created, err := s.count(ctx, `SELECT COUNT(*) FROM trips t JOIN alpha_participants p ON p.user_id=t.user_id WHERE t.created_at >= $1`, window.Since30)
	if err != nil {
		return TripMetrics{}, err
	}
	completed, err := s.count(ctx, `SELECT COUNT(*) FROM trips t JOIN alpha_participants p ON p.user_id=t.user_id WHERE t.status='COMPLETED' AND t.created_at >= $1`, window.Since30)
	if err != nil {
		return TripMetrics{}, err
	}
	return TripMetrics{Created: created, Completed: completed}, nil
}

func (s *Service) aiMetrics(ctx context.Context, window dashboardWindow) (AIMetrics, error) {
	total, err := s.count(ctx, `SELECT COUNT(*) FROM ai_generation_traces t JOIN alpha_participants p ON p.user_id=t.user_id WHERE t.created_at >= $1`, window.Since30)
	if err != nil {
		return AIMetrics{}, err
	}
	success, err := s.count(ctx, `SELECT COUNT(*) FROM ai_generation_traces t JOIN alpha_participants p ON p.user_id=t.user_id WHERE t.created_at >= $1 AND t.status IN ('completed','completed_with_warnings')`, window.Since30)
	if err != nil {
		return AIMetrics{}, err
	}
	repairs, err := s.count(ctx, `SELECT COUNT(*) FROM ai_generation_traces t JOIN alpha_participants p ON p.user_id=t.user_id WHERE t.created_at >= $1 AND (t.repair_duration_ms IS NOT NULL OR t.repair_summary_json IS NOT NULL)`, window.Since30)
	if err != nil {
		return AIMetrics{}, err
	}
	fallbacks, err := s.count(ctx, `SELECT COUNT(*) FROM ai_generation_traces t JOIN alpha_participants p ON p.user_id=t.user_id WHERE t.created_at >= $1 AND (t.generation_summary_json @> '{"fallbackUsed": true}'::jsonb OR t.output_summary_json @> '{"fallbackUsed": true}'::jsonb)`, window.Since30)
	if err != nil {
		return AIMetrics{}, err
	}
	avgLatency, err := s.count(ctx, `SELECT COALESCE(AVG(duration_ms),0)::int FROM ai_generation_traces t JOIN alpha_participants p ON p.user_id=t.user_id WHERE t.created_at >= $1 AND duration_ms IS NOT NULL`, window.Since30)
	if err != nil {
		return AIMetrics{}, err
	}
	tokens, err := s.count(ctx, `SELECT COALESCE(SUM(token_total_estimate),0)::int FROM ai_generation_traces t JOIN alpha_participants p ON p.user_id=t.user_id WHERE t.created_at >= $1`, window.Since30)
	if err != nil {
		return AIMetrics{}, err
	}
	regenerated, err := s.eventCount(ctx, "itinerary_regenerated", window.Since30)
	if err != nil {
		return AIMetrics{}, err
	}
	removed, err := s.eventCount(ctx, "place_removed", window.Since30)
	if err != nil {
		return AIMetrics{}, err
	}
	replaced, err := s.eventCount(ctx, "place_replaced", window.Since30)
	if err != nil {
		return AIMetrics{}, err
	}
	accepted, err := s.eventCount(ctx, "itinerary_accepted", window.Since30)
	if err != nil {
		return AIMetrics{}, err
	}
	badPlaces, err := s.count(ctx, `SELECT COUNT(*) FROM alpha_feedback WHERE created_at >= $1 AND category='ai' AND metadata ->> 'issueType' = 'bad_place'`, window.Since30)
	if err != nil {
		return AIMetrics{}, err
	}
	return AIMetrics{
		Generations:      total,
		SuccessRate:      ratio(success, total),
		RepairRate:       ratio(repairs, total),
		FallbackRate:     ratio(fallbacks, total),
		AverageLatencyMS: avgLatency,
		TokenUsage:       tokens,
		Regenerated:      regenerated,
		RemovedPlaces:    removed,
		ReplacedPlaces:   replaced,
		Accepted:         accepted,
		BadPlaceReports:  badPlaces,
	}, nil
}

func (s *Service) feedbackMetrics(ctx context.Context, window dashboardWindow) (FeedbackMetrics, error) {
	total, err := s.count(ctx, `SELECT COUNT(*) FROM alpha_feedback WHERE created_at >= $1`, window.Since30)
	if err != nil {
		return FeedbackMetrics{}, err
	}
	byCategory, err := s.countBy(ctx, `SELECT category, COUNT(*) FROM alpha_feedback WHERE created_at >= $1 GROUP BY category`, window.Since30)
	if err != nil {
		return FeedbackMetrics{}, err
	}
	byStatus, err := s.countBy(ctx, `SELECT status, COUNT(*) FROM alpha_feedback WHERE created_at >= $1 GROUP BY status`, window.Since30)
	if err != nil {
		return FeedbackMetrics{}, err
	}
	return FeedbackMetrics{
		Total:           total,
		BugReports:      byCategory["bug"],
		AIReports:       byCategory["ai"],
		FeatureRequests: byCategory["feature_request"],
		ByCategory:      byCategory,
		ByStatus:        byStatus,
	}, nil
}

func (s *Service) usageMetrics(ctx context.Context, window dashboardWindow) (UsageMetrics, error) {
	dau, err := s.distinctUsers(ctx, window.Since1)
	if err != nil {
		return UsageMetrics{}, err
	}
	wau, err := s.distinctUsers(ctx, window.Since7)
	if err != nil {
		return UsageMetrics{}, err
	}
	mau, err := s.distinctUsers(ctx, window.Since30)
	if err != nil {
		return UsageMetrics{}, err
	}
	return UsageMetrics{DAU: dau, WAU: wau, MAU: mau}, nil
}

func (s *Service) healthMetrics(ctx context.Context, window dashboardWindow) (HealthMetrics, error) {
	aiFailures, err := s.count(ctx, `SELECT COUNT(*) FROM ai_generation_traces t JOIN alpha_participants p ON p.user_id=t.user_id WHERE t.created_at >= $1 AND t.status='failed'`, window.Since30)
	if err != nil {
		return HealthMetrics{}, err
	}
	productErrors, err := s.eventCount(ctx, "error_occurred", window.Since30)
	if err != nil {
		return HealthMetrics{}, err
	}
	retries, err := s.count(ctx, `SELECT COUNT(*) FROM trip_generation_jobs j JOIN alpha_participants p ON p.user_id=j.requested_by_user_id WHERE j.created_at >= $1 AND j.retried_from_job_id IS NOT NULL`, window.Since30)
	if err != nil {
		return HealthMetrics{}, err
	}
	return HealthMetrics{
		Failures:  aiFailures + productErrors,
		Retries:   retries,
		Incidents: aiFailures + productErrors,
	}, nil
}

func (s *Service) funnel(ctx context.Context, window dashboardWindow) ([]FunnelStage, error) {
	stages := []struct {
		name  string
		event string
	}{
		{"Registration", "signup_completed"},
		{"Profile completed", "profile_completed"},
		{"Trip created", "trip_created"},
		{"AI itinerary generated", "itinerary_generated"},
		{"Trip reviewed", "trip_reviewed"},
		{"Trip shared", "share_created"},
		{"User returned", "user_returned"},
		{"Second trip created", "second_trip_created"},
	}
	out := make([]FunnelStage, 0, len(stages))
	previous := 0
	for index, stage := range stages {
		count, err := s.count(ctx, `SELECT COUNT(DISTINCT user_id) FROM product_analytics_events WHERE user_id IS NOT NULL AND occurred_at >= $1 AND event_name=$2`, window.Since30, stage.event)
		if err != nil {
			return nil, err
		}
		conversion := 1.0
		dropoff := 0
		if index > 0 {
			conversion = ratio(count, previous)
			dropoff = max(previous-count, 0)
		}
		out = append(out, FunnelStage{Name: stage.name, Users: count, Conversion: conversion, DropoffFromPrevious: dropoff})
		previous = count
	}
	return out, nil
}

func (s *Service) featureUsage(ctx context.Context, window dashboardWindow) ([]FeatureMetric, error) {
	rows, err := s.db.Query(ctx,
		`SELECT feature, COUNT(*), COUNT(DISTINCT user_id), MIN(occurred_at), GREATEST(COUNT(*) - COUNT(DISTINCT user_id), 0)::int
		 FROM product_analytics_events
		 WHERE occurred_at >= $1
		 GROUP BY feature`,
		window.Since30,
	)
	if err != nil {
		return nil, fmt.Errorf("feature usage: %w", err)
	}
	defer rows.Close()
	byFeature := map[string]FeatureMetric{}
	for rows.Next() {
		var metric FeatureMetric
		var firstUse time.Time
		if err := rows.Scan(&metric.Feature, &metric.UsageCount, &metric.UniqueUsers, &firstUse, &metric.RepeatUse); err != nil {
			return nil, fmt.Errorf("scan feature usage: %w", err)
		}
		metric.FirstUse = &firstUse
		byFeature[metric.Feature] = metric
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]FeatureMetric, 0, len(trackedFeatures))
	for _, feature := range trackedFeatures {
		metric := byFeature[feature]
		if metric.Feature == "" {
			metric.Feature = feature
			metric.Unused = true
		}
		out = append(out, metric)
	}
	return out, nil
}

func (s *Service) ListWeeklyReports(ctx context.Context, limit, offset int) ([]WeeklyReport, error) {
	limit, offset = normalizeLimitOffset(limit, offset, 100)
	rows, err := s.db.Query(ctx, `SELECT `+weeklyReportColumns+` FROM weekly_alpha_reports ORDER BY generated_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list weekly alpha reports: %w", err)
	}
	defer rows.Close()
	reports := []WeeklyReport{}
	for rows.Next() {
		report, err := scanWeeklyReport(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, *report)
	}
	return reports, rows.Err()
}

func (s *Service) GenerateWeeklyReport(ctx context.Context, weekStart time.Time, generatedBy *uuid.UUID) (*WeeklyReport, error) {
	if weekStart.IsZero() {
		now := s.now()
		offset := (int(now.Weekday()) + 6) % 7
		weekStart = time.Date(now.Year(), now.Month(), now.Day()-offset-7, 0, 0, 0, 0, time.UTC)
	}
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, time.UTC)
	weekEnd := weekStart.AddDate(0, 0, 7)
	dashboard, err := s.Dashboard(ctx)
	if err != nil {
		return nil, err
	}
	topBugs, _ := s.topFeedbackTitles(ctx, "bug", weekStart, weekEnd)
	topRequests, _ := s.topFeedbackTitles(ctx, "feature_request", weekStart, weekEnd)
	removedPlaces, _ := s.topEventMetadata(ctx, "place_removed", "placeName", weekStart, weekEnd)
	popularDestinations, _ := s.topTripDestinations(ctx, weekStart, weekEnd)
	markdown := buildWeeklyMarkdown(weekStart, weekEnd, *dashboard, topBugs, topRequests, removedPlaces, popularDestinations)
	metrics, err := json.Marshal(dashboard)
	if err != nil {
		return nil, err
	}
	reportID := uuid.New()
	return scanWeeklyReport(s.db.QueryRow(ctx,
		`INSERT INTO weekly_alpha_reports (id, week_start, week_end, summary_markdown, metrics, generated_by_user_id)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (week_start, week_end) DO UPDATE SET
		   summary_markdown=EXCLUDED.summary_markdown,
		   metrics=EXCLUDED.metrics,
		   generated_by_user_id=EXCLUDED.generated_by_user_id,
		   generated_at=NOW()
		 RETURNING `+weeklyReportColumns,
		idArg(reportID),
		weekStart,
		weekEnd,
		markdown,
		metrics,
		nullableIDArg(generatedBy),
	))
}

func (s *Service) count(ctx context.Context, query string, args ...any) (int, error) {
	var count int
	if err := s.db.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("alpha count query: %w", err)
	}
	return count, nil
}

func (s *Service) eventCount(ctx context.Context, event string, since time.Time) (int, error) {
	return s.count(ctx, `SELECT COUNT(*) FROM product_analytics_events WHERE event_name=$1 AND occurred_at >= $2`, event, since)
}

func (s *Service) distinctUsers(ctx context.Context, since time.Time) (int, error) {
	return s.count(ctx, `SELECT COUNT(DISTINCT user_id) FROM product_analytics_events WHERE user_id IS NOT NULL AND occurred_at >= $1`, since)
}

func (s *Service) countBy(ctx context.Context, query string, args ...any) (map[string]int, error) {
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("alpha grouped count query: %w", err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			return nil, fmt.Errorf("scan alpha grouped count: %w", err)
		}
		out[key] = count
	}
	return out, rows.Err()
}

func (s *Service) topFeedbackTitles(ctx context.Context, category string, from, to time.Time) ([]string, error) {
	rows, err := s.db.Query(ctx,
		`SELECT title
		 FROM alpha_feedback
		 WHERE category=$1 AND created_at >= $2 AND created_at < $3
		 ORDER BY priority DESC, created_at DESC
		 LIMIT 5`,
		category,
		from,
		to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStringList(rows)
}

func (s *Service) topEventMetadata(ctx context.Context, eventName, key string, from, to time.Time) ([]string, error) {
	rows, err := s.db.Query(ctx,
		`SELECT metadata ->> $2 AS value
		 FROM product_analytics_events
		 WHERE event_name=$1 AND occurred_at >= $3 AND occurred_at < $4 AND metadata ? $2
		 GROUP BY value
		 ORDER BY COUNT(*) DESC, value ASC
		 LIMIT 5`,
		eventName,
		key,
		from,
		to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStringList(rows)
}

func (s *Service) topTripDestinations(ctx context.Context, from, to time.Time) ([]string, error) {
	rows, err := s.db.Query(ctx,
		`SELECT destination
		 FROM trips t JOIN alpha_participants p ON p.user_id=t.user_id
		 WHERE t.created_at >= $1 AND t.created_at < $2
		 GROUP BY destination
		 ORDER BY COUNT(*) DESC, destination ASC
		 LIMIT 5`,
		from,
		to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStringList(rows)
}

func scanStringList(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]string, error) {
	out := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out, rows.Err()
}

func deriveAlerts(d Dashboard) []AlphaAlert {
	alerts := []AlphaAlert{}
	if d.AI.Generations >= 5 && 1-d.AI.SuccessRate > 0.2 {
		alerts = append(alerts, AlphaAlert{Type: "ai_failure_rate_spike", Severity: "warning", Message: "AI failure rate is above 20%.", Value: 1 - d.AI.SuccessRate})
	}
	if d.AI.FallbackRate > 0.2 && d.AI.Generations >= 5 {
		alerts = append(alerts, AlphaAlert{Type: "fallback_rate_spike", Severity: "warning", Message: "AI fallback rate is above 20%.", Value: d.AI.FallbackRate})
	}
	if d.Feedback.BugReports >= 5 {
		alerts = append(alerts, AlphaAlert{Type: "bug_reports_spike", Severity: "warning", Message: "Bug report volume is elevated.", Value: float64(d.Feedback.BugReports)})
	}
	if d.Costs.EstimatedOpenAICost >= 25 {
		alerts = append(alerts, AlphaAlert{Type: "openai_cost_spike", Severity: "warning", Message: "Estimated OpenAI cost crossed the alpha review threshold.", Value: d.Costs.EstimatedOpenAICost})
	}
	if d.Users.Invited >= 5 && ratio(d.Users.Retained, d.Users.Invited) < 0.25 {
		alerts = append(alerts, AlphaAlert{Type: "retention_drop", Severity: "warning", Message: "Retention is below 25%.", Value: ratio(d.Users.Retained, d.Users.Invited)})
	}
	if d.AI.AverageLatencyMS > 30000 {
		alerts = append(alerts, AlphaAlert{Type: "generation_latency_increase", Severity: "warning", Message: "Average generation latency is above 30 seconds.", Value: float64(d.AI.AverageLatencyMS)})
	}
	return alerts
}

func buildWeeklyMarkdown(weekStart, weekEnd time.Time, d Dashboard, topBugs, topRequests, removedPlaces, popularDestinations []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Weekly Alpha Report\n\n")
	fmt.Fprintf(&b, "Period: %s to %s\n\n", weekStart.Format("2006-01-02"), weekEnd.AddDate(0, 0, -1).Format("2006-01-02"))
	fmt.Fprintf(&b, "## Summary\n\n")
	fmt.Fprintf(&b, "- Invited users: %d\n", d.Users.Invited)
	fmt.Fprintf(&b, "- Active users: %d\n", d.Users.Active)
	fmt.Fprintf(&b, "- Retained users: %d\n", d.Users.Retained)
	fmt.Fprintf(&b, "- Trips created: %d\n", d.Trips.Created)
	fmt.Fprintf(&b, "- AI generations: %d, success rate: %.0f%%, repair rate: %.0f%%, fallback rate: %.0f%%\n", d.AI.Generations, d.AI.SuccessRate*100, d.AI.RepairRate*100, d.AI.FallbackRate*100)
	fmt.Fprintf(&b, "- Average generation latency: %d ms\n", d.AI.AverageLatencyMS)
	fmt.Fprintf(&b, "- Estimated OpenAI cost: $%.2f\n\n", d.Costs.EstimatedOpenAICost)
	appendList(&b, "Top bugs", topBugs)
	appendList(&b, "Top feature requests", topRequests)
	appendList(&b, "Most removed AI places", removedPlaces)
	appendList(&b, "Most popular destinations", popularDestinations)
	fmt.Fprintf(&b, "## Retention\n\n")
	fmt.Fprintf(&b, "- DAU: %d\n- WAU: %d\n- MAU: %d\n- Retention rate: %.0f%%\n\n", d.Usage.DAU, d.Usage.WAU, d.Usage.MAU, ratio(d.Users.Retained, max(d.Users.Invited, 1))*100)
	fmt.Fprintf(&b, "## Recommendations\n\n")
	if len(d.Alerts) == 0 {
		fmt.Fprintf(&b, "- No blocking alpha health alerts. Continue inviting in small batches and watch funnel drop-offs.\n")
	} else {
		for _, alert := range d.Alerts {
			fmt.Fprintf(&b, "- %s: %s\n", alert.Type, alert.Message)
		}
	}
	return b.String()
}

func appendList(b *strings.Builder, title string, values []string) {
	fmt.Fprintf(b, "## %s\n\n", title)
	if len(values) == 0 {
		fmt.Fprintf(b, "- None recorded.\n\n")
		return
	}
	for _, value := range values {
		fmt.Fprintf(b, "- %s\n", value)
	}
	fmt.Fprintln(b)
}

func estimateOpenAICost(tokens int) float64 {
	// Conservative v1 estimate until provider-specific pricing is configured.
	return math.Round(float64(tokens)*0.0000005*100) / 100
}

func ratio(numerator, denominator int) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}
