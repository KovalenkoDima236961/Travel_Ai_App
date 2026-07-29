# Weekly Alpha Report

Trip Service can generate a human-readable weekly report from alpha dashboard
metrics, feedback, analytics events, AI traces, and trip summaries.

## API

| Method/path | Purpose |
| --- | --- |
| `GET /ops/alpha/reports/weekly` | List previously generated reports |
| `POST /ops/alpha/reports/weekly/generate` | Generate and persist a report for a week |

`weekStart` is optional on generate. If omitted, the service uses the current
week start. The report stores `weekStart`, `weekEnd`, `summaryMarkdown`, raw
dashboard metrics JSON, generator user ID, and generation timestamp.

## Contents

Each report includes:

- invited, active, and retained users
- trips created
- AI generation count, success rate, repair rate, fallback rate, and average
  latency
- estimated OpenAI cost
- top bugs
- top feature requests
- most removed AI places
- most popular destinations
- DAU, WAU, MAU, and retention rate
- alert-derived recommendations

The report is intentionally Markdown so it can be pasted into release notes,
issue trackers, Slack, or a go/no-go checklist without a rendering dependency.

## Privacy

Weekly reports must stay aggregate or sanitized. The generator must not include
raw emails, prompts, receipts, comments, private notes, raw AI responses, share
tokens, or full itineraries. Popular destinations and removed-place names should
be reviewed before broad sharing outside the product team.

## Operating Cadence

Generate a report at least once per week during closed alpha and after any
larger invite batch. Use the report to decide whether to:

- invite more testers
- pause invites
- roll back a model or provider change
- prioritize bug fixes
- change onboarding or feedback prompts
