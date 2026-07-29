# Alpha error catalog

User-facing errors should show a short safe code and a support/reference ID when the route has one. Never expose stack traces, SQL, provider credentials, raw prompts, raw responses, or internal tokens.

| Safe code | User message | Retryable | Fallback eligible | Ops action | Escalate when |
| --- | --- | --- | --- | --- | --- |
| `ai_provider_timeout` | AI generation timed out. The trip is saved and can be retried. | Yes | Yes | Retry job or switch to fallback | More than 3 in 15 minutes |
| `ai_provider_rate_limited` | AI is busy. Try again later. | Yes | Yes | Reduce concurrency, use fallback | Alert fires or retries loop |
| `ai_provider_quota_exceeded` | AI quota is reached for now. | No | Yes | Disable OpenAI, route fallback | Any real alpha occurrence |
| `ai_provider_budget_exceeded` | AI budget limit is reached. | No | Yes | Disable AI/OpenAI, review spend | Any occurrence |
| `ai_provider_invalid_response` | AI returned an unusable response. | Yes | Yes | Review AI trace and retry simpler | Repair rate high |
| `ai_provider_repair_failed` | AI output could not be repaired safely. | Yes | Maybe | Retry simpler or mark failed | Repeated same model/prompt |
| `generation_job_stuck` | Generation is taking too long. | Yes | Maybe | Mark stale job failed or retry | Stale count > 0 after recovery |
| `itinerary_conflict` | The itinerary changed. Reload latest before saving. | Yes | No | None unless repeated UX issue | Users lose work |
| `provider_unavailable` | A supporting provider is unavailable. | Yes | Yes | Pause provider or use cached/mock data | Hot trip reads degrade |
| `notification_delivery_failed` | Trip action succeeded but notification failed. | No | No | Check Notification Service and retry notification if safe | Core mutation also failed |
| `public_share_expired` | This share link expired. | No | No | Owner creates a new share | Expiry is wrong |
| `public_share_password_required` | Password is required. | Yes | No | Check share settings, rate limit | Unlock failures spike |
| `service_unavailable` | Service is temporarily unavailable. | Yes | No | Check `/ready`, queues, dependencies | 5xx/ready failure persists |
| `migration_required` | Service version requires a migration. | No | No | Stop rollout, run migration safety review | Any alpha rehearsal |
| `feature_disabled` | This alpha feature is not enabled. | No | No | Verify feature flag profile | Required alpha feature disabled |

Support triage starts with request ID, user-safe timestamp, route, service version, and this safe code. Attach only redacted logs and safe Ops trace summaries.
