# Product Analytics

Product analytics v1 is a privacy-safe event system in Trip Service. It is for
closed-alpha learning and launch safety, not user profiling.

## Event Contract

Authenticated clients submit events to `POST /analytics/events`.

Core fields:

- `eventName`: allowlisted event name.
- `feature`: optional feature override; invalid values fall back to the event's
  default feature.
- `entityType`: coarse entity type such as `trip`, `generation_job`, or
  `feedback`.
- `entityId`: stored as a hash, never raw.
- `metadata`: sanitized JSON object.
- `sessionId`: stored as a hash.
- `appVersion`, browser, OS, device type, request ID, correlation ID.

Supported feature groups are `authentication`, `profile`, `trips`, `ai`,
`budget`, `routes`, `sharing`, `notifications`, `feedback`, `settings`,
`retention`, `health`, and `onboarding`.

## Tracked Events

Current allowlisted events include:

- Authentication: `signup_completed`, `login`, `logout`
- Profile: `profile_completed`, `preferences_updated`
- Trips: `trip_created`, `itinerary_generated`, `itinerary_regenerated`,
  `itinerary_edited`, `itinerary_archived`, `trip_reviewed`
- AI: `ai_generation_started`, `ai_generation_completed`,
  `ai_generation_failed`, `repair_triggered`, `fallback_used`,
  `itinerary_accepted`, `place_removed`, `place_replaced`
- Budget/routes/sharing: `budget_created`, `budget_edited`,
  `route_recalculated`, `share_created`, `share_opened`
- Notifications/settings/retention/health: `notification_opened`,
  `experimental_setting_changed`, `user_returned`, `second_trip_created`,
  `error_occurred`
- Feedback: `feedback_submitted`, `ai_feedback_submitted`,
  `bug_report_submitted`, `feature_request_submitted`

Unknown event names are rejected before persistence.

## Funnel Analytics

The alpha dashboard computes the closed-alpha funnel from participant metadata
and accepted events:

1. Invited
2. Activated
3. First login
4. First trip
5. First AI generation
6. Trip reviewed
7. Trip shared
8. User returned
9. Second trip created

Each stage includes users, conversion from invited users, and drop-off from the
previous stage.

## Feature Usage

For each tracked feature, the dashboard reports:

- usage count
- unique users
- first use timestamp
- repeat use count
- unused flag

Usage windows include DAU, WAU, and MAU. Feature names are low-cardinality and
validated by the backend.

## AI Quality Analytics

AI quality combines Trip Service generation traces with privacy-safe events and
feedback:

- generation count
- success, repair, and fallback rates
- average latency
- token usage and estimated OpenAI cost
- regenerated itineraries
- removed and replaced places
- accepted itineraries
- bad-place feedback reports

Provider, model alias, prompt version, and grounding version may be stored as
safe metadata. Raw prompts, raw AI responses, and generated itinerary text are
not stored in analytics.

## Metrics

Trip Service exports:

- `alpha_active_users`
- `alpha_trip_generation_total`
- `alpha_ai_success_total`
- `alpha_feedback_total`
- `alpha_bug_reports_total`
- `alpha_feature_requests_total`
- `alpha_retention_rate`
- `alpha_generation_latency`
- `alpha_openai_cost_estimate`
- `alpha_analytics_events_total`
- `alpha_analytics_rejected_total`

Prometheus rules in
`infra/observability/prometheus/rules/alpha-product-analytics-alerts.yml` notify
Ops about AI failure spikes, fallback spikes, bug report spikes, estimated
OpenAI cost spikes, retention drops, and generation latency increases.
