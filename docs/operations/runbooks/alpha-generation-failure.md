# Alpha generation failure

Use when itinerary generation fails, times out, or returns invalid output.

## Triage

1. Capture time, user-safe report, trip ID if supplied, request ID/correlation ID, service versions, and safe error code.
2. Open Ops Dashboard, Alpha Overview, Jobs, and AI Generations.
3. Check `generation_jobs_status_total`, `ai_provider_errors_total`, fallback rate, repair rate, and Worker queue delay.
4. Confirm the trip still exists and no duplicate itinerary version was created.

## Recovery

1. If the job is failed and `canRetry=true`, use Ops Retry with a reason.
2. If the job is stale running, use Ops Mark failed, then retry if appropriate.
3. If OpenAI errors are widespread, disable `ai_generation_enabled` or route to fallback according to the OpenAI outage runbook.
4. If invalid output recurs for one prompt/model, keep the job failed and escalate to AI platform with the safe trace summary.

## Do not

- Do not edit the database to mark a job complete.
- Do not paste raw prompts, raw itineraries, or provider errors into tickets.
- Do not increase retry count or concurrency during a provider incident.

## Escalate

Escalate when three failures occur in 15 minutes, repair rate exceeds the alpha target, fallback also fails, or any saved itinerary is suspected corrupt.
