# Incident Response Starter

1. Record time, reporter, affected environment, suspected asset, and a safe
   reproduction. Do not paste tokens, passwords, OCR, calendar details, raw
   prompts, or customer receipts into tickets/logs.
2. Contain the issue: disable the public share, affected provider, ops route or
   integration feature where appropriate. Preserve minimal redacted evidence.
3. Rotate `JWT_ACCESS_SECRET`, `INTERNAL_SERVICE_TOKEN`/plural rotation set,
   public-share access secret, provider credentials, and calendar encryption
   key as applicable. Follow [config-hardening.md](config-hardening.md) for
   internal-token overlap rotation.
4. Revoke refresh tokens for affected users (or all users for a JWT incident),
   disconnect/revoke calendar connections, invalidate affected exports, and
   delete/quarantine suspect receipt objects. A forced password-reset workflow
   is a product follow-up; until then coordinate user resets through support.
5. Inspect structured logs by request ID and low-cardinality metrics only. Do
   not turn on raw prompt/payload logging to investigate an incident.
6. Triage severity and impact, notify the incident owner, remediate, add a
   regression test/security scan rule, and document the timeline and rotation.

For a leaked public link, disable it immediately and issue a new one only after
confirming its scope/password/expiry. For a leaked internal token, deploy the
overlap token set to receivers, move callers to the new token, verify failures,
then remove the old value.
