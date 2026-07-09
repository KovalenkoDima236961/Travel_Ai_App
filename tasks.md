# 3. Add AI Policy-Aware Trip Repair v1: generate AI repair proposals from policy/risk issues, preview diffs, apply selected repaired itinerary safely, and integrate with approval/risk workflow.

You are a senior full-stack engineer and AI product architect. Continue building the web-based AI travel planning application.

Your task:
Implement AI Policy-Aware Trip Repair v1: generate AI repair proposals from policy/risk issues, preview diffs, apply selected repaired itinerary safely, and integrate with approval/risk workflow.

Context:
We already have a microservices-based AI travel planning app.

Existing services:

- Auth Service:
  - Go microservice
  - issues JWT access tokens and refresh tokens
- User/Profile Service:
  - Go microservice
  - stores user profile/preferences
  - owns workspace membership and workspace roles
- Trip Service:
  - Go microservice
  - owns trips, workspace trips, itinerary data, workspace policies, approval workflow, smart approval risk scoring, approval checklist, budgets, workspace shared budgets, cost analytics, cost splitting, availability metadata, templates, AI template adaptation jobs, comments, activity, version history, background jobs, conflict detection, and permissions
  - supports personal trips and workspace trips
  - checks workspace access through User/Profile Service
  - supports trip-level collaboration roles
- Worker Service:
  - Go microservice
  - processes RabbitMQ-backed jobs
- Notification Service:
  - Go microservice
  - owns in-app/email/web-push/SSE notifications and preferences
- External Integrations Service:
  - Go microservice
  - owns places, routes, weather, calendar, exchange rates, prices, availability provider adapters, quota/rate limits
- AI Planning Service:
  - Python FastAPI service
  - supports itinerary generation, partial regeneration, budget optimization, template adaptation, destination context/RAG, validation/repair, and Ollama/mock modes
- Web App:
  - Next.js app under apps/web
  - supports auth, trips, workspace switcher, workspace pages, workspace policies, approval workflow, approval risk scoring, workspace approvals queue, budgets, workspace budgets, trip analytics, cost splitting, templates, AI template adaptation, availability cards, exports, offline mode, PWA install, etc.
- Infra:
  - Postgres
  - RabbitMQ
  - Prometheus/Grafana

Current workflow:

- Workspace owners/admins define policy rules.
- Workspace trips can be submitted for approval.
- Approval checklist evaluates budgets, policy, cost splitting, availability, and quality.
- Smart approval risk scoring explains risk factors and suggested actions.
- Users can manually fix issues or regenerate days/items.
- There is no targeted AI repair flow that turns policy/risk issues into a reviewable repair proposal.

Problem:
The app can identify planning problems, but users still need to fix most of them manually. Reviewers and planners need a safe “Repair with AI” workflow that proposes changes but does not auto-apply them.

Goal:
Add AI policy-aware trip repair:

- User selects repair mode or selected risk/policy factors.
- Trip Service creates a repair job.
- Worker sends current itinerary, policy evaluation, risk factors, and constraints to AI Planning Service.
- AI returns repaired itinerary + structured repair summary.
- Trip Service validates and stores a repair proposal.
- Web App shows before/after diff, risk/policy improvement, cost changes, warnings.
- User applies or discards the proposal.
- Applying checks itineraryRevision, replaces itinerary safely, increments revision, creates version/activity, resets approval if needed, and invalidates risk/approval.
- No auto-apply and no auto-approval.

Do NOT add:

- auto-apply repairs
- auto-approval after repair
- booking changes
- provider price auto-application
- payment/checkout behavior
- legal/compliance guarantees
- full CRDT merge
- complex multi-agent repair system
- AI modifying comments/collaborators/shares/calendar sync
- native mobile
- Kubernetes
- new backend service

For v1:

- Implement AI endpoint in AI Planning Service: POST /repair-itinerary.
- Implement Trip Service repair job and repair proposal storage.
- Use existing Worker Service/background job pattern.
- Use existing itinerary validation/repair logic.
- Use existing conflict detection with itineraryRevision.
- Support proposal preview before apply.
- Support applying full repaired itinerary only, not partial patch apply, unless repo already has a safe diff apply utility.
- Keep repair deterministic in mock mode.
- Keep AI output reviewable and editable after apply.

Important codebase consistency requirement:
Before implementing, inspect existing services and follow the same patterns exactly:

- services/ai-planning-service
- services/trip-service
- services/worker-service
- services/external-integrations-service
- services/user-service
- services/notification-service
- apps/web

Do not invent a different architecture if the repository already has conventions.

Match existing patterns for:

- FastAPI route structure
- Pydantic schemas
- Ollama/mock mode
- AI prompt builder
- AI validation/repair
- Go service modules
- Uber Fx modules
- Zap logging
- config loading
- HTTP middleware
- auth/JWT middleware
- internal service token middleware
- response/error helpers
- pgxpool/sqlc usage
- migrations
- sqlc queries
- background job dispatch
- RabbitMQ worker handling
- trip permissions
- workspace policy evaluation
- approval risk scoring
- itinerary validation
- version history
- activity events
- approval reset-on-edit
- frontend API clients/hooks
- TanStack Query
- UI components
- tests
- smoke scripts
- docs

Part 1: AI Planning Service endpoint

1. Add endpoint:

POST /repair-itinerary

Request:

{
"itinerary": {
"...": "current itinerary JSON"
},
"tripContext": {
"title": "Vienna Weekend",
"destination": "Vienna",
"startDate": "2026-09-10",
"durationDays": 3,
"budget": {
"amount": 700,
"currency": "EUR"
},
"travelers": 3,
"pace": "balanced"
},
"policy": {
"schemaVersion": 1,
"rules": {}
},
"policyEvaluation": {
"status": "blocking",
"results": []
},
"approvalRisk": {
"score": 68,
"status": "high",
"factors": []
},
"issues": [
{
"type": "maxTripBudget",
"severity": "blocking",
"message": "Trip is €320 over budget.",
"affected": {
"dayNumber": 2,
"itemIndex": null
}
}
],
"constraints": {
"repairMode": "policy_compliance",
"selectedIssueTypes": ["maxTripBudget", "noLateActivitiesAfter"],
"preserveConfirmedItems": true,
"minimizeChanges": true,
"preserveUserEditedItems": true,
"doNotChangeAccommodation": false,
"doNotChangeDates": true,
"maxChangedItems": 10,
"specialInstructions": "Prefer cheaper public transport and free museums."
},
"context": {
"userProfile": {},
"userPreferences": {},
"destinationContext": {},
"weatherContext": {}
}
}

Response:

{
"repairedItinerary": {
"...": "valid itinerary JSON"
},
"repairSummary": {
"repairMode": "policy_compliance",
"changedItemCount": 5,
"addedItemCount": 2,
"removedItemCount": 1,
"movedItemCount": 2,
"estimatedCostBefore": {
"amount": 920,
"currency": "EUR"
},
"estimatedCostAfter": {
"amount": 690,
"currency": "EUR"
},
"majorChanges": [
"Moved late concert earlier.",
"Replaced private tour with free walking route."
],
"issuesAddressed": [
"maxTripBudget",
"noLateActivitiesAfter"
],
"issuesRemaining": [
"availability_unchecked"
],
"warnings": [
"Availability must be checked again after repair."
]
},
"changes": [
{
"type": "item_replaced",
"dayNumber": 2,
"itemIndex": 3,
"before": {
"name": "Private tour"
},
"after": {
"name": "Self-guided historic walk"
},
"reason": "Reduce cost and walking risk."
}
]
}

2. Pydantic schemas.

Create or extend:

- RepairItineraryRequest
- RepairTripContext
- RepairIssue
- RepairConstraints
- RepairItineraryResponse
- RepairSummary
- RepairChange

Reuse existing itinerary schemas if available.

3. Repair modes.

Support enum:

- policy_compliance
- reduce_budget_risk
- fix_schedule_risk
- reduce_walking
- add_rest_time
- replace_disallowed_items
- selected_issues

4. Mock mode.

Mock mode must be deterministic.

Behavior:

- Keep itinerary schema valid.
- Add/modify item names with deterministic markers only where relevant.
- For reduce_budget_risk:
  - reduce estimatedCost amounts by a deterministic percentage or replace high-cost items with lower-cost alternatives.
- For fix_schedule_risk:
  - move late items earlier.
- For reduce_walking:
  - add note or reorder nearby items if coordinates exist, otherwise keep stable.
- For add_rest_time:
  - add rest/free_time item per day if missing.
- For replace_disallowed_items:
  - replace item type/name if it matches issue.
- Include repairSummary and changes.

No external calls.

5. Ollama mode.

Add prompt builder for itinerary repair.

The prompt must instruct model to:

- return strict JSON only.
- follow existing itinerary schema exactly.
- address the selected issues first.
- minimize changes where requested.
- preserve confirmed/user-edited items where requested.
- keep trip dates/duration stable unless explicitly allowed.
- keep destination stable.
- do not modify comments/collaborators/share/calendar/approval metadata.
- keep costs as estimates.
- do not claim booking/availability.
- produce structured changes and warnings.

6. Repair prompt strategy.

Prompt sections:

- Role: itinerary repair engine.
- Current itinerary.
- Trip context.
- Workspace policy.
- Policy violations.
- Approval risk factors.
- Repair mode.
- Preservation constraints.
- Output schema.
- JSON-only instruction.

7. Validation and repair.

After AI response:

- validate repaired itinerary against existing schema.
- validate day count and dates.
- validate no malformed times.
- validate item costs if present.
- validate constraints:
  - destination unchanged unless allowed.
  - startDate unchanged unless allowed.
  - duration unchanged unless allowed.
- if invalid:
  - run existing repair pass once if available.
- if still invalid:
  - return controlled error.

8. AI warnings.

Response should warn when:

- policy may still be violated.
- budget may still be too high.
- availability must be rechecked.
- cost estimates are uncertain.
- major changes were necessary.
- repair was partial.

Part 2: Trip Service data model

9. Add repair proposal table.

Create migration in Trip Service:

trip_repair_proposals:

- id UUID primary key
- trip_id UUID not null references trips(id) on delete cascade
- job_id UUID null
- created_by_user_id UUID not null
- status TEXT not null default 'pending'
- repair_mode TEXT not null
- base_itinerary_revision INT not null
- base_risk_score INT null
- proposed_risk_score INT null
- base_policy_status TEXT null
- proposed_policy_status TEXT null
- issues_json JSONB not null
- proposal_json JSONB not null
- created_at TIMESTAMP not null default NOW()
- updated_at TIMESTAMP not null default NOW()
- applied_at TIMESTAMP null
- applied_by_user_id UUID null
- discarded_at TIMESTAMP null
- discarded_by_user_id UUID null
- expired_at TIMESTAMP null

Constraints:

- status in ('pending', 'applied', 'discarded', 'expired', 'failed')
- repair_mode in supported modes
- base_itinerary_revision >= 0

Indexes:

- index on trip_id
- index on job_id
- index on status
- index on created_at DESC
- index on (trip_id, status)

10. Proposal JSON shape.

proposal_json:

{
"repairedItinerary": {},
"repairSummary": {
"repairMode": "policy_compliance",
"changedItemCount": 5,
"estimatedCostBefore": {
"amount": 920,
"currency": "EUR"
},
"estimatedCostAfter": {
"amount": 690,
"currency": "EUR"
},
"majorChanges": [],
"issuesAddressed": [],
"issuesRemaining": [],
"warnings": []
},
"changes": [],
"diff": {
"daysChanged": [],
"itemsAdded": [],
"itemsRemoved": [],
"itemsModified": [],
"itemsMoved": []
},
"validation": {
"valid": true,
"warnings": []
}
}

11. Add job type.

If existing trip_generation_jobs table/job type supports extension, add:

policy_repair

Job payload should include:

- tripId
- repairMode
- selectedIssueTypes
- selectedRiskFactorIds or selectedFactorTypes
- expectedItineraryRevision
- constraints
- specialInstructions

If existing job schema cannot store this cleanly:

- use request_payload JSONB.

Part 3: SQL/sqlc

12. Add queries.

Create sqlc queries:

- CreateTripRepairProposal
- GetTripRepairProposalByID
- ListTripRepairProposalsByTrip
- UpdateTripRepairProposalStatusApplied
- UpdateTripRepairProposalStatusDiscarded
- ExpirePendingRepairProposalsForTripRevision
- GetPendingRepairProposalByJobID
- CreatePolicyRepairJob / reuse generation job queries

Part 4: Trip Service repair module

13. Add module.

Create:

services/trip-service/internal/triprepair/

Suggested files:

- types.go
- dto.go
- service.go
- handler.go
- repository.go
- proposal_builder.go
- diff.go
- validator.go
- module.go
- errors.go

Adjust to existing structure.

Part 5: Trip Service endpoints

14. Add endpoints.

POST /trips/{tripId}/repair-jobs
GET /trips/{tripId}/repair-jobs/{jobId}
GET /trips/{tripId}/repair-proposals
GET /trips/{tripId}/repair-proposals/{proposalId}
POST /trips/{tripId}/repair-proposals/{proposalId}/apply
POST /trips/{tripId}/repair-proposals/{proposalId}/discard

15. Create repair job.

POST /trips/{tripId}/repair-jobs

Request:

{
"expectedItineraryRevision": 12,
"repairMode": "policy_compliance",
"selectedIssueTypes": [
"maxTripBudget",
"noLateActivitiesAfter"
],
"selectedRiskFactorTypes": [
"trip_budget_exceeded",
"late_activity"
],
"constraints": {
"preserveConfirmedItems": true,
"minimizeChanges": true,
"preserveUserEditedItems": true,
"doNotChangeAccommodation": false,
"doNotChangeDates": true,
"maxChangedItems": 10
},
"specialInstructions": "Prefer free activities and public transport."
}

Validation:

- expectedItineraryRevision required.
- repairMode valid.
- selectedIssueTypes max 20.
- selectedRiskFactorTypes max 20.
- specialInstructions max 1000.
- maxChangedItems 1–50 if provided.

Permissions:

- user must have trip edit permission.
- personal trips can use repair if policy/risk exists? Recommended:
  - v1 focuses on workspace trips.
  - For personal trips, allow generic repair only if existing risk/policy not applicable? Recommended: return 400 not_supported_for_personal_trips.
- workspace viewer cannot create repair job.
- public share cannot create repair job.

Behavior:

