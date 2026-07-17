# Onboarding & First-Run Experience v1

## Principles

Onboarding is optional, short, and routes users into existing product flows. It
does not gate navigation, create a second preference store, run an automated
tour, or send analytics. A storage failure must never prevent trip planning.

The browser persists progress under `onboarding:{userId}`. Dismissed feature
tips use `onboardingTipsDismissed:{userId}`, and a dismissed first-trip setup
card uses `tripSetupChecklistDismissed:{userId}:{tripId}`. Every key is scoped
to the authenticated user so accounts sharing a browser do not read each
other's UI state. Profile and travel answers remain authoritative in User
Service through the existing profile and preferences endpoints.

## Entry points

- A user with no trips and active onboarding sees the first-run dashboard on
  `/trips`.
- `/getting-started` provides welcome, preference, choose-start, and completion
  states. It is linked from Settings, the account menu, and Cmd/Ctrl+K.
- Known destination opens `/trips/new?mode=destination`.
- AI discovery opens `/trips/new?mode=discovery`.
- Templates open `/templates?firstRun=true`.
- Multi-destination planning opens `/trips/new?mode=route`.
- `/demo-trip` is a frontend-only, read-only example with sample data and no API
  mutations.

Skipping changes only onboarding UI state. A user with no trips still sees
helpful creation choices. Restarting removes prior onboarding progress on that
device but does not clear saved profile/preferences.

## Preference fields

The four-step wizard uses React Hook Form and Zod. It writes home city/country,
currency, and language to the existing profile. It writes travel styles, pace,
walking limit, food and dietary preferences, transport, accommodation, and the
existing avoid list to preferences. Budget comfort is normalized into the
existing `budget`/`luxury` travel-style signals instead of adding duplicate
storage. Create Trip consumes these saved defaults for currency, language,
pace, walking, transport, origin, and interests. The existing planning context
continues to supply them to discovery and AI generation.

## First-trip setup checklist

The first created trip enters `first_trip_setup`. Its Command Center computes:

- destination and dates from destination, start date, and duration;
- itinerary from saved itinerary days;
- budget from trip budget data;
- route/transport as complete for a single destination, or when every
  multi-destination leg has a mode;
- checklist from the saved checklist/Command Center summary;
- collaborators from group membership (optional for a personal trip);
- Trip Health when health has loaded, with critical issues marked as needing
  attention.

The card links to existing deep-linked sections, disappears after five items
are complete, and can be dismissed per trip. Completing onboarding also hides
it.

## Contextual tips

Tips are inline, dismissible, keyboard-accessible, and rendered only in private
authenticated feature surfaces. Definitions live in
`apps/web/src/lib/onboarding/tips.ts`; copy lives under `onboarding.tips` in all
four message catalogs. To add a tip, add its stable ID and translation key,
translate the message in `en`, `es`, `uk`, and `fr`, then mount
`<ContextualTip tipId="..." />` beside the relevant feature. Never mount private
tips on public share pages.

## Accessibility and i18n

Cards are real links, wizard choices use radio/checkbox semantics, progress has
visible text and `progressbar` attributes, and dismiss controls have accessible
names. The demo announces its read-only state. All new copy must exist in the
English, Spanish, Ukrainian, and French catalogs; do not rely on English
fallback for onboarding strings.
