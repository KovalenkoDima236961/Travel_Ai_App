# AI Training Data

This directory is for consent-safe, manually curated training examples. It is not a dump of production trips.

Allowed content:
- Synthetic examples created by the project.
- Golden/eval cases that are marked holdout and excluded from training.
- User-derived examples only after explicit consent, sanitizer pass, quality scoring, and human review.

Disallowed content:
- Receipts, OCR text, calendars, comments, private notes, emails, phone numbers, home addresses, tokens, passwords, API keys, raw prompts, hidden system instructions, raw logs, raw provider payloads, and unlicensed provider text.

Use `scripts/ai/validate-training-examples.sh` before importing manual examples.