1. Load trip.
2. Check edit permission.
3. Check expectedItineraryRevision matches current.
4. Evaluate workspace policy.
5. Calculate approval risk.
6. Build issue list from selected issue/risk types, or all major issues if none selected.
7. If no repairable issues:
   - return 400 no_repairable_issues.
8. Create job.
9. Dispatch worker.
10. Return job.

11. Get repair job.

GET /trips/{tripId}/repair-jobs/{jobId}

Return existing job DTO plus:

- repairMode
- status
- createdProposalId if completed
- error if failed

17. List proposals.

GET /trips/{tripId}/repair-proposals?status=pending

View permission required.

Response:
{
"proposals": [
{
"id": "uuid",
"tripId": "uuid",
"jobId": "uuid",
"status": "pending",
"repairMode": "policy_compliance",
"baseItineraryRevision": 12,
"baseRiskScore": 68,
"proposedRiskScore": 35,
"basePolicyStatus": "blocking",
"proposedPolicyStatus": "warning",
"summary": {...},
"createdAt": "..."
}
]
}

List response should not include full repaired itinerary if large.

18. Get proposal detail.

GET /trips/{tripId}/repair-proposals/{proposalId}

View permission required.
Return full proposal_json including repairedItinerary and diff.

19. Apply proposal.

POST /trips/{tripId}/repair-proposals/{proposalId}/apply

Request:

{
"expectedItineraryRevision": 12
}

Permissions:

- edit permission required.

Rules:

- proposal status must be pending.
- proposal.base_itinerary_revision must equal request expectedItineraryRevision.
- current trip itineraryRevision must equal expectedItineraryRevision.
- if current revision changed:
  - return 409 itinerary_conflict.
- proposal must not be expired.

Behavior:

1. Load proposal and trip.
2. Check permission.
3. Check revision.
4. Validate repairedItinerary again.
5. Save repaired itinerary.
6. Increment itineraryRevision.
7. Create itinerary version:
   - source = AI_POLICY_REPAIR if enum supports.
   - otherwise MANUAL_EDIT with metadata:
     {
     "source": "ai_policy_repair",
     "proposalId": "uuid",
     "repairMode": "policy_compliance"
     }
8. Mark proposal applied.
9. Discard/expire other pending repair proposals for older revision.
10. Create activity event:
    - trip_repair_proposal_applied
11. Reset approval to draft if needed through existing reset helper.
12. Recalculate policy/risk optionally for response.
13. Return updated trip + proposal status.

14. Discard proposal.

POST /trips/{tripId}/repair-proposals/{proposalId}/discard

Request:
{
"reason": "Not suitable"
}

Permissions:

- edit permission required, or creator can discard own.
  Recommended:
- edit permission required.

Behavior:

- status discarded.
- discarded_at/by set.
- create activity event optional.

21. Expire proposals.

When itineraryRevision changes due to any material edit:

- expire pending repair proposals for that trip where base_itinerary_revision < current revision.
- Can happen in existing approval reset/material edit flow.
- At minimum, apply endpoint must detect conflict.

Part 6: Worker processing

22. Add worker handler for policy_repair.

Processing steps:

1. Claim job.
2. Load trip and current itinerary.
3. Verify job expected revision still matches.
4. Load workspace policy.
5. Evaluate policy.
6. Calculate approval risk.
7. Build repair issues.
8. Load profile/preferences/context if existing.
9. Call AI Planning Service /repair-itinerary.
10. Validate repaired itinerary.
11. Build diff between current and repaired itinerary.
12. Recalculate proposed policy evaluation/risk using repaired itinerary without saving if possible.
    - If hard to recalc without saving, calculate best-effort using in-memory calculator.
    - Otherwise leave proposedRiskScore null with warning.
13. Store proposal as pending.
14. Mark job completed with proposalId.
15. Notify requester optionally.

16. No auto-apply.

Worker must never replace trip itinerary directly.
Only creates repair proposal.

24. Failure behavior.

If AI fails:

- job failed.
- no proposal created unless deterministic fallback repair exists.
  Recommended v1:
- no deterministic fallback for repair unless mock mode.
- fail clearly.

Error codes:

- itinerary_conflict
- no_repairable_issues
- ai_repair_failed
- validation_failed
- proposal_build_failed
- policy_evaluation_failed
- risk_calculation_failed

25. Proposed risk/policy recalculation.

Best effort:

- Evaluate repaired itinerary with same policy/risk calculators in memory.
- If calculators require DB trip state only, create helper that accepts itinerary override.
- If too large, calculate summary deltas:
  - baseRiskScore
  - proposedRiskScore null
  - include warning.

Recommended:

- Implement calculator override if feasible because before/after is core UI value.

Part 7: Diff generation

26. Backend diff.

Create simple itinerary diff utility.

Detect:

- day_added
- day_removed
- day_modified
- item_added
- item_removed
- item_modified
- item_moved
- item_replaced

Use item id if present.
Fallback identity:

- dayNumber + normalized name + startTime
- if no stable match, compare by index.

Diff item:

{
"type": "item_modified",
"dayNumber": 2,
"itemIndex": 3,
"fieldChanges": [
{
"field": "startTime",
"before": "22:30",
"after": "19:30"
},
{
"field": "estimatedCost.amount",
"before": 120,
"after": 40
}
],
"reason": "Avoid late activity and reduce cost."
}

27. Diff limits.

Limit diff output:

- max 100 changes.
- if too many, include warning:
  “Large repair changed many itinerary items.”

28. Preserve metadata.

AI should not remove:

- split metadata if item is essentially same.
- place metadata if item remains same and still relevant.
- user-edited cost split if cost remains.
- accommodation unless allowed.

Backend validation should warn if important metadata disappeared unexpectedly.

Part 8: AI issue selection

29. Build issues from policy/risk.

If user selected issue/risk types:

- include only selected repairable issues.

If not selected:

- include all blocking/high/medium repairable issues.

Repairable:

- budget exceeded
- daily budget exceeded
- max item cost
- late activity
- missing rest
- walking too high
- disallowed activity type
- accommodation too expensive if allowed
- missing cost estimates maybe repairable by estimating costs
- availability unchecked not directly repairable, but AI can suggest alternatives; better suggested action is check availability.

Not repairable by AI:

- provider outage
- quota exceeded
- missing API key
- need user payment
- approval status itself
- comments/notifications

30. Repair mode mapping.

policy_compliance:

- include policy blocking/warning violations.

reduce_budget_risk:

- include budget/cost factors.

fix_schedule_risk:

- include late activities, overlaps, impossible timing.

reduce_walking:

- include walking/route factors.

add_rest_time:

- include rest/density factors.

replace_disallowed_items:

- include disallowed activity factors.

selected_issues:

- include only selected.

Part 9: Version/history/activity/approval integration

31. Version history.

Applying proposal creates itinerary version with metadata:
{
"source": "ai_policy_repair",
"proposalId": "uuid",
"repairMode": "policy_compliance",
"baseRevision": 12,
"baseRiskScore": 68,
"proposedRiskScore": 35
}

32. Activity events.

Add:

- trip_repair_job_created
- trip_repair_proposal_created
- trip_repair_proposal_applied
- trip_repair_proposal_discarded
- trip_repair_proposal_expired

Metadata safe:

- proposalId
- repairMode
- baseRiskScore
- proposedRiskScore
- changedItemCount
- warningCount

Do not include full itinerary in activity metadata.

33. Approval reset.

Applying repair is a material edit:

- if status approved or pending_approval, reset to draft using existing helper.
- create reset approval event as existing logic does.

34. Notifications.

Optional:

- Notify trip collaborators/workspace owners when a repair proposal is applied.
  Recommended:
- Use existing activity feed only in v1 to avoid notification spam.
- If notifying, exclude actor and keep metadata small.

Part 10: Web App types/API/hooks

35. Add types.

Create:

apps/web/types/trip-repair.ts

Types:

- RepairMode
- RepairJob
- RepairProposal
- RepairProposalStatus
- RepairProposalDetail
- RepairSummary
- RepairChange
- RepairDiff
- CreateRepairJobInput
- ApplyRepairProposalInput

36. API client.

Create:

apps/web/lib/api/trip-repair.ts

Functions:

- createTripRepairJob(tripId, input)
- getTripRepairJob(tripId, jobId)
- listTripRepairProposals(tripId, params)
- getTripRepairProposal(tripId, proposalId)
- applyTripRepairProposal(tripId, proposalId, input)
- discardTripRepairProposal(tripId, proposalId, reason?)

37. Hooks.

Create:

apps/web/hooks/useCreateTripRepairJob.ts
apps/web/hooks/useTripRepairJob.ts
apps/web/hooks/useTripRepairProposals.ts
apps/web/hooks/useTripRepairProposal.ts
apps/web/hooks/useApplyTripRepairProposal.ts

Use TanStack Query:

- poll job status until terminal.
- invalidate trip/proposals/risk/policy/approval/activity after apply/discard.
- stop polling on completed/failed/cancelled.

Part 11: Web App UI

38. Entry points.

Add “Repair with AI” button in:

- TripApprovalPanel
- RiskScoreCard / RiskSuggestedActions
- TripPolicyPanel
- WorkspaceApprovalsQueue item actions optional

Show only if:

- user has edit permission.
- trip is workspace trip.
- risk/policy issues exist.
- online.

Hide/disable for:

- viewer
- public share
- personal trip if unsupported
- offline mode

39. Repair dialog.

Create:

apps/web/components/trip-repair/CreateRepairJobDialog.tsx

Fields:

- repairMode
- issue selection list
- constraints:
  - preserve confirmed items
  - minimize changes
  - preserve user-edited items
  - do not change accommodation
  - do not change dates
  - max changed items
- specialInstructions

Issue list:

- from approval risk factors and policy evaluation results.
- grouped by severity.
- default selected:
  - blocking/high factors.

CTA:

- Generate repair proposal

40. Job status card.

Create:

apps/web/components/trip-repair/RepairJobStatusCard.tsx

States:

- queued
- running
- validating
- proposal ready
- failed

If completed:

- show View proposal.

If failed:

- show error and retry action.

41. Proposal list.

Create:

apps/web/components/trip-repair/RepairProposalsPanel.tsx

Show pending proposals:

- repair mode
- createdAt
- base revision
- risk before/after
- policy before/after
- changed item count
- status
- actions:
  - Preview
  - Apply
  - Discard

42. Proposal detail/preview.

Create:

apps/web/components/trip-repair/RepairProposalPreview.tsx

Show:

- summary
- before/after risk score
- before/after policy status
- cost before/after
- major changes
- warnings
- diff viewer
- apply/discard buttons

43. Diff viewer.

Create:

apps/web/components/trip-repair/ItineraryRepairDiff.tsx

Display grouped changes:

- Added items
- Removed items
- Modified items
- Moved items
- Day-level changes

For each change:

- day
- item
- before
- after
- reason if available

Keep readable; do not build complex CRDT diff.

44. Apply flow.

Apply button:

- requires confirmation:
  “Apply this repaired itinerary? This will replace the current itinerary and create a new version.”
- sends expectedItineraryRevision = proposal.baseItineraryRevision.
- handles 409:
  - show “Trip changed since this proposal was created. Generate a new repair proposal.”
- on success:
  - navigate/refetch trip
  - show toast:
    “Repair applied. Please review the itinerary.”

45. Discard flow.

Discard button:

- optional reason.
- confirmation.
- updates proposal state.

46. Approval integration.

In approval panel:

- If risk high/critical or policy blocking/warning:
  - show repair CTA.
- If pending proposal exists:
  - show “Repair proposal ready.”

47. Offline behavior.

If offline:

- show proposals if cached maybe read-only.
- disable create/apply/discard:
  “AI repair requires internet.”

Part 12: Frontend suggested action integration

48. Risk action router.

Update existing approval-risk action router:

- action type `repair_with_ai`
- opens CreateRepairJobDialog with selected factor type.

49. Policy panel actions.

Policy evaluation suggested actions for repairable violations can include:

- repair_with_ai
- regenerate_day
- open_budget_optimization

For repair_with_ai:

- pass selected ruleKey.

Part 13: Observability

50. Metrics.

Add metrics if existing:

- trip_repair_jobs_created_total
- trip_repair_jobs_completed_total
- trip_repair_jobs_failed_total
- trip_repair_proposals_created_total
- trip_repair_proposals_applied_total
- trip_repair_proposals_discarded_total
- trip_repair_ai_duration_seconds
- trip_repair_validation_failures_total

Labels:

- repair_mode
- status
- fallback? if any
  Avoid:
- tripId
- destination
- userId
- template title

51. Logs.

Structured logs:

- jobId
- proposalId
- tripId
- userId
- repairMode
- baseRevision
- baseRiskScore
- proposedRiskScore
- changedItemCount
- durationMs
- errorCode

Do not log full itinerary/prompt in production.

Part 14: Backend tests

52. AI Planning Service tests.

Test:

- mock repair policy_compliance returns valid itinerary.
- mock repair reduce_budget_risk reduces costs.
- mock repair fix_schedule_risk moves late items earlier.
- mock repair add_rest_time adds rest item.
- validates request.
- rejects invalid repairMode.
- prompt builder includes issues/policy/risk/constraints.
- JSON-only instruction present.

53. Trip Service endpoint tests.

Test:

- editor can create repair job.
- viewer cannot create repair job.
- personal trip unsupported if chosen.
- stale expectedItineraryRevision returns 409.
- no repairable issues returns 400.
- job created and dispatched.
- proposal list/detail permission enforced.

54. Worker tests.

Test:

- successful AI repair creates pending proposal, not applied trip.
- proposal contains repairedItinerary, summary, diff.
- job completed with proposalId.
- AI failure fails job.
- invalid AI output fails or repairs once.
- proposed risk/policy calculated if implemented.
- no auto-apply.

55. Apply tests.

Test:

- apply pending proposal with matching revision succeeds.
- itineraryRevision increments.
- itinerary version created.
- activity created.
- proposal marked applied.
- approval reset to draft if prior approved/pending.
- stale revision returns 409.
- applied/discarded proposal cannot be applied.
- other pending proposals expired after apply.

56. Discard tests.

Test:

- editor can discard.
- viewer cannot discard.
- discarded proposal cannot apply.

57. Diff tests.

Test:

- item added detected.
- item removed detected.
- item modified detected.
- item moved detected.
- large diff capped.

58. Permission tests.

Test:

- workspace member/editor can repair.
- workspace viewer cannot.
- trip collaborator editor can repair if trip permissions allow.
- public share denied.

Part 15: Frontend tests

