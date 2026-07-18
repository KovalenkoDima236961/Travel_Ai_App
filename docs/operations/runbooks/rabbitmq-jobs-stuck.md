# Runbook: RabbitMQ jobs stuck

1. Confirm queue and worker state: open http://localhost:15672 locally, call
   Worker `/ready`, inspect `worker-service` logs, and inspect Trip Service ops
   job status when authorized.
2. Check the queue, consumer count, unacked messages, retry/dead-letter queues,
   and the job record's correlation ID/status. Compare the message attempt with
   persisted job state—redelivery must be safe.
3. Check dependency failures: RabbitMQ credentials, Trip/AI/Notification
   readiness, stale expected itinerary revision, provider quota, and cancellation.
4. Retry/cancel through protected Trip ops endpoints only after understanding
   whether the job side effect is idempotent. Requeue a DLQ message using the
   protected Worker ops route; never discard it before capturing sanitized
   payload metadata and cause.
5. In local development, restart the worker after correcting config. If queue
   credentials changed with a persistent volume, recreate only confirmed local
   data. In staging/production, escalate sustained queue growth with metrics,
   queue name, correlation IDs, affected job IDs, and recent deployment changes.

Inspect Grafana's worker/RabbitMQ dashboards when `observability` is enabled.
