# Trip Invitations

Trip invitations support registered-user and email-first collaboration.

## Lifecycle

| Status | Meaning |
| --- | --- |
| `pending` | Invitation can be accepted by the invited user or a user whose email matches. |
| `accepted` | Invitation has created or accepted the collaborator membership. |
| `declined` | Invited user declined. |
| `expired` | Invitation passed `expiresAt`. |
| `revoked` | Owner revoked the invitation. |

Default expiration is 14 days. Maximum expiration is 90 days.

## Owner Actions

- Create: `POST /trips/{tripId}/invitations`
- List: `GET /trips/{tripId}/invitations`
- Resend: `POST /trips/{tripId}/invitations/{invitationId}/resend`
- Revoke: `POST /trips/{tripId}/invitations/{invitationId}/revoke`

`message` is optional and capped at 500 characters.

## Invitee Actions

- Accept: `POST /trips/{tripId}/invitations/{invitationId}/accept`
- Decline: `POST /trips/{tripId}/invitations/{invitationId}/decline`

The legacy collaborator invitation endpoints remain available for existing registered-user flows.