59. API/hook tests.

- create repair job calls endpoint.
- job hook polls until completed.
- proposal list hook works.
- apply mutation handles success/conflict.
- discard mutation works.

60. Dialog tests.

CreateRepairJobDialog:

- shows risk/policy issues.
- defaults high/blocking issues selected.
- validates maxChangedItems.
- submit sends correct payload.
- hidden/disabled for viewer/offline.

61. Proposal preview tests.

- shows risk before/after.
- shows policy before/after.
- shows major changes.
- shows warnings.
- diff groups changes.
- apply confirmation.
- conflict error shown.

62. Integration tests.

- RiskScoreCard repair action opens dialog.
- TripPolicyPanel repair action opens dialog.
- ApprovalPanel shows pending proposal.
- Apply invalidates trip/risk/approval queries.

Part 16: Smoke tests

63. Update scripts/smoke-test.sh.

API smoke:

1. Login workspace owner/member.
2. Create workspace trip with itinerary.
3. Add policy that creates warning/blocking violation.
4. GET risk and confirm issue exists.
5. POST /trips/{tripId}/repair-jobs with mock AI.
6. Poll job until completed.
7. GET repair proposal.
8. Assert proposal pending and contains repairedItinerary/diff.
9. Apply proposal with expected revision.
10. Assert trip itineraryRevision incremented.
11. Assert proposal applied.
12. Assert approval status reset to draft if applicable.
13. Try applying same proposal again and assert error.
14. Try stale revision and assert conflict.

15. Update scripts/web-smoke-test.md.

Manual test:

1. Create workspace trip with policy/risk issues.
2. Open approval panel/risk panel.
3. Click Repair with AI.
4. Select budget/schedule issues.
5. Generate proposal.
6. Review proposal diff.
7. Apply proposal.
8. Confirm itinerary changed and risk/policy improved.
9. Confirm activity/version created.
10. Confirm approval status is draft.
11. Generate another proposal, edit trip manually, then try applying old proposal and confirm conflict.

Part 17: Documentation

65. Update AI Planning Service README.

Document:

- /repair-itinerary endpoint
- request/response schema
- repair modes
- mock/ollama behavior
- validation/repair
- limitations

66. Update Trip Service README.

Document:

- policy_repair job type
- trip_repair_proposals table
- repair job endpoints
- proposal apply/discard endpoints
- permission rules
- revision conflict behavior
- version/activity/approval reset behavior
- limitations

67. Update Web App README.

Document:

- Repair with AI flow
- repair modes
- proposal preview
- diff viewer
- apply/discard behavior
- conflict handling
- limitations

68. Update root README.md.

Mention:

- AI Policy-Aware Trip Repair v1.

69. User-facing limitations.

Document:

- AI repair creates proposals, not automatic changes.
- User must review before applying.
- Repair does not guarantee approval.
- Repair does not book anything.
- Availability and prices must be checked again.
- Applying repair replaces the itinerary and creates a version.
- If the trip changes after proposal creation, generate a new proposal.

Part 18: Security and quality requirements

- Backend must enforce trip edit permissions.
- Public share cannot create/apply repair proposals.
- AI repair must not modify comments/collaborators/public shares/calendar sync.
- AI prompt must not include secrets or unauthorized private data.
- Do not log full prompt/itinerary in production.
- AI output must be validated before proposal is stored.
- Proposal apply must check itineraryRevision to avoid overwriting newer edits.
- No auto-apply.
- No auto-approval.
- No booking/payment actions.
- Applying repair must create version/activity and reset approval if needed.
- Existing approval/risk/policy workflows must not regress.
- Existing itinerary editing/version/conflict logic must not regress.
- Keep code consistent with existing service patterns.
- Keep tests and docs updated.

Expected output:
AI Planning Service exposes `/repair-itinerary`.
Trip Service supports policy_repair jobs and repair proposals.
Worker generates AI repair proposals from policy/risk issues without auto-applying.
Web App lets users generate, preview, apply, or discard repair proposals.
Proposal preview shows risk/policy improvement, cost changes, warnings, and itinerary diff.
Applying proposal safely checks itineraryRevision, replaces itinerary, increments revision, creates version/activity, resets approval if needed, and invalidates risk/approval.
Docs, tests, and smoke tests are updated.

# 4. Internationalization v1: add English, Spanish, Ukrainian, and French language selection, translate Web App UI, store preferred language in user profile, pass language to AI generation/adaptation/repair, and localize exports/notifications where practical

You are a senior full-stack engineer and product-minded frontend architect. Continue building the web-based AI travel planning application.

Your task:
Implement Internationalization v1: support English, Spanish, Ukrainian, and French across the Web App, user language preference, localized UI text, localized AI output language, and localized exports/notifications where practical.

Context:
We already have a microservices-based AI travel planning app.

Existing services:

- Auth Service:
  - Go microservice
  - issues JWT access tokens and refresh tokens
- User/Profile Service:
  - Go microservice
  - stores user profile/preferences
  - currently stores or should store preferred language if available
- Trip Service:
  - Go microservice
  - owns trips, workspace trips, itinerary data, templates, AI generation jobs, AI template adaptation jobs, AI repair jobs, budgets, workspace budgets, cost analytics, cost splitting, approval workflow, workspace policies, approval risk scoring, comments, activity, version history, notifications integration, and permissions
  - supports personal trips and workspace trips
  - calls AI Planning Service for generation/regeneration/optimization/adaptation/repair
- Worker Service:
  - Go microservice
  - processes RabbitMQ-backed jobs
- Notification Service:
  - Go microservice
  - owns in-app/email/web-push/SSE notifications and notification preferences
- External Integrations Service:
  - Go microservice
  - owns places, routes, weather, calendar, exchange rates, prices, availability provider adapters, quota/rate limits
- AI Planning Service:
  - Python FastAPI service
  - supports itinerary generation, partial regeneration, budget optimization, template adaptation, policy-aware repair, destination context/RAG, validation/repair, and Ollama/mock modes
- Web App:
  - Next.js app under apps/web
  - supports auth, trips, workspace switcher, workspace pages, templates, AI template adaptation, budgets, workspace budgets, cost analytics, cost splitting, approval workflow, workspace policies, approval risk scoring, AI repair proposals, availability cards, exports, offline mode, PWA install, notifications, etc.
- Infra:
  - Postgres
  - RabbitMQ
  - Prometheus/Grafana

Current language behavior:

- The app currently supports only English user-facing UI.
- AI-generated itinerary text is likely generated in English.
- Exports, notifications, toasts, empty states, validation messages, and dialogs are likely English-only.

Goal:
Add four selectable languages:

- English
- Spanish
- Ukrainian
- French

Use language codes:

- en = English
- es = Spanish
- uk = Ukrainian
- fr = French

Important:
“Spanish” and “French” are languages. “Spain” and “France” are countries. Use language names/codes internally.

High-level requirements:

- Add Web App internationalization.
- Add language selector.
- Persist preferred language in user profile.
- Use selected language for Web App UI.
- Pass preferred language to AI Planning Service so generated content can be in the selected language.
- Localize major exports where practical.
- Localize notifications where practical.
- Keep English as fallback.
- Do not translate existing stored trips automatically.
- Do not introduce a heavy translation CMS in v1.

Do NOT add:

- automatic translation of all historical trips
- paid translation service
- localization CMS
- translator admin dashboard
- country-specific legal/travel rules
- region-specific tax/accounting/payment localization
- browser plugin translation
- native mobile
- Kubernetes
- new backend service

For v1:

- Focus on app UI, user preference, AI output language, and main generated/exported text.
- Translate core user-facing strings in Web App.
- Backend error codes can remain stable, but frontend should map common errors to localized messages where possible.
- Existing data that was saved in English remains English.
- New AI-generated text should follow selected language.
- Keep route structure simple unless repo already supports locale-prefixed routes.
- English fallback must work for missing translation keys.

Important codebase consistency requirement:
Before implementing, inspect existing services and follow the same patterns exactly:

- apps/web
- services/user-service
- services/trip-service
- services/worker-service
- services/ai-planning-service
- services/notification-service

Do not invent a different architecture if the repository already has conventions.

Match existing patterns for:

- Next.js routing/app router
- layouts/providers
- settings page
- profile/preferences API
- frontend API clients/hooks
- TanStack Query
- forms/validation
- toast/error handling
- exports
- notification UI
- Go service config/DTO patterns
- AI request DTOs
- tests
- smoke scripts
- docs

Part 1: Language model and constants

1. Define supported languages.

In Web App, create:

apps/web/lib/i18n/languages.ts

Export:

export const SUPPORTED_LANGUAGES = ["en", "es", "uk", "fr"] as const;

export type SupportedLanguage = typeof SUPPORTED_LANGUAGES[number];

export const DEFAULT_LANGUAGE: SupportedLanguage = "en";

export const LANGUAGE_LABELS = {
en: "English",
es: "Español",
uk: "Українська",
fr: "Français"
};

2. Backend language enum.

In User/Profile Service and Trip Service DTOs, use:

- en
- es
- uk
- fr

Reject unsupported values with validation error.

3. Fallback behavior.

Fallback order:

1. user preferredLanguage if set and supported.
2. browser language if supported.
3. English.

Browser mapping:

- en-US/en-GB -> en
- es-ES/es-MX/etc -> es
- uk-UA -> uk
- fr-FR/fr-CA/etc -> fr

Part 2: Web App i18n library

4. Choose i18n approach.

Preferred for Next.js:

- next-intl

If the project already has an i18n library, use the existing one.
If not, add next-intl.

Do not add multiple i18n systems.

5. Translation files.

Create:

apps/web/messages/en.json
apps/web/messages/es.json
apps/web/messages/uk.json
apps/web/messages/fr.json

Keep namespaces organized:

{
"common": {},
"navigation": {},
"auth": {},
"settings": {},
"profile": {},
"trips": {},
"itinerary": {},
"templates": {},
"workspaces": {},
"budgets": {},
"analytics": {},
"costSplitting": {},
"approval": {},
"policy": {},
"risk": {},
"repair": {},
"availability": {},
"notifications": {},
"exports": {},
"offline": {},
"pwa": {},
"errors": {},
"validation": {}
}

6. Provider.

Add i18n provider in Web App root layout/provider layer.

If using next-intl without locale-prefixed routes:

- load messages based on stored/current language.
- provide messages to client components.

If using locale-prefixed routes:

- add routes like /en, /es, /uk, /fr only if this does not disrupt existing auth/deep links.
  Recommended v1:
- Avoid locale-prefixed routes unless app already uses that architecture.
- Store language in user profile/localStorage and apply in provider.

7. Translation helper.

Create utilities:

- getInitialLanguage()
- normalizeLanguage(input)
- isSupportedLanguage(input)
- getBrowserLanguage()
- getStoredLanguage()
- setStoredLanguage()

Local storage key:
app_language

Part 3: Translation coverage

8. Translate core UI first.

Replace hardcoded strings with translation keys in:

- main navigation
- auth pages
- dashboard/home
- trip list
- trip create form
- trip detail page
- itinerary item cards
- itinerary editing controls
- templates pages/dialogs
- workspace switcher/pages/settings
- budgets and workspace budgets
- cost analytics dashboard
- cost splitting UI
- approval panel/dialogs/checklist
- workspace policy rules UI
- approval risk scoring UI
- AI repair proposal UI
- availability card
- notifications dropdown/page
- settings/profile page
- offline/PWA pages
- common buttons/toasts/dialogs

9. Common keys.

At minimum include:

common:

- save
- cancel
- close
- delete
- edit
- create
- update
- apply
- discard
- retry
- loading
- error
- success
- warning
- confirm
- back
- next
- previous
- search
- filter
- clear
- yes
- no
- enabled
- disabled
- optional
- required

navigation:

- trips
- templates
- workspaces
- settings
- notifications
- offlineTrips
- approvals
- budgets
- analytics

10. Validation messages.

Localize common frontend validation:

- required field
- invalid email
- invalid date
- invalid currency
- amount must be positive
- value too short
- value too long
- unsupported language
- invalid time
- invalid percentage

11. Error messages.

Keep backend error codes stable.
In frontend, map common error codes to localized messages.

Create:

apps/web/lib/i18n/error-messages.ts

Example:

- unauthorized
- forbidden
- not_found
- validation_error
- itinerary_conflict
- edit_lock_conflict
- workspace_policy_blocking_violation
- provider_rate_limited
- provider_quota_exceeded
- ai_generation_failed
- repair_proposal_conflict

Fallback:

- English generic error
- or backend message if no translation exists

12. Dates/currencies/numbers.

Add formatting utilities:

apps/web/lib/i18n/format.ts

Functions:

- formatDate(date, language)
- formatDateTime(date, language)
- formatMoney(amount, currency, language)
- formatNumber(value, language)
- formatPercent(value, language)

Use Intl APIs.

Locale mapping:

- en -> en-US
- es -> es-ES
- uk -> uk-UA
- fr -> fr-FR

13. Avoid translating dynamic user content.

Do not automatically translate:

- trip titles created by users
- comments
- notes written by users
- existing itinerary text
- workspace names
- template names
- traveler names

Only translate UI labels and new AI outputs when requested/generated.

Part 4: Language selector UI

14. Add language selector component.

Create:

apps/web/components/i18n/LanguageSelector.tsx

Display:

- English
- Español
- Українська
- Français

Behavior:

- updates local language immediately.
- saves to localStorage.
- if authenticated, updates user profile preferredLanguage.
- shows toast:
  “Language updated” in selected language.

15. Place selector.

Add to:

- Settings page
- optionally user menu/header

Recommended:

- Settings → Profile/Preferences section
- user menu compact selector if app has one

16. Loading behavior.

If user is authenticated:

- load profile preferredLanguage.
- apply it.
- localStorage acts as fast initial fallback.
- if profile differs from localStorage, prefer profile after loaded and update localStorage.

If anonymous:

- use localStorage/browser language.

17. Auth boundary.

On logout:

- do not necessarily clear app language.
- Language is device preference and can remain.
- If existing logout clears all local data, preserve app_language if appropriate.

Part 5: User/Profile Service changes

18. Add preferredLanguage if missing.

If user profile already has preferredLanguage:

- validate supported values and reuse.

If missing:

- add migration/field:
  preferred_language TEXT NOT NULL DEFAULT 'en'

or nullable default:
preferred_language TEXT NULL

Recommended:

- default en for existing users.

Constraint:

- preferred_language in ('en','es','uk','fr')

19. Update profile DTOs.

