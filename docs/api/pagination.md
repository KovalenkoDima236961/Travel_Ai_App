# Pagination and filtering

New browser-facing lists use cursor pagination. Requests accept `limit` and an
opaque `cursor`; responses include `pagination.nextCursor`, `pagination.hasMore`
and the applied `pagination.limit`. The default is 20 and the maximum is 100
unless an endpoint documents a smaller safety limit. A cursor is opaque: never
construct, sort, or persist it as an identifier.

Filters and sorting are explicit query parameters. Dates are inclusive ISO
date/date-time values as documented by the endpoint. A filter/sort change
invalidates the previous cursor.

The existing trip and expense lists use `items`, `limit` and `offset`; the
notification list uses `items` and `nextCursor`. They remain valid v1 legacy
contracts and are represented exactly in the generated types. When an endpoint
moves to the envelope it must accept both during a documented transition or be
released as a deliberate breaking change.
