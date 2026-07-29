# Online Model Evaluation

Online evaluation compares a user-visible baseline result with an asynchronous
candidate result produced from the same safe input, grounding, prompt version,
validator version, and policy context. It is evidence for human rollout
decisions, not an automatic promotion mechanism.

Implementation note: this slice adds deployment registration, deterministic
assignment persistence, request metadata propagation, feedback capture, and
ops summary endpoints. Completing asynchronous shadow jobs and writing completed
comparison metrics remains a separate Worker Service wiring task.

## Metrics

Quality metrics include schema validity, grounded-place rate, hallucinated
place count, destination mismatches, duplicates, missing coordinates,
unrealistic duration, overpacked days, opening-hours risk, budget plausibility,
route plausibility, preference match, policy compliance, language quality, and
overall quality.

Reliability metrics include success, timeout, parse failure, validation
failure, repair attempted/succeeded, fallback used, and candidate error code.

Performance metrics include total/model/validation/repair latency, input and
output token estimates, context bytes, and memory/load estimates when
available.

## Comparison Rules

Compute candidate-minus-baseline deltas for score metrics, candidate count
minus baseline count for error metrics, percentage latency change, repair-rate
change, and parse-failure change. Do not compare raw prompt or full itinerary
text in persistent rows.

Candidate failures, skipped shadow requests, timeouts, and missing comparisons
must remain visible in reports to avoid survivorship bias.

## Sampling

Sampling is deterministic, not random per request. A stable assignment bucket
is computed from assignment salt and user/workspace/trip identity. V1 executes
at most one candidate per request and skips shadow under load.

## Minimum Sample Sizes

Default guardrails require at least 50 comparable samples in a 60-minute window.
Higher-risk rollouts should require larger samples, language-specific minimums,
and separate review for multi-destination or policy-heavy trips.

## Latency Evaluation

Shadow latency is measured independently and must never delay the primary
response. For user-visible candidates, p50 and p95 latency are evaluated
against baseline; default hard guidance is no more than a 25 percent p95
increase unless ops explicitly accepts the risk.

## User Feedback

For user-visible candidates, safe feedback categories can be attributed to the
deployment and assignment: better/worse than standard, bad places, bad
schedule, too slow, wrong language, formatting problem, or other. Short notes
are redacted and length-limited. Skipping a travel-day item is not automatically
model failure.

Attribution windows:

- Immediate generation feedback: 24 hours.
- Itinerary corrections: 7 days.
- Post-trip feedback: after trip completion.

## Statistical Cautions

Online traffic is not a frozen benchmark. User mix, destination coverage,
seasonality, provider freshness, language distribution, and opt-in behavior can
shift results. Do not tune repeatedly against the frozen holdout set, do not
train continuously from live traffic, and do not promote from one metric alone.

## Rollback Thresholds

Suggested defaults:

- Candidate failure rate above 5 percent.
- Parse failure rate above 1 percent.
- Any hallucination or destination mismatch regression.
- Repair-rate increase above 5 percent.
- P95 latency increase above 25 percent.
- Overall quality delta below zero.
- Language score drop above 5 percent.

Critical failures in user-visible rollout pause candidate traffic and restore
baseline routing for new requests. Shadow quality regressions continue to be
recorded unless infrastructure guardrails fail.

## Offline vs Online Evaluation

Frozen offline evaluation measures repeatable quality on curated validation,
test, and holdout cases. Online evaluation measures realistic runtime quality,
latency, reliability, and user behavior under production-like traffic. Offline
success is required before online rollout; online success still requires human
approval.
