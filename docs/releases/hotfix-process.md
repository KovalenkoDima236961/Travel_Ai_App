# Hotfix process

A hotfix is allowed for an active security issue, material outage, data integrity risk, or a narrowly scoped regression that cannot wait for the normal train. Use `hotfix/<ticket-or-summary>` from the latest release tag; keep the change minimal and avoid unrelated refactors.

1. State the impact, affected version/image SHA, rollback availability, and owner.
2. Add the smallest focused fix and tests. Use the next PATCH version (for example `0.2.0` to `0.2.1`); a security fix still uses PATCH unless it intentionally breaks compatibility.
3. Update `CHANGELOG.md` Security/Fixed/Migration Notes as applicable. Do not put unresolved exploit details in public notes.
4. Run focused tests plus `./scripts/release/check-release.sh ci`; security scans and fresh migration checks remain required.
5. Generate notes, publish immutable version/SHA image tags, verify staging-like smoke, and tag the release.
6. After stabilization, open follow-up work for root cause, tests/alerts, documentation, and any deferred cleanup; merge the hotfix back to the active development line if needed.
