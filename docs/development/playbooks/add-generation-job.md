# Playbook: add an AI generation job

1. Define job type, validated input, expected itinerary revision, idempotency/correlation behavior, and terminal states in Trip Service.
2. Create/persist the job before publishing to RabbitMQ. The worker must claim safely, propagate correlation ID, tolerate redelivery, and honor cancellation.
3. Call a bounded AI Planning endpoint that returns strict schema-validated JSON. Recheck permissions and revision before saving results.
4. Define retry/backoff/DLQ policy and whether retry is safe. Record sanitized failure code; never store or log raw sensitive prompts.
5. Publish activity/notification only when useful and respect preferences. Expose status to the Web polling hook.
6. Test queued/running/completed/failed/cancelled, stale revision, retry/idempotency, invalid AI JSON/repair, and worker loss.
7. Update [AI generation guide](../../features/ai-generation.md), endpoint inventory, and observability dashboards if metrics change.

See the [stuck jobs](../../operations/runbooks/rabbitmq-jobs-stuck.md) and [AI failure](../../operations/runbooks/ai-generation-failing.md) runbooks.