GET profile should return:
{
"preferredLanguage": "uk"
}

PUT/PATCH profile should accept:
{
"preferredLanguage": "uk"
}

20. Validation.

Reject unsupported language:

- 400 validation_error
- field preferredLanguage

21. Tests.

Add User/Profile Service tests:

- default language en.
- update preferredLanguage to es/uk/fr.
- reject unsupported value.
- GET profile returns preferredLanguage.

Part 6: Trip Service / AI language propagation

22. Add language to AI requests.

Every Trip Service call to AI Planning Service should include desired output language:

- full itinerary generation
- day regeneration
- item regeneration
- quality improvement
- budget optimization
- template adaptation
- policy-aware repair

Add field:
language: "en" | "es" | "uk" | "fr"

or:
outputLanguage: "uk"

Use one name consistently.
Recommended:

- outputLanguage

23. Determine output language.

Trip Service should determine language in this order:

1. explicit language in request if endpoint accepts it.
2. user profile preferredLanguage from User/Profile Service.
3. trip language if stored in trip metadata.
4. en.

For v1:

- Use user profile preferredLanguage.
- Optional: allow frontend to pass outputLanguage when starting AI job.

24. Add optional trip language metadata.

Recommended:

- Add trip field if useful:
  language TEXT NULL

If adding DB field is too much:

- store in trip metadata if existing.
- For v1, user preference may be enough.

Better v1:

- Add `language` to trip create/generation request and store on trips:
  - trip.language TEXT NOT NULL DEFAULT 'en'
    This helps maintain language consistency for future regenerations.

Migration:
ALTER TABLE trips ADD COLUMN language TEXT NOT NULL DEFAULT 'en';
constraint language in ('en','es','uk','fr').

For existing trips:

- language = en.

25. New trip creation.

When creating trip:

- default language = current user preferredLanguage.
- frontend can allow user to override language in advanced settings if desired.
- keep simple: no separate selector in trip create unless easy.

26. Regeneration.

When regenerating existing trip/day/item:

- use trip.language if set.
- fallback user preferredLanguage.
  This prevents a French trip from suddenly regenerating in Ukrainian if user changed UI language later.

27. Template adaptation / repair.

When adapting template:

- target trip language = request.outputLanguage or current user preferredLanguage.
- AI output should use that language.
- created trip stores language.

When repairing:

- use trip.language.
- Repair summary can be in same language or frontend-translated?
  Recommended:
- AI-generated repair summary in trip.language.
- UI labels translated separately.

Part 7: AI Planning Service changes

28. Add language field to schemas.

Update Pydantic request schemas:

- GenerateItineraryRequest
- RegenerateDayRequest
- RegenerateItemRequest
- BudgetOptimizationRequest
- TemplateAdaptationRequest
- RepairItineraryRequest

Add:
outputLanguage: Literal["en", "es", "uk", "fr"] = "en"

29. Prompt language instruction.

Add prompt section:

Output language:

- Write all user-facing itinerary content in {languageName}.
- This includes day titles, item names, descriptions, notes, summaries, warnings, and recommendation text.
- Keep JSON keys in English.
- Keep enum values in English.
- Keep currency codes unchanged.
- Keep place names in their common/local form where appropriate.
- Do not mix languages unless the place name or proper noun is naturally local.

30. Language names.

Map:

- en: English
- es: Spanish
- uk: Ukrainian
- fr: French

31. Strict JSON.

Continue requiring strict JSON.
Only text values inside JSON should be localized.
Keys/enums remain stable.

Example:
{
"type": "activity",
"name": "Прогулянка історичним центром",
"description": "Почніть день з огляду..."
}

Do not translate:

- type enum
- category enum
- source enum
- status enum

32. Mock mode.

Mock mode should produce deterministic localized text.

Example:

- en: "Morning city walk"
- es: "Paseo matutino por la ciudad"
- uk: "Ранкова прогулянка містом"
- fr: "Promenade matinale en ville"

Add simple translation map for mock output.

33. Validation.

Language does not change schema validation.
Ensure validation accepts localized text in string fields.

34. Tests.

AI Planning Service tests:

- request with outputLanguage uk returns Ukrainian mock text.
- request with outputLanguage es returns Spanish mock text.
- JSON keys/enums stay English.
- unsupported language rejected.
- prompt builder includes language instruction.

Part 8: Notifications localization

35. Notification Service approach.

For v1, localize notification title/message at creation time if recipient language is available.

Preferred:

- The service creating notification sends:
  - type
  - metadata
  - recipient user ID
- Notification Service resolves recipient preferredLanguage and renders localized title/message.

If current architecture sends title/message directly:

- Add optional fields:
  - titleKey
  - messageKey
  - messageParams
  - language
    or keep existing behavior and localize only frontend static notification labels.

Recommended v1 pragmatic approach:

- Do not rewrite whole notification architecture.
- Add localized frontend rendering for known notification types if notification metadata has type/params.
- For notifications that only store plain title/message, leave stored language as-is.

36. Notification type translations.

Add frontend translations for common notification types:

- workspace_invited
- comment_created
- trip_submitted_for_approval
- trip_approved
- trip_changes_requested
- trip_approval_reset_to_draft
- repair_proposal_ready
- budget_exceeded
- role_changed
- collaborator_invited

37. Future-proof.

If easy, update Notification Service to store:

- type
- metadata
- title
- message

Frontend can render localized title from type/metadata and fallback to stored title.

38. Email notifications.

For v1:

- If email templates are simple and Notification Service can access user preferredLanguage, localize key templates.
- Otherwise, keep email in English and document limitation.

Recommended:

- Localize at least subject/body for high-priority emails if architecture supports it:
  - workspace invitation
  - approval requested
  - changes requested
  - trip approved

Part 9: Exports localization

39. PDF exports.

Update export components:

- trip PDF
- cost analytics PDF
- workspace budget PDF
- cost splitting PDF
- repair proposal PDF if exists

Use translations for:

- section headings
- labels
- disclaimers
- generated at
- totals
- warnings
- table headers

Do not translate user-generated trip content unless it was generated in that language already.

40. CSV exports.

CSV headers should be localized based on current UI language.

Examples:

- en: Day, Time, Item, Cost
- uk: День, Час, Елемент, Вартість
- es: Día, Hora, Elemento, Coste
- fr: Jour, Heure, Élément, Coût

41. ICS exports.

Calendar event titles/descriptions:

- Use itinerary content as stored.
- Static labels in descriptions can be localized.

42. Export disclaimer.

Translate disclaimers:

- costs are estimates
- availability may change
- not accounting/payment record
- AI-generated draft needs review

Part 10: Forms and validation libraries

43. Zod / React Hook Form.

If validation messages are hardcoded:

- replace with translation functions.
- For shared schemas outside React components, return error codes and map to localized messages in UI if easier.

44. Backend validation messages.

Do not rely on backend English messages for UI.
Frontend should show localized generic messages using error code/field.

45. Toasts.

Localize all toasts:

- saved
- created
- updated
- deleted
- copied
- applied
- failed
- conflict
- permission denied

Part 11: Data model implications

46. Existing content.

Do not migrate existing content.
Existing trips/templates/comments remain in whatever language they were created.

47. New generated content.

New generated AI content should follow trip.language/outputLanguage.

48. Editing content.

If user edits an item manually, keep their typed language.
Do not auto-translate manual edits.

49. Templates.

Template metadata has user-written text.
Do not translate automatically.

When using/adapting template:

- deterministic create-from-template copies content as-is.
- AI adaptation can output target language.

50. Search.

Do not implement multilingual search in v1.
Existing search may match stored text only.

Part 12: Web App routing and SEO

51. Routing.

Recommended v1:

- Do not add locale prefix routes.
- Keep existing routes stable:
  - /trips
  - /settings
  - /workspaces
- Language changes affect UI content only.

52. HTML lang attribute.

Update document/html lang attribute according to selected language.

If app router root layout cannot access client-selected language server-side:

- set via client effect on document.documentElement.lang.
- If using next-intl server provider, set properly.

53. Direction.

All four languages are LTR.
No RTL support needed in v1.

Part 13: Tests

54. Web App i18n unit tests.

Test:

- normalizeLanguage.
- browser language mapping.
- fallback to English.
- missing translation fallback.
- formatDate/formatMoney for all languages.

55. Language selector tests.

Test:

- renders four languages.
- selecting Ukrainian updates localStorage.
- authenticated selection calls profile update.
- unsupported language not accepted.

56. UI translation tests.

Smoke-level component tests:

- navigation renders Spanish labels.
- settings renders Ukrainian labels.
- trip create form renders French labels.
- validation message localized.

57. AI language propagation tests.

Trip Service tests:

- trip create stores user preferred language.
- generation request includes outputLanguage.
- regeneration uses trip.language.
- template adaptation uses selected/user language.
- repair uses trip.language.

AI Planning Service tests:

- prompt includes language instruction.
- mock output localized.
- enums remain English.

58. User/Profile Service tests.

- update preferredLanguage.
- reject unsupported language.
- default existing profile language.

59. Export tests.

- CSV headers localized.
- PDF labels/disclaimers localized if test setup supports.

Part 14: Smoke tests

60. Update scripts/smoke-test.sh.

API smoke:

1. Create/login user.
2. Update profile preferredLanguage to uk.
3. Create trip.
4. Assert trip.language = uk if added.
5. Trigger AI generation in mock mode.
6. Assert AI request/output uses Ukrainian mock text.
7. Change preferredLanguage to fr.
8. Regenerate existing Ukrainian trip.
9. Assert regeneration still uses trip.language uk, not fr, if trip.language implemented.
10. Create new trip and assert language fr.
11. Reject unsupported language "de".

12. Update scripts/web-smoke-test.md.

Manual test:

1. Open Settings.
2. Switch language to Ukrainian.
3. Confirm navigation/settings/trip UI updates.
4. Create a new trip.
5. Generate itinerary.
6. Confirm generated text is Ukrainian.
7. Switch UI to French.
8. Confirm UI changes to French.
9. Confirm existing Ukrainian trip content remains Ukrainian.
10. Create a new trip and confirm generated text can be French.
11. Export PDF/CSV and confirm labels are localized.
12. Test Spanish and English fallback.

Part 15: Documentation

62. Update Web App README.

Document:

- supported languages
- translation file structure
- how to add a new translation key
- language selector behavior
- fallback behavior
- formatting utilities
- limitations

63. Update User/Profile Service README.

Document:

- preferredLanguage field
- allowed values
- validation

64. Update Trip Service README.

Document:

- trip language/outputLanguage behavior
- AI request language propagation
- regeneration language behavior

65. Update AI Planning Service README.

Document:

- outputLanguage field
- prompt language rules
- mock localization behavior
- keys/enums remain English

66. Update Notification Service README if changed.

Document:

- localized notification strategy
- limitations

67. Update root README.md.

Mention:

- Internationalization v1 with English, Spanish, Ukrainian, French.

68. User-facing limitations.

Document:

- existing trips are not automatically translated.
- user comments/manual notes are not auto-translated.
- some emails/backend errors may remain English in v1.
- place names may remain in local/common language.
- AI output language depends on selected trip/output language and model quality.

Part 16: Security and quality requirements

- Do not expose user profile data while resolving language.
- Do not use external translation APIs in v1.
- Do not send private user content to any new service for translation.
- Keep translation files in repository.
- English fallback must always work.
- Missing translation keys should not crash the app.
- Backend must validate supported language codes.
- AI JSON keys/enums must remain stable and English.
- AI text values should follow outputLanguage.
- Existing routes should not break.
- Existing trips/templates/comments must not be mutated.
- Existing AI generation/adaptation/repair flows must not regress.
- Existing exports must not regress.
- Keep code consistent with existing service patterns.
- Keep tests and docs updated.

Expected output:
The Web App supports English, Spanish, Ukrainian, and French UI.
Users can select language in settings, and the preference is persisted.
User/Profile Service stores preferredLanguage.
Trip Service passes outputLanguage to AI Planning Service and stores/uses trip language where implemented.
AI Planning Service generates user-facing itinerary/adaptation/repair text in the requested language while keeping JSON keys/enums stable.
Main exports and notification UI are localized where practical.
English fallback works for missing keys.
Docs, tests, and smoke tests are updated.

# 5. AI Trip Discovery v1: create a beautiful AI-powered trip creation flow where users can describe their desired trip, get destination suggestions based on preferences and previous trips, use a smart “Surprise me” button, refine bad suggestions, and create a trip from the selected destination.

You are a senior full-stack engineer and product-minded AI architect. Continue building the web-based AI travel planning application.

Your task:
Implement AI Trip Discovery v1: create a beautiful AI-powered trip creation flow where users can describe their desired trip, get destination suggestions based on preferences and previous trips, use a smart “Surprise me” button, refine bad suggestions, and create a trip from the selected destination.

Context:
We already have a microservices-based AI travel planning app.

Existing services:

- Auth Service:
  - Go microservice
  - issues JWT access tokens and refresh tokens
- User/Profile Service:
  - Go microservice
  - stores user profile/preferences
  - stores preferredLanguage
  - owns workspace membership and workspace roles
- Trip Service:
  - Go microservice
  - owns trips, workspace trips, trip creation, itinerary generation jobs, AI template adaptation jobs, AI repair jobs, budgets, workspace budgets, cost analytics, cost splitting, approval workflow, workspace policies, approval risk scoring, comments, activity, version history, templates, conflict detection, notifications integration, and permissions
  - supports personal trips and workspace trips
  - calls AI Planning Service for generation/regeneration/optimization/adaptation/repair
- Worker Service:
  - Go microservice
  - processes RabbitMQ-backed jobs
- Notification Service:
  - Go microservice
  - owns in-app/email/web-push/SSE notifications and notification preferences
- External Integrations Service:
  - Go microservice
  - owns places, routes, weather, calendar, exchange rates, prices, availability provider adapters, quota/rate limits
- AI Planning Service:
  - Python FastAPI service
  - supports itinerary generation, partial regeneration, budget optimization, template adaptation, policy-aware repair, destination context/RAG, validation/repair, multilingual output, and Ollama/mock modes
- Web App:
  - Next.js app under apps/web
  - supports auth, trips, workspace switcher, workspace pages, templates, AI template adaptation, budgets, workspace budgets, cost analytics, cost splitting, approval workflow, workspace policies, approval risk scoring, AI repair proposals, availability cards, exports, offline mode, PWA install, notifications, internationalization, etc.
- Infra:
  - Postgres
  - RabbitMQ
  - Prometheus/Grafana

