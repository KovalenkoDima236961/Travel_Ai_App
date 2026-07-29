# Analytics Privacy

Alpha analytics are privacy-first by design. They answer product and safety
questions with the minimum data needed.

## Never Collect

Do not send or store:

- passwords
- tokens, authorization headers, cookies, or secrets
- emails inside analytics payloads
- receipts or OCR text
- raw prompts
- raw AI responses
- generated itineraries
- comments or private notes
- exact locations unless a feature explicitly requires them
- payment card data or passport/phone/SSN-like identifiers

## Sanitization Rules

The backend drops metadata keys matching sensitive terms such as password,
token, secret, authorization, cookie, email, prompt, receipt, OCR, note,
private, query, search, address, latitude, longitude, exact, payment, card, CVV,
SSN, passport, and phone.

Sanitization also:

- limits metadata object depth
- limits key count and array length
- trims long strings
- redacts obvious bearer tokens, OpenAI-style keys, and `password=`/`token=`
  text patterns
- hashes session IDs and entity IDs
- rejects unsupported event names

## Client Rules

Frontend analytics calls should send coarse workflow information only:

- event name
- feature group
- entity type
- entity ID when useful, knowing the backend hashes it
- counts, booleans, status names, job types, timing values

Do not include raw form fields, destination search strings, itinerary text,
prompts, receipts, private notes, or screenshots in analytics metadata.

## Session Replay

Session replay is not implemented in alpha v1. If Microsoft Clarity, PostHog
Replay, or another recorder is added later, it must:

- require explicit product/privacy approval
- mask inputs and forms
- disable admin/Ops pages
- respect consent policy
- disable anonymous recordings when required by policy
- document retention and deletion behavior

## Access and Retention

Analytics endpoints require authenticated users. Ops dashboard and report routes
require existing Ops authorization. Alpha analytics tables should be covered by
the platform data retention policy before public beta.
