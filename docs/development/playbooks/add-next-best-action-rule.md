# Add a next-best-action rule

1. Confirm the source state comes from an authoritative service response already present in `TripCommandCenterInput` or the compact summary.
2. Add a deterministic candidate in `next-best-action.ts` with stable ID, reason, severity, source, capability, priority, and a canonicalizable internal href.
3. Rank blocking/safety/departure-critical work before user decisions, core planning, and optional improvements.
4. Supply a valid viewer alternative or ensure permission filtering chooses one. Never suggest a destructive action.
5. Add ranking tests against adjacent higher/lower rules, viewer behavior, archived/completed lifecycle, and disabled capability.
6. Keep localized wording in the frontend when possible; never use an LLM for v1 ranking.