Current create trip behavior:

- The app supports a normal form-based create trip flow.
- This works when the user already knows the destination.
- The page feels boring and does not help users who do not know where they want to go.

Problem:
Many users do not start with a fixed destination. They start with a vague idea:

- “I want a cheap weekend trip.”
- “I want something warm with good food.”
- “I want mountains and not too much walking.”
- “Surprise me.”
- “Something like my Prague trip, but new.”
- “Find me a 4-day trip in September under €700.”

Goal:
Add an AI-powered discovery flow:

- User can choose between:
  1. “I know where I want to go” → existing form
  2. “Help me choose” → new AI discovery experience
- User can write a natural-language prompt.
- User can use quick chips.
- User can press a smart “Surprise me” button.
- Trip Service combines user prompt, preferences, previous trips, language, budget, season, workspace policy, and existing trips.
- AI Planning Service suggests 3–5 destination ideas.
- Each suggestion explains why it fits, possible downsides, estimated budget, best duration, and trip preview.
- User can refine suggestions:
  - cheaper
  - warmer
  - more nature
  - more city
  - less walking
  - different country
  - similar places
  - not this vibe
- User selects a destination.
- App creates a draft trip from the suggestion and optionally starts itinerary generation.
- Do not automatically create a trip from “Surprise me” without user confirmation.

Do NOT add:

- real flight search
- real hotel booking
- automatic booking
- visa/legal guarantees
- health/safety guarantees
- payments
- full ML ranking model
- destination marketplace
- public destination database admin
- external travel recommendation APIs in v1
- complex conversational memory engine
- native mobile
- Kubernetes
- new backend service

For v1:

- Implement AI destination suggestions using existing AI Planning Service.
- Implement backend orchestration in Trip Service.
- Use existing user profile/preferences and previous trips.
- Use mock mode for deterministic tests.
- Store discovery sessions/suggestions if useful.
- Build a polished Web App experience.
- Keep existing create trip form working.
- English fallback must work.
- Support existing selected language/outputLanguage.
- Workspace policy should guide suggestions when creating workspace trips.
- User must confirm before a trip is created.

Important codebase consistency requirement:
Before implementing, inspect existing services and follow the same patterns exactly:

- apps/web
- services/trip-service
- services/user-service
- services/ai-planning-service
- services/worker-service
- services/external-integrations-service
- services/notification-service

Do not invent a different architecture if the repository already has conventions.

Match existing patterns for:

- Next.js routing/app router
- layouts/providers
- i18n
- frontend API clients/hooks
- TanStack Query
- forms/validation
- toasts/error handling
- trip creation
- trip generation jobs
- workspace permission checks
- workspace policy constraints
- user profile/preference fetching
- Go service modules
- Uber Fx modules
- Zap logging
- config loading
- HTTP middleware
- auth/JWT middleware
- response/error helpers
- pgxpool/sqlc
- migrations
- sqlc queries
- AI request clients
- FastAPI route structure
- Pydantic schemas
- Ollama/mock mode
- AI prompt builder
- tests
- smoke scripts
- docs

Part 1: Product flow

1. Update Create Trip page.

The create trip page should offer two modes:

Mode A:
“I know where I want to go”

- Existing form-based flow.

Mode B:
“Help me choose”

- New AI Trip Discovery flow.

Recommended hero copy:

- “Where should we go next?”
- “Describe your ideal trip and we’ll suggest destinations that fit you.”

Input examples:

- “A cheap 3-day trip with good food and warm weather.”
- “Mountains, nature, and not too much walking.”
- “Something romantic for a long weekend.”
- “A city break similar to Prague but new.”

2. Discovery actions.

Support:

- Prompt-based discovery.
- Smart Surprise Me.
- Refine suggestions.
- Create trip from selected suggestion.

3. User confirmation.

Never create a trip immediately after pressing “Surprise me.”
Flow:
Surprise me → show destination suggestion(s) → user confirms → create trip.

Part 2: AI Planning Service endpoint

4. Add endpoint:

POST /suggest-destinations

Request:

{
"prompt": "I want a cheap 3-day trip somewhere warm with good food.",
"mode": "prompt" | "surprise" | "refine",
"outputLanguage": "en" | "es" | "uk" | "fr",
"userContext": {
"homeCity": "Bratislava",
"homeCountry": "Slovakia",
"preferredCurrency": "EUR",
"preferredLanguage": "uk",
"preferences": {
"travelStyles": ["food", "city_break"],
"pace": "balanced",
"maxWalkingKmPerDay": 10,
"foodPreferences": ["local food"],
"avoid": ["nightclubs"],
"preferredTransport": ["train", "public_transport"]
}
},
"tripContext": {
"durationDays": 3,
"startDate": "2026-09-10",
"dateFlexibility": "flexible",
"budget": {
"amount": 700,
"currency": "EUR"
},
"travelers": 2,
"origin": "Bratislava, Slovakia",
"scope": "personal" | "workspace"
},
"previousTrips": [
{
"destination": "Prague",
"country": "Czechia",
"durationDays": 3,
"budget": {
"amount": 450,
"currency": "EUR"
},
"tags": ["city", "food", "architecture"],
"likedSignals": ["walkable city", "good food"],
"createdAt": "2026-05-12"
}
],
"workspacePolicyConstraints": {
"summary": "Keep total budget under 700 EUR. Avoid late activities after 22:00.",
"rules": {}
},
"refinement": {
"previousSuggestions": [],
"selectedSuggestionId": "optional",
"instruction": "Cheaper and more nature."
},
"constraints": {
"suggestionCount": 5,
"avoidPreviouslyVisited": true,
"preferNovelty": true,
"includeReasoning": true,
"maxTravelComplexity": "medium"
}
}

Response:

{
"sessionTitle": "Warm budget food trips",
"suggestions": [
{
"id": "stable-id-or-generated",
"destination": "Valencia, Spain",
"city": "Valencia",
"country": "Spain",
"region": "Valencian Community",
"matchScore": 87,
"recommendedDurationDays": 4,
"bestFor": ["food", "architecture", "warm weather"],
"estimatedBudget": {
"amount": 520,
"currency": "EUR",
"confidence": "medium"
},
"bestTimeToGo": "Spring or early autumn",
"whyItFits": "You like walkable city trips with strong food culture and moderate budgets.",
"possibleDownsides": [
"Can be hot in August.",
"Flights or long train connections may affect budget."
],
"tripPreview": {
"title": "Valencia food and architecture escape",
"summary": "A relaxed city break with markets, old town walks, paella, and beach time.",
"sampleDay": [
"Central Market and old town walk",
"Turia Gardens",
"Paella dinner"
]
},
"tags": ["food", "city_break", "warm", "architecture"],
"suggestedPromptForItinerary": "Create a 4-day Valencia food and architecture trip with relaxed pace and a 520 EUR budget.",
"concerns": [
{
"type": "budget_uncertainty",
"message": "Transport cost from your origin is not verified."
}
]
}
],
"followUpQuestions": [
"Do you prefer beach cities or historic cities?"
],
"warnings": [
"Budgets are rough estimates and do not include live flight/hotel prices."
]
}

5. Pydantic schemas.

Create:

- DestinationSuggestionRequest
- DestinationSuggestionMode
- DestinationUserContext
- DestinationTripContext
- PreviousTripSummary
- DestinationRefinementContext
- DestinationSuggestionResponse
- DestinationSuggestion
- DestinationBudgetEstimate
- DestinationTripPreview
- DestinationConcern

6. Modes.

Support:

- prompt
- surprise
- refine

Prompt mode:

- Use user prompt heavily.

Surprise mode:

- If prompt is empty, use user preferences, previous trips, and novelty.
- Should produce smart-random suggestions, not random city names.
- Add some variety while staying plausible.

Refine mode:

- Use previous suggestions and refinement instruction.
- Avoid repeating rejected suggestion unless asking for similar places.

7. Mock mode.

Mock mode must be deterministic and language-aware.

For prompt mode:

- Return fixed suggestions based on prompt keywords:
  - warm/food → Valencia, Naples, Lisbon
  - mountains/nature → Salzburg, Ljubljana, Innsbruck
  - cheap/weekend → Kraków, Budapest, Brno
  - museums/culture → Vienna, Paris, Florence
  - beach → Valencia, Nice, Split

For surprise mode:

- Use user preferences and previous trips:
  - If previous trip includes Prague, suggest Vienna/Kraków/Ljubljana but avoid Prague.
  - If user likes food, include Valencia/Naples/Lisbon.
  - If user likes nature, include Salzburg/Ljubljana.
- Deterministic ordering.

For refine mode:

- If instruction includes cheaper, return cheaper alternatives.
- If warmer, return warmer alternatives.
- If nature, return more nature-heavy alternatives.
- If city, return city-break alternatives.

8. Ollama mode.

Add prompt builder for destination suggestions.

The prompt must instruct:

- Return strict JSON only.
- Do not claim real-time prices or availability.
- Use rough estimates only.
- Consider user preferences and previous trips.
- Avoid suggesting the same destination if avoidPreviouslyVisited is true.
- Explain why each suggestion fits.
- Include possible downsides.
- Include suggestedPromptForItinerary.
- Keep JSON keys/enums in English.
- Localize user-facing text values to outputLanguage.
- Avoid unsafe or illegal travel suggestions.
- Do not provide visa/legal/health guarantees.

9. Language behavior.

Use outputLanguage:

- User-facing text values should be localized.
- destination/city/country names can use common names for that language where natural.
- JSON keys/enums stay English.

Part 3: Trip Service data model

10. Decide whether to persist discovery sessions.

Recommended v1:
Persist sessions and suggestions so user can refine, revisit, and create from suggestion.

Add migration:

trip_discovery_sessions:

- id UUID primary key
- user_id UUID not null
- workspace_id UUID null
- mode TEXT not null
- prompt TEXT null
- output_language TEXT not null default 'en'
- status TEXT not null default 'completed'
- request_json JSONB not null
- response_json JSONB not null
- created_trip_id UUID null
- created_at TIMESTAMP not null default NOW()
- updated_at TIMESTAMP not null default NOW()

Constraints:

- mode in ('prompt', 'surprise', 'refine')
- status in ('completed', 'failed', 'created_trip')
- output_language in ('en','es','uk','fr')

Indexes:

- user_id, created_at desc
- workspace_id, created_at desc
- created_trip_id

Optional:
trip_discovery_feedback:

- id UUID primary key
- session_id UUID not null
- suggestion_id TEXT not null
- user_id UUID not null
- feedback_type TEXT not null
- feedback_text TEXT null
- created_at TIMESTAMP not null default NOW()

feedback_type:

- not_for_me
- too_expensive
- too_far
- too_much_walking
- warmer
- colder
- more_nature
- more_city
- similar
- accepted

If feedback table is too much for v1:

- store refinement history in request_json.

11. SQL/sqlc queries.

Add:

- CreateTripDiscoverySession
- GetTripDiscoverySessionByID
- ListTripDiscoverySessionsByUser
- MarkTripDiscoverySessionCreatedTrip
- CreateTripDiscoveryFeedback optional
- ListRecentDiscoverySessions optional

Part 4: Trip Service discovery module

12. Add module.

Create:

services/trip-service/internal/tripdiscovery/

Suggested files:

- types.go
- dto.go
- service.go
- handler.go
- repository.go
- ai_client.go
- context_builder.go
- previous_trips.go
- module.go
- errors.go

Adjust to repo conventions.

13. Endpoints.

Add:

POST /trip-discovery/suggestions
POST /trip-discovery/surprise-me
POST /trip-discovery/{sessionId}/refine
POST /trip-discovery/{sessionId}/suggestions/{suggestionId}/create-trip
GET /trip-discovery/sessions
GET /trip-discovery/sessions/{sessionId}

Alternative:
Use one endpoint with mode.
But explicit endpoints are easier for frontend.

14. POST /trip-discovery/suggestions.

Request:

{
"prompt": "I want a cheap 3-day trip somewhere warm with good food.",
"scope": "personal" | "workspace",
"workspaceId": "uuid-or-null",
"durationDays": 3,
"startDate": "2026-09-10",
"dateFlexibility": "flexible",
"budget": {
"amount": 700,
"currency": "EUR"
},
"travelers": 2,
"origin": "Bratislava, Slovakia",
"quickChips": ["warm", "food", "low_budget"],
"outputLanguage": "uk",
"avoidPreviouslyVisited": true,
"preferNovelty": true
}

Validation:

- prompt optional but required for suggestions endpoint unless quickChips present.
- prompt max 1000.
- durationDays optional 1–30.
- startDate optional valid date.
- budget optional amount >= 0 currency 3 letters.
- travelers optional 1–50.
- quickChips max 20.
- outputLanguage supported.
- workspaceId required if scope=workspace.

15. POST /trip-discovery/surprise-me.

Request:

{
"scope": "personal" | "workspace",
"workspaceId": "uuid-or-null",
"durationDays": 3,
"startDate": null,
"budget": {
"amount": 500,
"currency": "EUR"
},
"travelers": 1,
"origin": "Bratislava, Slovakia",
"outputLanguage": "en",
"noveltyLevel": "balanced"
}

noveltyLevel:

- safe
- balanced
- adventurous

Behavior:

- No prompt required.
- Build suggestions from profile/preferences/previous trips.
- Return suggestions; do not create trip.

16. POST /trip-discovery/{sessionId}/refine.

Request:

{
"instruction": "Make it cheaper and more nature-focused.",
"selectedSuggestionId": "valencia-spain",
"feedbackType": "too_expensive",
"outputLanguage": "uk"
}

Validation:

- instruction required, max 1000.
- feedbackType optional enum.
- selectedSuggestionId optional.

Behavior:

- Load previous session.
- Check owner/access.
- Build refine request using previous suggestions.
- Create new discovery session linked to previous session if schema supports.
- Return new suggestions.

Optional DB:

- add parent_session_id UUID null to trip_discovery_sessions.

Recommended:

- add parent_session_id.

17. POST /trip-discovery/{sessionId}/suggestions/{suggestionId}/create-trip.

Request:

{
"title": "Valencia food weekend",
"startDate": "2026-09-10",
"durationDays": 4,
"budget": {
"amount": 520,
"currency": "EUR"
},
"travelers": 2,
"workspaceId": null,
"autoGenerateItinerary": true
}

Behavior:

