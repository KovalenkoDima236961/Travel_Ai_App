# Runbook: notifications not sending

1. Confirm the triggering action created an in-app notification row using the
   authenticated `/notifications` API. If not, inspect Trip/Worker internal
   batch call logs and internal-token configuration.
2. Check recipient preferences, trip mute, quiet-hour/digest grouping, dedupe
   key, priority, and self-notification suppression. These may intentionally
   prevent delivery.
3. For email, inspect Notification Service logs and `EMAIL_NOTIFICATIONS_ENABLED`,
   provider mode, SMTP configuration, and the mock-vs-real environment. For
   push, check VAPID settings and active subscription. For SSE, inspect stream
   connection/heartbeat limits and the browser proxy route.
4. Check pending/history digest endpoints and the internal digest-processing
   path when an event should be batched rather than immediate.
5. Do not resend blindly: dedupe protects users. Correct the cause, test with a
   non-sensitive event, and record request/correlation ID. Never log email
   credentials, push keys, or full notification metadata.
