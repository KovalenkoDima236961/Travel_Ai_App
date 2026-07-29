# OpenAI Cost Controls

AI Planning Service records token metrics from provider responses, but durable
cost accounting belongs in the Trip Service or Ops data store so it can be
queried across app instances.

## Implemented Now

- OpenAI usage fields in safe provider metadata
- Prometheus counters for input and output tokens
- configuration fields for daily/monthly UAH limits, per-user/per-trip limits,
  concurrency, and operation-specific models
- startup validation for positive limit values
- no prompt or output storage in usage metadata

## Required Before Production Alpha

Add a durable usage ledger with:

- provider, model, operation, status, error code
- user, workspace, trip, generation job, request, and correlation identifiers
- provider request ID
- input, output, total, cached-input, and reasoning tokens
- original cost amount and currency
- optional estimated UAH amount, exchange rate, source, and timestamp
- latency, retry count, fallback flag, and creation time

Pricing must be configuration-driven. If pricing is unknown, store usage and
leave cost null. Never invent a price.

## Spend Rejection Policy

When a spend or request limit is exceeded:

- reject the OpenAI invocation before the provider call
- return `ai_provider_budget_exceeded`
- avoid trip mutation
- use configured fallback only when policy allows it
- emit a low-cardinality metric and safe Ops event

The current provider layer exposes configuration and metrics hooks. It does not
yet enforce distributed spend limits because no durable ledger is wired here.

