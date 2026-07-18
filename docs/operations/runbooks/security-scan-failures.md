# Runbook: security scan failures

1. Reproduce locally with `./scripts/security-scan.sh`. Use `--audit` for
   non-blocking triage and `--zap` only when the target is deliberately running.
2. Identify scanner, rule, affected dependency/file/image, severity, exploit
   context, and whether a secret was exposed. Preserve evidence without copying
   secret values into issues.
3. For a real secret, revoke/rotate it immediately through the approved secret
   system, remove it from current code/config, and follow repository incident
   response guidance. Do not merely suppress the finding.
4. Update vulnerable dependencies through normal tested updates. Run affected
   tests and rescans. A false positive or temporarily accepted risk must use the
   documented accepted-risks process with owner, scope, expiry, and mitigation.
5. Do not lower severity thresholds, disable a broad scanner, or commit scan
   credentials to get CI green.

See [security tools](../../security/tools.md) and [accepted risks](../../security/accepted-risks.md).
