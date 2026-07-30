# Timeline API

Timeline & Calendar Planning v1 does not introduce a separate schedule API.

The write contract remains:

```http
PUT /trips/{id}/itinerary
Content-Type: application/json

{
  "expectedItineraryRevision": 12,
  "itinerary": {
    "days": [
      {
        "day": 1,
        "title": "Arrival",
        "items": [
          {
            "time": "09:30",
            "startTime": "09:30",
            "endTime": "11:00",
            "durationMinutes": 90,
            "type": "activity",
            "name": "Museum",
            "schedulingStatus": "Scheduled"
          }
        ]
      }
    ]
  }
}
```

The server validates permissions, expected itinerary revision, schedule consistency, and existing itinerary structure. Successful saves increment `itineraryRevision`, create an itinerary version, record `itinerary_updated`, record `itinerary_schedule_updated` when schedule fields changed, and notify collaborators using the existing itinerary update notification.

Feature flags:

- `agenda_view_enabled`
- `timeline_view_enabled`
- `calendar_view_enabled`
- `timeline_drag_drop_enabled`
- `schedule_conflict_detection_enabled`
