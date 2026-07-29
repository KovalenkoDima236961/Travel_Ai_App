# OpenAI Privacy

OpenAI requests are built server-side in AI Planning Service. The API key is read
from server environment only and must never be placed in `NEXT_PUBLIC_*`,
frontend runtime config, database rows, logs, error responses, or generated
exports.

## Sanitizer

`app/privacy.py` exposes `sanitize_ai_payload`, returning:

- `sanitizedPayload`
- `removedFields`
- `warnings`
- `blocked`
- `sanitizerVersion`

The sanitizer removes sensitive fields such as tokens, authorization headers,
emails, phone numbers, user IDs, cookies, share credentials, receipt/OCR data,
calendar titles/descriptions, comments, private notes, logs, stack traces, and
raw provider payloads. It also redacts sensitive string patterns and neutralizes
prompt-injection phrases.

Removed values are not logged or retained.

## Storage Controls

`OPENAI_STORE_RESPONSES=false` is the default. The provider sends `store=false`
for Responses API calls and does not depend on provider-side conversation state.

Do not claim Zero Data Retention unless the OpenAI organization is actually
configured for it. Some provider features can be incompatible with stricter
retention settings; review the official data-control docs for the account used
by staging or production.

## Allowed Context

Allowed model context includes destination, dates, duration, traveler count,
broad accessibility requirements, budget, travel preferences, pace, transport
preferences, sanitized accommodation context, route stops, trusted grounding
records, weather summaries, workspace policy text, and output language.

## Disallowed Context

Do not send receipts, OCR text, exact home addresses, comments, emails, phone
numbers, tokens, cookies, share passwords, private notes, internal logs, stack
traces, or unnecessary database IDs.

