# Collaboration API

The authoritative contract is `docs/api/openapi/trip-service.openapi.yaml`.

## Invitations

- `GET /trips/{tripId}/invitations`
- `POST /trips/{tripId}/invitations`
- `POST /trips/{tripId}/invitations/{invitationId}/accept`
- `POST /trips/{tripId}/invitations/{invitationId}/decline`
- `POST /trips/{tripId}/invitations/{invitationId}/resend`
- `POST /trips/{tripId}/invitations/{invitationId}/revoke`

## Members

- `GET /trips/{tripId}/members`
- `POST /trips/{tripId}/members/transfer-ownership`
- `POST /trips/{tripId}/members/leave`

## Comments

- `GET /trips/{tripId}/comments`
- `POST /trips/{tripId}/comments`
- `POST /trips/{tripId}/comments/{commentId}/resolve`
- `POST /trips/{tripId}/comments/{commentId}/reopen`

## Suggestions

- `GET /trips/{tripId}/suggestions`
- `POST /trips/{tripId}/suggestions`
- `POST /trips/{tripId}/suggestions/{suggestionId}/accept`
- `POST /trips/{tripId}/suggestions/{suggestionId}/reject`
- `POST /trips/{tripId}/suggestions/{suggestionId}/resolve`

## Votes

- `GET /trips/{tripId}/votes`
- `POST /trips/{tripId}/votes`
- `DELETE /trips/{tripId}/votes/{voteId}`

Run `npm run contracts:generate` in `apps/web` after contract changes to update `apps/web/src/lib/api/generated/*`.
