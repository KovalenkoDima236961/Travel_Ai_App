# Personalization v2

Personalization is deterministic, explainable, and optional. It combines the current user's saved profile and preferences, their accessible trip aggregates, explicit lightweight feedback, and workspace policy for the current planning scope.

## Signals and privacy

Used signals are travel styles, pace, walking limit, food/dietary preferences, transport and accommodation preferences, avoid list, safe aggregates of the current user's trips, and feedback chips. We do not use receipt OCR, private expense notes, calendar details, raw prompts, tokens, collaborator history, advertising data, or cross-site tracking.

Feedback accepts a small allowlist of metadata (`destination`, `style`, `transport`, `currency`, `category`, and source). The Trip Service validates trip access before accepting trip-linked feedback. Users can clear their feedback at `DELETE /personalization/feedback`.

## Precedence and explainability

Explicit request fields win over personal signals. Blocking workspace policy wins over both; for example, a flight preference is removed when a workspace policy disallows flights. Planning prompts receive only a privacy-minimized summary, never individual feedback rows or IDs.

Every compatible recommendation can expose factual reasons, concerns, and the signals used. The app never applies personalized changes automatically: users retain normal review and mutation controls.

## Endpoints

- `GET /users/me/preferences/completeness`
- `GET /personalization/context`
- `POST|GET|DELETE /personalization/feedback`
- `GET /personalization/feedback/summary`
- `GET /trip-templates/recommended`
- `GET /trips/{tripId}/budget-suggestion`

The preference completeness score is weighted across location, language/currency, travel style, pace, walking, transport, food/dietary needs, accommodation, and avoid list. It works safely for incomplete profiles.
