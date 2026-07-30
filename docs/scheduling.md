# Scheduling

Timeline & Calendar Planning v1 keeps the itinerary JSON as the single source of truth.
There is no separate schedule table.

Each itinerary item may include:

- `time`: canonical start time used by existing clients.
- `startTime`: compatibility alias for schedule-aware clients. Trip Service normalizes it to `time`.
- `endTime`: optional HH:mm end time.
- `durationMinutes`: optional positive duration.
- `allDay`: optional all-day flag.
- `timezone`: optional display timezone label.
- `schedulingStatus`: `Scheduled`, `Unscheduled`, `Conflict`, or `NeedsReview`.

Scheduled items require a valid HH:mm start time unless `allDay` is true. Unscheduled items remain inside their itinerary day and use `schedulingStatus: "Unscheduled"` with no time fields.

Writes continue through `PUT /trips/{id}/itinerary` with `expectedItineraryRevision`. The server validates the payload, stores a new itinerary version, records activity, and notifies collaborators through the existing itinerary update path.
