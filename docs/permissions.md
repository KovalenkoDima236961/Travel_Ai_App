# Collaboration Permissions

Trip Service resolves permissions server-side for every private collaboration action.

## Roles

| Role | View | Comment | Vote | Edit plan | Manage members/share | Delete/transfer |
| --- | --- | --- | --- | --- | --- | --- |
| Owner | yes | yes | yes | yes | yes | yes |
| Editor | yes | yes | yes | yes | no | no |
| Viewer | yes | yes | yes | no | no | no |

The current authorization code maps accepted collaborator roles through `TripAccess`. `viewer` can read private trip data and contribute comments/votes; `editor` can mutate itinerary-backed resources; only `owner` can manage collaborators, share settings, trip deletion, and ownership transfer.

The database check constraint tolerates additional future role strings, but v1 public invite/update validation only accepts `viewer` and `editor`.

## Rules

- Pending, declined, expired, revoked, and removed collaborators do not have private trip access.
- Workspace access is resolved separately and can grant owner/editor/viewer-equivalent access for workspace trips.
- Public-share access is intentionally read-only and sanitized; it does not reuse private collaboration permissions.
- Sensitive member fields are hidden from non-managers.
- Suggestions that mutate itinerary JSON require `expectedItineraryRevision` and fail on stale revisions.