1. Load session and suggestion.
2. Check user owns session.
3. Check workspace permission if workspaceId provided.
4. Create draft trip with:
   - destination from suggestion
   - title from request or suggestion tripPreview title
   - startDate/duration/budget/travelers
   - language from session/outputLanguage/user preference
   - source metadata:
     {
     "source": "trip_discovery",
     "sessionId": "...",
     "suggestionId": "...",
     "matchScore": 87
     }
5. If autoGenerateItinerary:
   - create generation job using suggestion.suggestedPromptForItinerary
   - include destination and trip context
   - return trip + job
6. Mark session status created_trip and created_trip_id.

Response:

{
"trip": {...},
"generationJob": {...}
}

18. GET sessions.

GET /trip-discovery/sessions?limit=20

Return recent sessions for current user.

19. Permissions.

Personal discovery:

- authenticated user only.

Workspace discovery:

- user must be active workspace member.
- to create workspace trip from suggestion, user must have role owner/admin/member.
- viewer can maybe generate suggestions but cannot create workspace trip.
  Recommended:
- viewer can view/use discovery read-only? Simpler:
  - viewer cannot create discovery for workspace.
  - member/admin/owner can.

20. User context builder.

Trip Service should gather:

- user profile
- user preferences
- preferred language/currency
- recent trips
- previous destinations
- previous trip durations/budgets/tags
- liked templates if available
- workspace policy if workspace scope
- origin/homeCity

Limit previous trips:

- last 10–20 trips.
- Do not send full itineraries.
- Send summaries only.

21. Previous trip summary.

Build:
{
"destination": "Prague",
"country": "Czechia",
"durationDays": 3,
"budget": {"amount": 450, "currency": "EUR"},
"tags": ["city", "food", "architecture"],
"pace": "balanced",
"createdAt": "..."
}

Do not send:

- comments
- collaborators
- share tokens
- calendar sync IDs
- raw provider data
- private notes
- full itinerary unless summarized.

22. Workspace policy constraints.

For workspace scope:

- fetch active workspace policy.
- convert to AI constraints using existing policy helper.
- include in AI request.

23. Output language.

Determine:

1. request.outputLanguage if provided.
2. user preferredLanguage.
3. en.

Created trip should store language if trip.language exists.

Part 5: Trip Service AI client

24. Extend AI client.

Add:
SuggestDestinations(ctx, request) response.

Use existing:

- base URL
- timeout
- logging
- error handling
- retries if any

Config:

- TRIP_DISCOVERY_ENABLED=true
- TRIP_DISCOVERY_AI_TIMEOUT_SECONDS=120
- TRIP_DISCOVERY_MAX_PREVIOUS_TRIPS=15
- TRIP_DISCOVERY_DEFAULT_SUGGESTION_COUNT=5

25. Error handling.

If AI fails:

- return controlled error:
  - trip_discovery_failed
- In mock/local mode, should not fail.

Do not create session with empty invalid suggestions unless status failed is useful.
Recommended:

- Store failed session only if existing pattern stores failed job/session.

Part 6: Create trip from suggestion

26. Trip source metadata.

Add to trip metadata or existing column:
{
"creationSource": "trip_discovery",
"discoverySessionId": "uuid",
"discoverySuggestionId": "string",
"discoveryMatchScore": 87,
"discoveryPrompt": "I want a cheap 3-day trip..."
}

Do not expose full previous trip context in trip metadata.

27. Generation prompt.

When autoGenerateItinerary is true:

- Use suggestion.suggestedPromptForItinerary.
- Include original user prompt/refinement as additional context.
- Include destination, duration, budget, travelers, language.
- Include workspace policy if workspace trip.

28. Activity events.

Add:

- trip_discovery_suggestions_created
- trip_created_from_discovery
- trip_discovery_refined

Activity metadata should be safe:

- sessionId
- suggestionId
- destination
- matchScore
- mode

Do not store full prompt in activity if it may include sensitive data; either omit or truncate.

29. Notifications.

No notifications needed for personal discovery.
For workspace trip created from discovery, use existing trip created notifications if any.

Part 7: Web App route and UI

30. Update route.

Existing:
apps/web/app/trips/new/page.tsx

Refactor to show two creation modes:

- Known destination
- Help me choose

Do not remove existing form.

31. New components.

Create:

apps/web/components/trip-discovery/TripCreateModeSelector.tsx
apps/web/components/trip-discovery/TripDiscoveryHero.tsx
apps/web/components/trip-discovery/TripDiscoveryPromptBox.tsx
apps/web/components/trip-discovery/TripDiscoveryQuickChips.tsx
apps/web/components/trip-discovery/SurpriseMeButton.tsx
apps/web/components/trip-discovery/DestinationSuggestionCard.tsx
apps/web/components/trip-discovery/DestinationSuggestionsGrid.tsx
apps/web/components/trip-discovery/TripDiscoveryRefineBar.tsx
apps/web/components/trip-discovery/CreateTripFromSuggestionDialog.tsx
apps/web/components/trip-discovery/DiscoverySessionHistory.tsx optional

32. Visual design.

The Help Me Choose screen should feel like an inspiration page, not a form.

Layout:

- large hero card
- prompt input
- quick chips
- surprise button
- suggestions as rich cards
- refine bar after suggestions

Example UI:

Title:
“Where should we go next?”

Subtitle:
“Describe your ideal trip, or let AI surprise you based on your preferences.”

Prompt placeholder:
“E.g. A cheap 3-day trip with warm weather, good food, and not too much walking…”

Quick chips:

- Weekend
- Warm weather
- Mountains
- Food trip
- Museums
- Low budget
- No flights
- Hidden gem
- Nature
- City break
- Romantic
- Family friendly
- Less walking

Buttons:

- Get suggestions
- Surprise me

33. Suggestion card content.

Each card should show:

- destination
- country
- match score
- tags
- estimated budget
- recommended duration
- why it fits
- possible downsides
- sample day/trip preview
- buttons:
  - Use this destination
  - Show similar
  - Not this vibe

34. Refine actions.

Provide quick refine buttons:

- Cheaper
- Warmer
- More nature
- More city
- Less walking
- Different country
- Similar places
- More hidden gem
- Better for food
- Better for museums

And free text:
“Tell us what to change…”

35. Create trip dialog.

When user clicks “Use this destination”:

- show dialog:
  - title
  - destination prefilled
  - startDate
  - durationDays
  - budget
  - travelers
  - scope/workspace
  - autoGenerateItinerary checkbox default true
- confirm creates trip.

After success:

- navigate to trip detail.
- If generation job started, show existing generation status UI.

36. Surprise Me UX.

Button behavior:

- if no preferences exist, still works but asks optional lightweight context:
  - budget
  - duration
  - origin
- if preferences exist, call surprise endpoint.
- show loading state:
  “Finding places that fit your travel style…”

37. Empty states.

If no suggestions:

- show friendly message.
- suggest changing budget/duration/prompt.
- offer normal create form.

38. Error states.

If AI fails:

- localized error.
- retry button.
- fallback suggestions optional from mock/static list only in local/dev.
- do not create trip.

39. Internationalization.

Add translation keys for all new UI in:

- en
- es
- uk
- fr

Namespace:
tripDiscovery

Include:

- hero title
- subtitles
- buttons
- chips
- card labels
- refine labels
- errors
- loading states
- create dialog labels

40. Accessibility.

- Prompt textarea has label.
- Buttons keyboard accessible.
- Suggestion cards have semantic headings.
- Match score not color-only.
- Loading states announced if existing pattern supports.

Part 8: Web App API/types/hooks

41. Types.

Create:

apps/web/types/trip-discovery.ts

Types:

- TripDiscoveryMode
- TripDiscoverySuggestion
- TripDiscoverySession
- TripDiscoveryRequest
- SurpriseMeRequest
- RefineDiscoveryRequest
- CreateTripFromSuggestionRequest
- TripDiscoveryResponse

42. API client.

Create:

apps/web/lib/api/trip-discovery.ts

Functions:

- getTripDiscoverySuggestions(input)
- surpriseMe(input)
- refineTripDiscovery(sessionId, input)
- createTripFromSuggestion(sessionId, suggestionId, input)
- listTripDiscoverySessions()
- getTripDiscoverySession(sessionId)

43. Hooks.

Create:

apps/web/hooks/useTripDiscoverySuggestions.ts
apps/web/hooks/useSurpriseMe.ts
apps/web/hooks/useRefineTripDiscovery.ts
apps/web/hooks/useCreateTripFromSuggestion.ts
apps/web/hooks/useTripDiscoverySessions.ts

Use TanStack Query/mutations:

- suggestions as mutation
- surprise as mutation
- refine as mutation
- create trip as mutation
- sessions as query optional

Invalidate:

- trips list after create.
- discovery sessions after create/refine.

Part 9: AI ranking/product rules

44. Match score.

AI returns matchScore 0–100.
Backend should clamp to 0–100.
Do not treat it as scientific.

UI label:
“Match score”
Tooltip:
“Estimated fit based on your prompt, preferences, and past trips.”

45. Budget estimate.

Budget is rough.
UI disclaimer:
“Estimated budget does not include live flight or hotel prices.”

46. Avoid repeated destinations.

If avoidPreviouslyVisited true:

- previous destinations should be discouraged.
- AI may still suggest similar but different destinations.

47. Novelty.

Surprise mode should balance:

- preference fit
- novelty
- feasibility
- budget
- travel complexity

Novelty levels:

- safe: close to known preferences
- balanced: mix familiar and new
- adventurous: more unusual suggestions

48. Bad suggestion recovery.

Every card should allow:

- Not this vibe
- Similar
- Cheaper
- Warmer
- More nature
  This is critical for trust.

Part 10: Backend tests

49. AI Planning Service tests.

Test:

- prompt mode returns valid suggestions in mock mode.
- surprise mode avoids previous destination.
- refine mode changes suggestions based on instruction.
- outputLanguage uk returns Ukrainian user-facing text.
- JSON keys/enums remain English.
- unsupported mode rejected.
- unsupported language rejected.
- prompt builder includes user preferences and previous trip summary but not private data.

50. Trip Service endpoint tests.

Test:

- authenticated user can request prompt suggestions.
- prompt too long rejected.
- unsupported outputLanguage rejected.
- surprise-me works without prompt.
- refine requires session ownership.
- non-owner cannot access/refine session.
- create trip from suggestion works.
- create trip does not happen during surprise-me.
- workspace viewer cannot create workspace trip from suggestion.
- workspace member can create workspace trip from suggestion.
- created trip stores discovery metadata.
- autoGenerateItinerary creates generation job.
- no autoGenerateItinerary creates draft only.
- created trip language matches outputLanguage/user preference.

51. Context builder tests.

Test:

- previous trips summarized and limited.
- private fields not included.
- workspace policy constraints included for workspace scope.
- user preferences included.
- default language/currency applied.

52. Repository tests.

Test:

- create session.
- get session by owner.
- list sessions.
- mark created trip.
- parent/refine session if implemented.

Part 11: Frontend tests

53. Component tests.

TripCreateModeSelector:

- switches between known destination and help me choose.

TripDiscoveryPromptBox:

- validates prompt.
- submits quick chips.

SurpriseMeButton:

- calls mutation.
- loading state.

DestinationSuggestionCard:

- renders destination, score, budget, tags, why, downsides.
- actions call callbacks.

TripDiscoveryRefineBar:

- quick refine buttons send expected instruction.
- free text works.

CreateTripFromSuggestionDialog:

- prefilled from suggestion.
- validates fields.
- submits create request.

54. Hook/API tests.

- suggestions mutation calls endpoint.
- surprise mutation calls endpoint.
- refine mutation calls endpoint.
- create trip mutation invalidates trips and navigates.

55. i18n tests.

- trip discovery UI renders in English, Spanish, Ukrainian, French.
- missing key falls back to English.

Part 12: Smoke tests

56. Update scripts/smoke-test.sh.

API smoke:

1. Login user.
2. Update profile preferences.
3. Create one previous trip to Prague.
4. POST /trip-discovery/surprise-me.
5. Assert suggestions returned.
6. Assert Prague is not first suggestion when avoidPreviouslyVisited true.
7. POST /trip-discovery/suggestions with prompt “cheap warm food weekend.”
8. Assert suggestions returned.
9. POST refine with “cheaper and more nature.”
10. Assert new session/suggestions returned.
11. POST create-trip from suggestion with autoGenerateItinerary=false.
12. Assert trip created with discovery metadata.
13. POST create-trip from another suggestion with autoGenerateItinerary=true.
14. Assert generation job created.
15. Try accessing another user’s session and assert forbidden.

16. Update scripts/web-smoke-test.md.

Manual test:

1. Open /trips/new.
2. Confirm two modes: known destination and help me choose.
3. Use prompt: “cheap 3-day warm food trip.”
4. Confirm suggestions appear as cards.
5. Click “Not this vibe” or “Cheaper.”
6. Confirm refined suggestions appear.
7. Click Surprise me.
8. Confirm no trip is created automatically.
9. Choose a suggestion.
10. Create trip with auto-generate enabled.
11. Confirm trip page opens and generation starts.
12. Switch UI language to Ukrainian and repeat; confirm UI and AI suggestion text are Ukrainian.

Part 13: Documentation

58. Update AI Planning Service README.

Document:

- /suggest-destinations endpoint
- request/response schema
- modes
- mock behavior
- language behavior
- limitations

59. Update Trip Service README.

Document:

- trip discovery endpoints
- discovery session model
- previous trip context rules
- create trip from suggestion
- permissions
- workspace behavior
- metadata
- limitations

60. Update Web App README.

Document:

- new Create Trip page modes
- prompt-based discovery
- Surprise Me
- refinement loop
- create trip from suggestion
- i18n keys

61. Update root README.md.

Mention:

- AI Trip Discovery v1.

62. User-facing limitations.

Document:

- suggestions are AI-generated estimates.
- budgets do not include live flight/hotel prices.
- destinations may not always be perfect.
- user should review before creating/generating itinerary.
- no booking is performed.
- no visa/legal/health guarantee.
- previous trips are summarized for personalization, not fully analyzed.

Part 14: Security and privacy

- Backend must enforce user/session ownership.
- Workspace permissions must be enforced.
- Do not send private comments/collaborators/share tokens/calendar IDs to AI.
- Do not send full previous itineraries unless summarized and sanitized.
- Do not log full prompts in production if they may contain sensitive data.
- Do not create trips automatically from Surprise Me.
- Do not claim real-time availability/prices.
- Do not call external travel booking APIs in v1.
- Do not expose discovery sessions to other users.
- Existing create trip flow must not regress.
- Existing AI generation flow must not regress.
- Existing i18n behavior must not regress.
- Keep code consistent with existing service patterns.
- Keep tests and docs updated.

