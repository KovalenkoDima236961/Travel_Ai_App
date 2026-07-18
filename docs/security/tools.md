# Security Tools and CI Gates

## Run locally

Install the named CLI tools, then run:

```bash
./scripts/security-scan.sh
./scripts/security-scan.sh --audit       # report without blocking
./scripts/security-scan.sh --zap         # requires the local stack
```

Individual scripts live under `scripts/security/`: `gitleaks.sh`, `gosec.sh`,
`govulncheck.sh`, `bandit.sh`, `pip-audit.sh`, `npm-audit.sh`, `trivy.sh`,
`semgrep.sh`, and `zap-baseline.sh`. `gosec` and `govulncheck` can be installed
with `go install`; Bandit/pip-audit/Semgrep are Python CLIs; Trivy and Gitleaks
are standalone CLIs. CI installs these tools and uses no provider credentials or
Ollama models.

`trivy.sh --image IMAGE` scans an already-built image in addition to the
filesystem. Set `ZAP_TARGET` for a host-reachable target on Linux. ZAP v1 is
unauthenticated by design; do not mistake it for authorization coverage.

## Gate policy

| Tool | Blocking threshold |
| --- | --- |
| Gitleaks | Any unreviewed secret |
| gosec | High severity, medium-or-higher confidence |
| govulncheck | Reachable known Go vulnerability |
| Bandit | High severity and high confidence |
| pip-audit | Any actionable advisory (the tool has no reliable severity filter) |
| npm/pnpm/yarn audit | High or critical advisory |
| Trivy filesystem/image | High or critical vulnerability, secret, or misconfiguration |
| Semgrep | `ERROR` findings from security-audit and OWASP Top Ten rules |
| ZAP baseline | Manual/nightly/release candidate; triage alerts before release |

Medium/low findings are reported and reviewed. High/critical findings must be
fixed, mitigated, or recorded with an owner and expiry in
[accepted-risks.md](accepted-risks.md); they are never silently suppressed.

## False-positive process

1. Confirm the exact tool rule, version, path, and reachability.
2. Fix the code/config if practical.
3. If an exception is justified, add a time-bounded entry to
   `accepted-risks.md` and the narrowest tool config exception with a reason.
4. Link the exception to an issue/owner and remove it at the review date.

`PIP_AUDIT_IGNORE` may only contain IDs documented in the accepted-risk file.
`.gitleaks.toml` permits only example/config/cache paths, not real credentials.
