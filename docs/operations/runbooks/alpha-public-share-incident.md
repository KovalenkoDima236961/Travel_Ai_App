# Alpha public share incident

Use for leaked links, wrong public content, password/expiry failures, or public-share abuse.

## Triage

1. Record the share token only in a private incident record; do not paste it in public docs.
2. Verify whether the response exposed private budget, expenses, comments, collaborators, activity, provider internals, AI traces, tokens, or share password data.
3. Check request IDs, rate-limit errors, and recent share settings changes.
4. Identify owner and trip ID through authenticated admin tooling only.

## Recovery

1. Disable the specific share through owner UI/API when possible.
2. If the issue is systemic, disable `public_sharing_enabled` in Feature Flags with an audit reason.
3. Rotate `PUBLIC_SHARE_ACCESS_SECRET` if access tokens may be compromised.
4. Re-run public share alpha tests and the sanitizer regression.
5. Issue a new share only after scope/password/expiry is reviewed.

## Do not

- Do not disclose private share tokens in Slack/docs/tickets without restricted access.
- Do not create a replacement link before confirming the old one is disabled.
- Do not weaken public share sanitization to keep alpha moving.

## Escalate

Escalate immediately for private data exposure, token leakage, bypassed password, or rate-limit abuse.
