# Alpha known issues

Do not include security-sensitive exploit details here. Link private incident records for sensitive details.

| Issue | User impact | Workaround | Severity | Status | Owner |
| --- | --- | --- | --- | --- | --- |
| Browser coverage is Chromium-first | Some WebKit/Firefox defects may be discovered later | Use Chromium for alpha unless release-candidate browser evidence passes | Medium | Open | Frontend |
| Spend-ratio Prometheus metric is optional until ledger export exists | Spend near-threshold alerts may rely on the spend-limit script instead of Prometheus | Run `scripts/ai/check-openai-spend-limit.sh` during go/no-go | Medium | Open | AI platform |
| Non-DB storage restore is documented but not automated by rollback rehearsal | Receipt/export files may need manual restore validation | Keep OCR disabled; verify export storage before invite | Medium | Open | SRE |
| Offline mutations are disabled | Users need network for normal trip edits | Keep `offline_mode_enabled=false` and show online-only state | Low | Accepted for alpha | Frontend |
| Real providers are disabled | Availability/transport/calendar provider-backed flows are not alpha commitments | Use mock/cached data and route basics only | Low | Accepted for alpha | Integrations |
