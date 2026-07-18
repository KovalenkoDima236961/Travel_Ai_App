# Travel Day Mode v1

Travel Day Mode is the private, mobile-first execution screen at
`/trips/{tripId}/today`. It complements the planner; it does not replace trip
detail editing or introduce location tracking.

## What it shows

- today’s Now / Next activity and a concise timeline;
- selected transport and route context;
- high-signal weather and verification warnings;
- due and overdue checklist items and reminders;
- accommodation, quick expense entry, and the existing receipt flow.

The screen uses the browser’s local calendar date sent explicitly to Trip
Service. It never asks for GPS permission, performs background location
tracking, or runs a navigation engine. Map buttons open an external map site.

## Travel statuses

An itinerary item may be `planned`, `done`, `skipped`, or `delayed`. Existing
items have no stable persisted ID, so v1 stores `travelStatus` in itinerary
JSON. Status updates therefore require `expectedItineraryRevision`, create a
version with source `TRAVEL_STATUS_UPDATED`, and return normal conflict errors.
They create `itinerary_item_status_updated` activity but intentionally do not
reset approval, start AI generation, or send broad notifications. Owners and
editors can update statuses; viewers are read-only for itinerary execution.

## Private API

`GET /trips/{tripId}/travel-day?date=YYYY-MM-DD` is available to accepted
owner/editor/viewer access only. The endpoint fails soft for optional
verification, checklist, reminders, and expenses. It never returns receipt OCR
or calendar event details. Public share routes do not expose it.

`PATCH /trips/{tripId}/itinerary/days/{dayNumber}/items/{itemIndex}/travel-status`
accepts `status`, optional short `note`, and `expectedItineraryRevision`.

## Offline and limitations

Successful reads are saved to user-scoped IndexedDB `cachedTravelDays` records.
When the network fails, the page renders the matching cached day and warns that
data can be stale. Logout and offline-data clearing remove these records.
Offline status changes reuse the existing full-itinerary mutation queue, so
they sync with the normal revision-conflict recovery. Checklist/reminder quick
updates follow their existing queue support. Receipt uploads remain online.

Transport data and prices are not booking confirmations. The app never books,
rebooks, pays, or offers emergency, legal, or medical advice.
