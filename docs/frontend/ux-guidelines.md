# Frontend UX Guidelines

These guidelines define the v1 usability patterns for the existing Tailwind design system. Shared primitives live in `apps/web/src/components/ui`; feature screens remain responsible for domain copy and permissions.

## Loading

- Use `PageLoadingState` only while the primary page identity is unknown.
- Render the page header as soon as its data exists. Use `SectionLoadingState` or `CardSkeleton` for slower independent queries.
- Give every loading region a localized label. The components expose `role="status"` and `aria-busy`.
- Mutation controls must stay disabled while pending, keep their width where practical, and pair loading text with `ButtonSpinner`.
- Do not block a whole dashboard because one card failed or is still loading.

## Empty states

Use `EmptyState` when an empty collection or missing setup is a valid product state. Supply:

- a concrete title;
- one sentence explaining the value of the missing content;
- one primary next step when the user has permission;
- an optional secondary action only when it is genuinely useful;
- viewer copy instead of a disabled editor action when no action is possible.

If an action must remain visible but disabled, provide `disabledReason`. Do not use an empty state for a failed request.

## Errors

Use `ErrorState` for page or card failures and `InlineError` for a field or mutation inside an otherwise usable surface.

- State what failed in the title.
- Explain what remains safe or available.
- Offer Retry when retrying can help.
- Add a secondary navigation action when the user may be stuck.
- Never pass a raw backend error as the normal-user description. `developmentDetails` is rendered only in development.
- Keep card failures local. A Trip Health failure must not hide route, budget, or itinerary content.

## Confirmation dialogs

Use `ConfirmDialog` for destructive or high-impact actions. Copy must answer:

1. What will happen?
2. What will remain unchanged?
3. Is the action immediate or reversible?
4. What data is affected?

The cancel action receives initial focus. The dialog traps focus, closes on Escape when safe, and returns focus to the trigger. Set `tone="danger"` only for destructive actions. Use `UnsavedChangesDialog` for discarding local form/editor changes.

## Forms

- Keep labels visible; placeholders are examples, not labels.
- Associate hints and errors through `aria-describedby`.
- Set `aria-invalid` only on the invalid field.
- Validate safe constraints before the API call: date order, finite/non-negative amounts, currency code, file type, and file size.
- Preserve entered data after server errors.
- Prevent double submit with the form/mutation pending state.
- Long forms should show `FormErrorSummary` after an unsuccessful submit; each item links to a stable field ID.
- Place currency beside amount and label approximate conversion.

## Mobile actions

Use `StickyMobileActionBar` for multi-step or scrollable edit flows. Keep existing desktop actions visible at `md` and above, and do not render two active action bars at the same viewport. Respect safe-area insets. Complex dialogs may use the existing full-height mobile sheet pattern.

## Microcopy

- Prefer specific verbs: “Generate itinerary,” “Restore version,” “Create share link,” “Sync to Google Calendar.”
- Explain consequences, not implementation details. Never ask a normal user to start a service.
- Use sentence case and short success messages: “Budget saved”, “Receipt uploaded”, “Route changes saved”.
- Provider wording is consistent: prices/schedules are estimates, conversion is approximate, and no result is a booking confirmation.
- AI copy says that suggestions may need review; it does not imply guaranteed correctness.

## Accessibility

- All interactive elements need a visible focus state and a comfortable touch target.
- Dialogs need an accessible name, focus containment, Escape behavior, and focus restoration.
- Status and confidence badges include visible text; color is supplementary.
- Loading regions use polite status semantics; critical errors use alerts.
- Progress bars expose label, min, max, and current value.
- Tabs/navigation expose the current item in text or `aria-current`, not color alone.
- Drag-and-drop ordering must retain the existing keyboard/button fallback.
- Decorative icons are `aria-hidden`; meaningful images have localized alt text.

## Deep links

Trip detail maps `tab` to a stable section. Supported focused targets include:

- `?tab=route&legId=...`
- `?tab=route&stopId=...`
- `?tab=budget&category=food`
- `?tab=health&issueId=...`
- `?tab=expenses&expenseId=...`
- `?tab=activity&eventId=...`

The target ID must be stable, focusable, scroll with header offset, and receive a temporary focus ring. If a requested target does not resolve after progressive data loading, scroll to the parent section and show localized friendly feedback. Preserve query parameters during in-page mutations and refreshes.

## Internationalization

- New user-facing strings belong in `messages/en.json`, `es.json`, `uk.json`, and `fr.json` under matching keys.
- Use `useTranslations(namespace)` in client UI. English remains the runtime fallback, not a substitute for updating the other catalogs.
- User content and provider content are not automatically translated.
- Keep dynamic values in ICU parameters; do not build translatable sentences with string concatenation.