Expected output:
The Create Trip page has a polished “Help me choose” AI discovery mode in addition to the existing known-destination form.
Users can enter a natural-language prompt, use quick chips, press Surprise Me, refine bad suggestions, and create a trip from a selected suggestion.
Trip Service orchestrates discovery using user preferences, previous trips, workspace policy, language, and AI Planning Service.
AI Planning Service exposes `/suggest-destinations` with prompt/surprise/refine modes and deterministic mock behavior.
Suggestions include destination cards with match score, budget estimate, why it fits, downsides, preview, and creation prompt.
No trip is created until the user confirms a suggestion.
Created trips store discovery metadata and can optionally start itinerary generation.
Docs, tests, and smoke tests are updated.

# 6. Add Multi-Destination & Multi-Modal Travel Planning v1: support trips with multiple stops, transfer legs between destinations, transport modes such as car/train/bus/flight/boat/bike/hiking, route builder UI, AI generation for transfer days, budget/route estimates, and map display

You are a senior full-stack engineer and product-minded AI architect. Continue building the web-based AI travel planning application.

Your task:
Implement Multi-Destination & Multi-Modal Travel Planning v1: support trips with multiple stops, transfer legs between destinations, transport modes such as car/train/bus/flight/boat/bike/hiking, route builder UI, AI generation for transfer days, budget/route estimates, and map display.

Context:
We already have a microservices-based AI travel planning app.

Existing services:

- Auth Service:
  - Go microservice
  - issues JWT access tokens and refresh tokens
- User/Profile Service:
  - Go microservice
  - stores user profile/preferences
  - stores preferredLanguage
  - owns workspace membership and workspace roles
- Trip Service:
  - Go microservice
  - owns trips, workspace trips, trip creation, itinerary generation jobs, AI trip discovery, AI template adaptation jobs, AI repair jobs, budgets, workspace budgets, cost analytics, cost splitting, approval workflow, workspace policies, approval risk scoring, comments, activity, version history, templates, conflict detection, notifications integration, and permissions
  - supports personal trips and workspace trips
  - calls AI Planning Service for generation/regeneration/optimization/adaptation/repair/discovery
- Worker Service:
  - Go microservice
  - processes RabbitMQ-backed jobs
- Notification Service:
  - Go microservice
  - owns in-app/email/web-push/SSE notifications and notification preferences
- External Integrations Service:
  - Go microservice
  - owns places, routes, weather, calendar, exchange rates, prices, availability provider adapters, quota/rate limits
- AI Planning Service:
  - Python FastAPI service
  - supports itinerary generation, partial regeneration, budget optimization, template adaptation, policy-aware repair, trip discovery, destination context/RAG, validation/repair, multilingual output, and Ollama/mock modes
- Web App:
  - Next.js app under apps/web
  - supports auth, trips, workspace switcher, workspace pages, trip discovery, templates, AI template adaptation, budgets, workspace budgets, cost analytics, cost splitting, approval workflow, workspace policies, approval risk scoring, AI repair proposals, availability cards, exports, offline mode, PWA install, notifications, internationalization, etc.
- Infra:
  - Postgres
  - RabbitMQ
  - Prometheus/Grafana

Current behavior:

- Trips are mostly modeled as one destination/city.
- Itinerary days contain activities inside one city.
- Route/walking estimates mostly assume walking inside a single destination.
- Create trip flow assumes the user provides one destination.
- AI Trip Discovery can suggest destinations but should now be able to suggest route-style trips too.

Problem:
Real travel is often multi-destination and multi-modal:

- Bratislava → Vienna → Salzburg → Hallstatt
- Barcelona → Valencia → Madrid
- Paris → Brussels → Amsterdam
- Tokyo → Kyoto → Osaka
- Road trips with several towns
- Train/backpacking routes
- Camping/hiking trips
- Island hopping by ferry/boat

The app needs to support:

- multiple stops/towns/cities in one trip
- transfer days between stops
- transport modes beyond walking
- car/train/bus/flight/boat/bike/hiking/public transport
- camping/hiking/adventure trip styles
- route-level budget and timing estimates

Goal:
Add Multi-Destination & Multi-Modal Travel Planning v1:

- A trip can have one or more route stops.
- Backward compatibility: old/single-city trips still work.
- Multi-destination trips include transfer legs between stops.
- Users can build/reorder route stops in the create trip page.
- Users can specify transport preferences and trip style.
- AI generation understands route stops and transfer days.
- Itinerary can contain transfer items/legs.
- Budget summary includes transfer costs.
- Map view shows route stops and transfer lines.
- Route validation warns about unrealistic routes.
- AI Trip Discovery can suggest multi-city routes, not only single destinations.

Do NOT add:

- real ticket booking
- train ticket purchase
- live flight search/prices
- hotel booking
- car rental checkout
- boat rental checkout
- camping permit booking
- advanced GPS hiking route generation
- turn-by-turn navigation
- offline navigation
- complex route optimization across countries
- legal/visa guarantees
- new backend service
- Kubernetes

For v1:

- Support structured route stops and transfer legs.
- Support planned transport modes.
- Use mock/estimated transfer durations and costs if real route provider does not support mode.
- Keep real provider integration optional/fail-open.
- Use AI to plan realistic transfer days.
- Keep user in control.
- Existing trips must not break.
- Single-destination trips should be represented as route with one stop internally if practical.
- Multi-destination trips should work with budgets, exports, map, approval/risk/policy as much as practical.

Important codebase consistency requirement:
Before implementing, inspect existing services and follow the same patterns exactly:

- services/trip-service
- services/ai-planning-service
- services/external-integrations-service
- services/worker-service
- services/user-service
- services/notification-service
- apps/web

Do not invent a different architecture if the repository already has conventions.

Match existing patterns for:

- migrations
- sqlc
- pgxpool
- Go modules
- Uber Fx
- Zap logging
- config/env
- HTTP handlers
- middleware
- JWT/trip/workspace permission checks
- response/error helpers
- itinerary schema validation
- version history
- activity events
- budget summary
- map view
- route estimate client
- AI request building
- FastAPI schemas/routes
- Ollama/mock mode
- i18n
- frontend API clients/hooks
- TanStack Query
- forms
- UI components
- tests
- smoke scripts
- docs

Part 1: Core domain model

1. Add route model to trips.

A trip should support:

{
"route": {
"origin": {
"name": "Bratislava",
"country": "Slovakia",
"coordinates": {
"lat": 48.1486,
"lng": 17.1077
}
},
"returnToOrigin": false,
"stops": [
{
"id": "stop_1",
"destination": "Vienna",
"city": "Vienna",
"country": "Austria",
"arrivalDate": "2026-09-10",
"departureDate": "2026-09-12",
"nights": 2,
"coordinates": {
"lat": 48.2082,
"lng": 16.3738
},
"accommodationHint": "hotel",
"notes": null
},
{
"id": "stop_2",
"destination": "Salzburg",
"city": "Salzburg",
"country": "Austria",
"arrivalDate": "2026-09-12",
"departureDate": "2026-09-14",
"nights": 2,
"coordinates": {
"lat": 47.8095,
"lng": 13.0550
},
"accommodationHint": "guesthouse",
"notes": null
}
],
"legs": [
{
"id": "leg_1",
"fromStopId": "origin",
"toStopId": "stop_1",
"fromName": "Bratislava",
"toName": "Vienna",
"mode": "train",
"departureDate": "2026-09-10",
"estimatedDurationMinutes": 70,
"estimatedDistanceKm": 80,
"estimatedCost": {
"amount": 18,
"currency": "EUR",
"category": "transport",
"confidence": "medium",
"source": "ai"
},
"notes": "Direct regional train recommended.",
"providerMetadata": null
}
],
"preferences": {
"preferredModes": ["train", "public_transport"],
"avoidModes": ["flight"],
"carAvailable": false,
"maxTransferHoursPerDay": 4,
"tripStyles": ["train_trip", "city_break"]
}
}
}

2. Backward compatibility.

Existing single-city trips:

- continue working with existing destination field.
- route can be null.
- OR route is auto-derived as one stop.
  Recommended:
- Add nullable route_json JSONB column to trips.
- Keep existing destination field.
- New multi-destination trips set route_json.
- Existing APIs include route if present.
- UI treats route null as single-destination.

3. Migration.

Trip Service migration:
ALTER TABLE trips ADD COLUMN route_json JSONB NULL;

Optional:
ALTER TABLE trips ADD COLUMN trip_type TEXT NOT NULL DEFAULT 'single_destination';

Allowed trip_type:

- single_destination
- multi_destination

If adding trip_type:

- existing trips default single_destination.
- multi-stop trips set multi_destination.

4. Route validation.

Validate:

- stops length 1–20.
- stop destination required.
- arrivalDate/departureDate valid if present.
- departureDate >= arrivalDate.
- nights >= 0.
- legs connect valid stop IDs or origin.
- supported transport mode.
- maxTransferHoursPerDay reasonable, 1–24.
- coordinates optional but if present lat/lng valid.
- tripStyles supported.
- route dates should not contradict trip start/end/duration if those exist.

5. Transport mode enum.

Supported v1 modes:

- walk
- car
- rental_car
- train
- bus
- flight
- boat
- ferry
- bike
- public_transport
- hiking
- other

6. Trip style enum.

Supported v1 trip styles:

- city_break
- road_trip
- train_trip
- backpacking
- camping
- hiking
- island_hopping
- nature
- beach
- food
- culture
- adventure
- family
- romantic
- low_budget
- luxury
- hidden_gem

7. Accommodation hint enum.

Supported:

- hotel
- hostel
- apartment
- guesthouse
- campsite
- cabin
- campervan
- home
- other
- unknown

Part 2: Itinerary schema updates

8. Support transfer itinerary items.

Add or extend itinerary item type:

- transfer

Transfer item shape:

{
"type": "transfer",
"name": "Train from Vienna to Salzburg",
"description": "Travel from Vienna to Salzburg by train, then check in and take a relaxed evening walk.",
"startTime": "09:30",
"endTime": "12:00",
"transfer": {
"legId": "leg_2",
"from": "Vienna",
"to": "Salzburg",
"mode": "train",
"estimatedDurationMinutes": 150,
"estimatedDistanceKm": 295,
"estimatedCost": {
"amount": 35,
"currency": "EUR",
"category": "transport",
"confidence": "medium",
"source": "ai"
},
"bookingRequired": false,
"notes": "Check train times before travel."
},
"estimatedCost": {
"amount": 35,
"currency": "EUR",
"category": "transport",
"confidence": "medium",
"source": "ai"
}
}

9. Day location.

Each itinerary day should optionally include:

- primaryStopId
- locationName
- transferDay boolean

Example:

{
"dayNumber": 3,
"date": "2026-09-12",
"title": "Transfer to Salzburg",
"primaryStopId": "stop_2",
"locationName": "Salzburg",
"transferDay": true,
"items": [...]
}

10. Existing itinerary validation.

Update validators to accept:

- transfer item type
- transfer object
- transport modes
- estimatedCost category transport
- day.primaryStopId/locationName/transferDay if present

Do not break old itinerary items.

Part 3: Trip Service API

11. Update trip create/update DTOs.

Trip creation should accept either:

- destination for single-city trip
- route for multi-destination trip

Request example:

{
"tripType": "multi_destination",
"title": "Austria train route",
"destination": "Austria",
"startDate": "2026-09-10",
"days": 5,
"budget": {
"amount": 900,
"currency": "EUR"
},
"travelers": 2,
"route": {...}
}

Rules:

- If tripType=single_destination, destination required.
- If tripType=multi_destination, route.stops required.
- destination can be derived as “Austria route” or first/primary stop.
- trips list should display route summary for multi-destination trips.

12. Add route endpoints.

Add endpoints:

GET /trips/{tripId}/route
PUT /trips/{tripId}/route

PUT request:
{
"expectedItineraryRevision": 12,
"route": {...}
}

Permissions:

- owner/editor can update route.
- viewer read-only.
- public share read-only if route included in public trip.

Behavior:

- Updating route is material if itinerary exists.
- Should increment itineraryRevision only if route is stored as part of itinerary? Recommended:
  - route update should increment routeRevision or update trip metadata but also mark itinerary potentially stale.
  - Simpler v1: route update increments itineraryRevision only if itinerary depends on route.
- If approval pending/approved, reset to draft if route materially changes.
- Create activity event route_updated.
- Expire stale repair proposals if needed.

13. Public share behavior.

Public trip share may include sanitized route:

- origin name
- stops
- legs
- transport modes
- durations/costs if itinerary exposes them

Do not expose:

- private notes
- internal provider metadata
- user IDs
- workspace policy metadata

14. Version history.

Itinerary versions should include route snapshot if route affects itinerary.
If existing version only stores itinerary JSON, add metadata routeSnapshot if useful.

Part 4: Transfer estimates / External Integrations

15. Extend route estimate endpoint.

Existing:
POST /routes/estimate

Extend request to support modes:

- car
- train
- bus
- flight
- boat/ferry
- bike
- hiking
- public_transport

Request:
{
"from": {
"name": "Vienna",
"lat": 48.2082,
"lng": 16.3738
},
"to": {
"name": "Salzburg",
"lat": 47.8095,
"lng": 13.0550
},
"mode": "train",
"date": "2026-09-12",
"currency": "EUR"
}

Response:
{
"mode": "train",
"estimatedDistanceKm": 295,
"estimatedDurationMinutes": 150,
"estimatedCost": {
"amount": 35,
"currency": "EUR",
"category": "transport",
"confidence": "low",
"source": "mock"
},
"provider": "mock",
"fallbackUsed": true,
"warnings": [
"This is an estimate, not a live schedule."
]
}

16. Mock estimator.

Implement deterministic estimates:

- walk: distance / 5 km/h
- bike: distance / 15 km/h
- hiking: distance / 3.5 km/h plus terrain warning
- car/rental_car: distance / 80 km/h plus 20 min buffer
- bus: distance / 60 km/h plus 30 min buffer
- train: distance / 100 km/h plus 20 min buffer
- flight: fixed airport overhead 180 min + flight distance / 700 km/h
- ferry/boat: distance / 35 km/h plus 30 min buffer
- public_transport: distance / 35 km/h plus 30 min buffer
- other: distance / 50 km/h

Cost estimates:

