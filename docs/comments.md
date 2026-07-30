# Trip Comments

Comments are private collaboration records stored in `itinerary_comments`.

## Targets

`targetType` supports:

- `trip`
- `day`
- `itinerary_item`
- `budget_item`
- `route`
- `attachment`

Existing item comments continue to use `dayNumber` and `itemIndex`. New clients should also send `targetType: "itinerary_item"` and a stable `targetId` such as `1:0` for day 1, item 0.

## Threads

Replies set `parentCommentId`. A reply inherits the parent target server-side so clients cannot move a reply into a different target thread.

## Mentions

Clients can pass `mentionUserIds`. The service validates mentioned users against accepted trip access before creating notifications.

## Resolution

Editors and owners can resolve or reopen comment threads:

- `POST /trips/{tripId}/comments/{commentId}/resolve`
- `POST /trips/{tripId}/comments/{commentId}/reopen`

Resolved comments keep their body and audit history. Delete remains a soft delete and normal list/count endpoints exclude deleted comments.
