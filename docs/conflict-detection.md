# Conflict Detection

Conflict detection runs in two places:

- Frontend: `detectScheduleConflicts` powers visible banners and warning badges.
- Backend: `validateAndNormalizeItinerary` rejects invalid scheduled writes before version creation.

Detected blocking cases include invalid time formats, end-before-start, overlapping scheduled items, duplicate schedule/title pairs, invalid duration, and scheduled items without a time.

Detected warning cases include explicit `NeedsReview`, very short transfers, missing travel time hints between placed activities, and unusually long scheduled days.

Existing AI validation already covers route, transport, opening-hours, weather, budget, accommodation, and policy feasibility. Manual itinerary saves continue through Trip Service so revision conflicts, permissions, audit/activity, version history, and notifications remain authoritative on the backend.
