# Alpha Dashboard

The Ops alpha dashboard is available inside the existing Ops page. It is a
closed-alpha operating surface, not a general analytics product.

## Panels

The dashboard shows:

- user counts: invited, active, inactive, retained
- trip counts: created and completed
- AI metrics: generations, success rate, repair rate, fallback rate, average
  latency, token usage, regenerated itineraries, removed/replaced places,
  accepted itineraries, bad-place reports
- feedback counts: total, bugs, AI reports, feature requests, category/status
  breakdown
- DAU, WAU, MAU
- OpenAI token usage and estimated cost
- health: failures, retries, incidents
- funnel stages and drop-offs
- per-feature usage and unused feature highlights
- dashboard-derived alerts

## Invite And Waitlist Management

Ops can create invite codes, disable invites, choose tester group, set max
activations, set expiration, and invite a waitlist entry. Raw invite codes are
shown only once immediately after creation.

Waitlist status changes should reflect explicit user/team decisions. The system
does not automatically invite users.

## Feedback Center

Ops can filter feedback by status, view the submitted category/title/sanitized
description, and update status or priority. Owner assignment and internal notes
are available through the alpha feedback API.

## Weekly Reports

Ops can generate and read weekly reports from the dashboard. Reports are stored
in Trip Service and should be generated after each alpha week or larger invite
batch.

## Alert Sources

Dashboard alerts come from dashboard metrics. Prometheus alert rules in
`infra/observability/prometheus/rules/alpha-product-analytics-alerts.yml` cover:

- AI failure rate spike
- fallback rate spike
- bug report spike
- estimated OpenAI cost spike
- retention drop
- generation latency increase

Use the dashboard and weekly report together before changing invite pace.