- walk: 0
- bike: low/0 if user-owned, else estimate
- hiking: 0
- car/rental_car: fuel estimate distance \* 0.18 EUR/km, rental_car add optional daily estimate if existing
- bus: distance \* 0.08 EUR/km
- train: distance \* 0.12 EUR/km
- flight: max(50, distance \* 0.15 EUR/km)
- ferry/boat: distance \* 0.20 EUR/km
- public_transport: distance \* 0.10 EUR/km
- boat rental should be an activity/accommodation-style estimate, not ordinary transfer booking in v1.

17. Real provider support.

If existing OpenRouteService supports only walking/driving/cycling:

- map car/rental_car to driving.
- bike to cycling.
- walk/hiking to walking if safe.
- train/bus/flight/ferry should use mock estimate in v1 with warning.
- Do not pretend live schedules exist.

18. Trip Service route estimation client.

Trip Service should be able to:

- estimate all route legs
- update route_json legs with estimates
- fail open with warnings if provider fails

Config:

- MULTI_DESTINATION_ENABLED=true
- ROUTE_LEG_ESTIMATION_ENABLED=true
- ROUTE_LEG_ESTIMATION_FAIL_OPEN=true
- ROUTE_LEG_MAX_STOPS=20
- ROUTE_LEG_TIMEOUT_SECONDS=8

Part 5: AI Planning Service updates

19. Update generation schemas.

Add route context to generation requests:

- route
- transport preferences
- trip styles
- max transfer time
- camping/hiking preferences

Request excerpt:
{
"route": {...},
"transportPreferences": {
"preferredModes": ["train"],
"avoidModes": ["flight"],
"carAvailable": false,
"maxTransferHoursPerDay": 4
},
"tripStyles": ["train_trip", "city_break"]
}

20. AI prompt rules.

Prompt should instruct:

- Plan across all route stops.
- Respect arrival/departure dates and nights per stop.
- Include transfer items on transfer days.
- Do not schedule dense sightseeing before/after long transfers.
- Use selected transport modes.
- Avoid disallowed transport modes.
- Add realistic rest after long travel.
- For camping trips, include campsite/accommodation-style notes but do not claim reservations.
- For hiking trips, include conservative day planning and safety notes, but do not generate technical GPS routes.
- For boat/ferry/island hopping, include transfer estimates as approximate and warn to verify schedules.
- Keep costs as estimates.
- Do not claim tickets/bookings are confirmed.
- Output in requested language.

21. Mock generation behavior.

For route with multiple stops:

- Generate days assigned to stops.
- Insert transfer item when day changes stop.
- Use route leg mode in transfer item.
- Produce deterministic simple itinerary.

Example:
Day 1: Arrival in Vienna
Day 2: Explore Vienna
Day 3: Transfer Vienna → Salzburg by train + Salzburg evening
Day 4: Explore Salzburg
Day 5: Return / relaxed final day

22. Template adaptation / repair / discovery.

Update AI Planning Service schemas/prompts:

- Template adaptation can adapt templates into route-style trips if route provided.
- Policy-aware repair can repair route-related issues.
- Trip discovery can suggest route suggestions, not only single destinations.

Part 6: AI Trip Discovery integration

23. Support route suggestions.

Extend destination suggestion response with optional route:

{
"suggestionType": "single_destination" | "route",
"destination": "Austria train route",
"route": {
"origin": {...},
"stops": [...],
"legs": [...],
"preferences": {...}
}
}

24. Discovery cards.

For route suggestions, cards should show:

- route title
- stops sequence
- transport mode
- estimated total route duration
- estimated transfer cost
- why it fits
- downsides
- “Use this route”

25. Create trip from route suggestion.

When user selects route suggestion:

- create multi_destination trip
- store route_json
- optionally auto-generate itinerary

Part 7: Budget integration

26. Transport cost category.

Ensure estimatedCost supports:

- category: transport

Transfer legs should contribute to:

- trip budget summary
- cost analytics
- workspace budget summary
- cost splitting if travelers > 1

27. Avoid double counting.

If a transfer leg has estimatedCost and itinerary transfer item also has estimatedCost:

- decide one source of truth.
  Recommended:
- Itinerary transfer item estimatedCost is included in budget summary.
- Route leg cost is used to prefill transfer item and route display.
- Avoid counting route leg separately if equivalent transfer item exists.

28. Accommodation/camping.

Camping as accommodation style:

- campsite accommodation cost may be stored in accommodation model or itinerary estimatedCost.
- Do not implement full campsite booking.
- Budget can include campsite cost as accommodation if user adds it.

Part 8: Policy/risk/repair integration

29. Workspace policy additions.

Extend workspace policy rules optionally:

- maxTransferHoursPerDay
- disallowedTransportModes
- preferredTransportModes already exists
- requireCarAvailableForRoadTrip optional
- maxTransportBudget optional

If too much for v1:

- add only:
  - maxTransferHoursPerDay
  - disallowedTransportModes

30. Policy evaluator.

Evaluate:

- transfer legs exceeding maxTransferHoursPerDay.
- disallowed transport modes used.
- flight used when disallowed.
- transport budget over threshold if implemented.

31. Approval risk scoring.

Add risk factors:

- too_many_stops_for_duration
- long_transfer_day
- disallowed_transport_mode
- route_estimate_missing
- high_transport_cost
- hiking_day_too_dense
- camping_accommodation_missing

32. AI repair.

Repair can suggest:

- remove a stop
- change transport mode
- add rest after transfer
- reduce route complexity
- replace flight with train if feasible
- split long transfer across days

Do not auto-apply.

Part 9: Web App route builder UI

33. Create trip page.

Update /trips/new.

Modes:

- Single destination
- Multi-destination route
- Help me choose

Multi-destination route builder:

- origin input
- stops list
- add stop
- remove stop
- reorder stops
- nights per stop
- arrival/departure date optional
- transport mode per leg
- route preferences

34. Components.

Create:

apps/web/components/routes/TripRouteBuilder.tsx
apps/web/components/routes/RouteStopCard.tsx
apps/web/components/routes/RouteLegCard.tsx
apps/web/components/routes/TransportModeSelector.tsx
apps/web/components/routes/TripStyleSelector.tsx
apps/web/components/routes/RouteSummaryCard.tsx
apps/web/components/routes/RouteValidationWarnings.tsx

35. Transport mode selector.

Show modes:

- Walking
- Car
- Rental car
- Train
- Bus
- Flight
- Ferry/boat
- Bike
- Hiking
- Public transport
- Other

Use icons if existing icon system supports it.

36. Trip style selector.

Chips:

- Road trip
- Train trip
- Backpacking
- Camping
- Hiking
- Island hopping
- Nature
- Beach
- Food
- Culture
- Adventure
- Family
- Romantic
- Low budget
- Hidden gem

37. Route validation UI.

Warn:

- “5 stops in 3 days may feel rushed.”
- “This transfer is longer than your max transfer time.”
- “Flight is selected but flights are in your avoid list.”
- “Camping selected, but no campsite/accommodation stop is configured.”
- “Hiking selected; routes are approximate and should be checked with local maps.”

38. Create trip dialog/submit.

When submitting:

- validate route.
- create trip with tripType=multi_destination.
- optionally auto-generate itinerary.

Part 10: Trip detail UI

39. Route overview panel.

Add route overview on trip detail:

- route sequence
- stops with dates/nights
- legs with mode/duration/cost
- warnings
- edit route button for editors

40. Itinerary UI.

Transfer items should render differently:

- transport icon
- from → to
- mode
- estimated duration
- estimated cost
- warning “Verify schedule before travel”

41. Map view.

Update map to show:

- markers for stops
- numbered stop order
- lines between stops
- existing activity markers if coordinates exist

If exact route geometry unavailable:

- draw straight lines with dashed/approx style.
- label as approximate.

42. Route editing.

Editors can update route.
If itinerary exists:

- show warning:
  “Changing the route may make the current itinerary outdated. Regenerate affected days after saving.”

V1 can update route without automatically rewriting itinerary.

Part 11: Frontend API/types/hooks

43. Types.

Create/update:

apps/web/types/route.ts

Types:

- TripRoute
- RouteStop
- RouteLeg
- TransportMode
- TripStyle
- AccommodationHint
- RouteValidationWarning

Update trip types to include:

- tripType
- route

44. API client.

Create/update:
apps/web/lib/api/trip-routes.ts

Functions:

- getTripRoute(tripId)
- updateTripRoute(tripId, input)

Update createTrip API to accept tripType/route.

45. Hooks.

Create:

- useTripRoute
- useUpdateTripRoute

Update:

- useCreateTrip
- useGenerateTrip
- useTrip
- useTripList if route summary displayed

Part 12: Exports/calendar/offline

46. PDF export.

Include:

- route overview
- stops
- transfer legs
- transport warnings

47. CSV export.

Include transfer items and transport mode/cost columns.

48. ICS export.

Transfer items should become calendar events if timed.
Title:
“Transfer: Vienna → Salzburg”
Description includes mode/duration/warnings.

49. Offline mode.

Cached trip should include route_json.
Offline editing route can be disabled in v1 unless existing offline mutation architecture supports it.

Part 13: Internationalization

50. Add translation keys.

Namespaces:

- routes
- transportModes
- tripStyles

Translate to:

- en
- es
- uk
- fr

Keys:

- Multi-destination route
- Add stop
- Remove stop
- Reorder
- Origin
- Stop
- Transfer
- Transport mode
- Road trip
- Train trip
- Camping
- Hiking
- Ferry/boat
- Estimated duration
- Estimated cost
- Verify schedules before travel
- Route may be unrealistic

Part 14: Backend tests

51. Trip Service tests.

Test:

- create single destination trip still works.
- create multi-destination trip with route works.
- invalid route rejected.
- unsupported transport mode rejected.
- get/update route permissions.
- viewer cannot update route.
- public share route sanitized.
- route update resets approval if needed.
- route update creates activity.
- generation request includes route.
- existing trip APIs do not break when route null.

52. Itinerary validation tests.

Test:

- transfer item accepted.
- invalid transfer mode rejected.
- transfer estimatedCost category transport accepted.
- old itinerary items still accepted.

53. Budget tests.

Test:

- transfer item costs included in budget.
- route leg cost not double-counted if matching transfer item exists.
- transport category appears in analytics.

54. Policy/risk tests.

Test:

- disallowed transport mode violation.
- max transfer hours violation.
- risk factor for long transfer.
- personal trip behavior unchanged.

55. External Integration tests.

Test:

- route estimate for car/train/bus/flight/ferry returns deterministic mock.
- unsupported mode rejected.
- ORS provider maps car/bike/walk where available.
- train/flight/ferry fallback to mock with warning.

56. AI Planning Service tests.

Test:

- mock generation with route produces transfer days.
- route prompt includes stops/legs/modes.
- no booking claims.
- multilingual output still works.
- old single-destination generation still works.

Part 15: Frontend tests

57. Route builder tests.

Test:

- add/remove/reorder stop.
- select transport mode.
- select trip style.
- route validation warnings.
- submit multi-destination create request.

58. Trip detail tests.

Test:

- route overview renders stops/legs.
- transfer item card renders mode/from/to/duration/cost.
- map receives route stops/lines.
- edit route button hidden for viewer.

59. Discovery tests.

Test:

- route suggestion card renders stop sequence.
- create trip from route suggestion sends route.

60. i18n tests.

Test route/transport/trip style labels in all four languages.

Part 16: Smoke tests

61. Update scripts/smoke-test.sh.

API smoke:

1. Login user.
2. Create multi-destination trip:
   - origin Bratislava
   - stops Vienna and Salzburg
   - train leg
3. Assert trip route exists.
4. Trigger itinerary generation in mock mode.
5. Assert itinerary has transfer item.
6. Assert budget summary includes transport cost.
7. Update route with car leg.
8. Assert activity created and route updated.
9. Create single-destination trip and assert old flow still works.
10. Request route estimate for train/bus/flight/ferry and assert response/warnings.

11. Update scripts/web-smoke-test.md.

Manual test:

1. Open /trips/new.
2. Select Multi-destination route.
3. Add origin and 3 stops.
4. Select train/car transport modes.
5. Select trip styles: train trip + hiking.
6. Create trip and auto-generate itinerary.
7. Confirm route overview appears.
8. Confirm transfer day appears.
9. Confirm map shows stop markers and route lines.
10. Confirm budget includes transfer cost.
11. Edit route and confirm warning about outdated itinerary.
12. Confirm single-destination create still works.

Part 17: Documentation

63. Update Trip Service README.

Document:

- route_json schema
- trip_type
- route endpoints
- transfer itinerary items
- budget behavior
- policy/risk integration
- public share sanitization
- limitations

64. Update External Integrations README.

Document:

- route estimate modes
- mock estimator rules
- provider fallback
- warnings for non-live schedules

65. Update AI Planning Service README.

Document:

- route context in generation requests
- transfer day behavior
- camping/hiking constraints
- no booking/live schedule claims

66. Update Web App README.

Document:

- multi-destination route builder
- transport mode selector
- trip style selector
- route overview/map
- limitations

67. Update root README.md.

Mention:

- Multi-Destination & Multi-Modal Travel Planning v1.

68. User-facing limitations.

Document:

- transport durations/costs are estimates.
- no live train/bus/flight/ferry schedules in v1.
- no booking or ticket purchase.
- hiking/camping suggestions require user verification.
- route lines on map may be approximate.
- changing route does not automatically rewrite the whole itinerary unless user regenerates.

Part 18: Security and quality requirements

- Existing single-destination trips must not break.
- Route update must enforce trip edit permissions.
- Public share must sanitize route metadata.
- Do not expose internal provider metadata.
- Do not claim live schedules/prices unless real provider explicitly supports them.
- Do not book transport/accommodation.
- Do not generate technical hiking navigation or safety guarantees.
- AI output must be validated before saving.
- Transport cost must avoid double counting.
- Approval/risk/policy behavior must remain consistent.
- Keep code consistent with existing service patterns.
- Keep tests and docs updated.

Expected output:
Trips can be single-destination or multi-destination.
Multi-destination trips store route stops, transfer legs, transport modes, and trip styles.
Create Trip page includes a route builder with stops, transport modes, and route validation warnings.
AI generation can create transfer days and multi-stop itineraries.
External Integrations can estimate transfer durations/costs for supported modes with mock/fallback behavior.
Trip detail shows route overview, transfer items, and route map.
Budget, policy, approval risk, exports, discovery, and public share support route data where practical.
Existing single-city trips continue working.
Docs, tests, and smoke tests are updated.
