# Alpha AI Quality Gates

These gates define readiness for OpenAI-backed alpha generation. Do not lower
thresholds only to pass a provider.

## Required Gates

| Gate | Target |
| --- | --- |
| Schema validity after bounded repair | >= 99% |
| Privacy sanitizer pass rate | 100% of external AI calls |
| Prohibited private fields sent | 0 known cases |
| Raw prompts or outputs in normal logs | 0 known cases |
| Mock fallback metadata | `fallbackUsed=true`, `qualityStatus=fallback_mock`, `needsReview=true` |
| Unsupported provider configuration | startup failure |
| API key in frontend config or bundle | 0 matches |
| Prompt/model version metadata | present on private AI responses |
| Public share provider internals | excluded |
| OpenAI vs Ollama eval comparison | completed before alpha promotion |

## Provider Quality Metrics

Track per provider and operation:

- schema validity
- repair rate
- grounded-place rate
- hallucinated-place count
- duplicate-place count
- destination mismatch count
- schedule and route plausibility
- budget plausibility
- language quality
- fallback rate
- latency
- input/output tokens
- estimated cost when pricing is configured

## Release Rule

OpenAI can be the alpha primary provider only when mock CI is green, manual
OpenAI evaluation has a documented report, fallback behavior is tested, and Ops
has enough metrics to detect provider failures without reading prompts.

