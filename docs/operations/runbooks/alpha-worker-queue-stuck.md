# Alpha worker queue stuck

Use when generation jobs remain queued/running, queue delay is high, or the DLQ grows.

## Triage

1. Check Worker `/ready`, RabbitMQ connection, DB connection, active jobs, and service version.
2. Check queue depth, unacked messages, consumers, retry queue, and DLQ in Ops.
3. Inspect recent failed jobs by safe error code and correlation ID.
4. Run `scripts/worker-reliability-smoke-test.sh` only against a mock/test deployment with a known test trip and token.

## Recovery

1. If Worker is down, restart the Worker service and verify `/ready`.
2. If a job is stale running, use Ops Mark failed with a reason.
3. If messages are in DLQ and the root cause is fixed, use Ops Requeue one message first.
4. If DLQ contains a poison job, keep it in DLQ or discard only with an approved reason.
5. If backlog grows, stop inviting users and disable AI generation.

## Do not

- Do not purge queues blindly.
- Do not replay all DLQ messages at once.
- Do not manually update job rows except through an approved repair.

## Escalate

Escalate when p95 queue delay exceeds 5 minutes for 10 minutes, DLQ grows after one requeue, or Worker repeatedly restarts during active jobs.
