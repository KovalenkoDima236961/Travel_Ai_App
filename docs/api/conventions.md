# API conventions

These conventions apply to new or deliberately migrated browser-facing
endpoints. Legacy endpoints retain their documented shape until consumers have
migrated.

## JSON and scalar values

- JSON property names use lower camel case.
- Identifiers are canonical RFC 4122 UUID strings unless a public share token
  is explicitly documented instead.
- Timestamps are RFC 3339 UTC strings. Date-only values use `YYYY-MM-DD`.
- Enums are lower snake case unless an established public DTO documents a
  legacy value; clients must not infer enum values from display text.
- Money is an object with a decimal JSON number `amount` and ISO 4217 uppercase
  `currency`. Floating values are estimates, not an instruction to round at
  the client.

## Response and pagination conventions

New single-resource responses use `{ "data": { ... } }`; new lists use
`{ "data": [...], "pagination": { "nextCursor", "hasMore", "limit" } }`.
Existing raw responses (for example `items`/`offset` trip and expense lists)
are explicitly modelled in their v1 spec and are not silently enveloped.
See [pagination](pagination.md) for migration rules.

## Error convention

The target response is:

```json
{
  "error": {
    "code": "validation_error",
    "message": "Invalid request.",
    "details": [{ "field": "startDate", "message": "Start date is required." }],
    "requestId": "..."
  }
}
```

Codes are stable, user-safe lower snake case. Current Go services still have
some legacy `{ "error": "message" }` paths; clients normalize both during the
incremental migration. Never return a stack trace, query text, token, provider
credential, raw prompt or OCR payload. See [errors](errors.md).

## Headers and authentication

- Private browser calls use `Authorization: Bearer <access JWT>`.
- The client sends or propagates `X-Request-ID`; `X-Correlation-ID` may link a
  multi-service operation. Error bodies include `requestId` when the handler
  has it available.
- Internal service calls require `X-Internal-Service-Token` and, where the
  service supports it, its service-name header. They are never browser calls.
- Mutations may accept `Idempotency-Key` or a documented `clientMutationId`
  only when their endpoint declares it. Do not invent a global retry key.

## Conflicts and sanitization

Itinerary mutations carry `expectedItineraryRevision`. A stale revision returns
`409 itinerary_conflict` and may include `currentItineraryRevision`; clients
must refetch and make a conscious merge/retry decision. Public responses use
an explicit public DTO and remove all private/user/security/operational fields.
