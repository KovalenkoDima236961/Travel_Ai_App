# Playbook: add a notification type

1. Add the backend type/category mapping and default channel behavior in Notification Service; choose whether it is immediate, grouped, digestible, or urgent.
2. Add preference behavior and any trip-mute interaction. Suppress self-notifications and deduplicate with a stable event key.
3. Send only an action-safe title/message/metadata through the internal batch endpoint. Do not put receipt bytes, raw OCR, secrets, prompts, or private notes in payloads.
4. Add in-app rendering, localized UI text, email/push/digest presentation, and navigation target in the Web App.
5. Test preference disabled, mute, quiet-hours/digest, dedupe, no-self-notification, SSE update, and each enabled delivery adapter using mocks.
6. Update [notifications guide](../../features/notifications.md) and API inventory when an endpoint/payload changes.

Run Notification Service tests and `./scripts/test-frontend.sh`.
