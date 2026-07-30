package entity

import (
	"time"

	"github.com/google/uuid"
)

type CollaboratorRole string

const (
	CollaboratorRoleViewer CollaboratorRole = "viewer"
	CollaboratorRoleEditor CollaboratorRole = "editor"
	// Future-ready roles are accepted by storage but not granted broad editing
	// powers until the permission matrix opts them in.
	CollaboratorRoleCommenter CollaboratorRole = "commenter"
	CollaboratorRoleGuest     CollaboratorRole = "guest"
	CollaboratorRoleAdmin     CollaboratorRole = "admin"
)

type CollaboratorStatus string

const (
	CollaboratorStatusPending  CollaboratorStatus = "pending"
	CollaboratorStatusAccepted CollaboratorStatus = "accepted"
	CollaboratorStatusDeclined CollaboratorStatus = "declined"
	CollaboratorStatusExpired  CollaboratorStatus = "expired"
	CollaboratorStatusRevoked  CollaboratorStatus = "revoked"
	CollaboratorStatusRemoved  CollaboratorStatus = "removed"
)

type TripCollaborator struct {
	ID              uuid.UUID
	TripID          uuid.UUID
	UserID          uuid.UUID
	Email           string
	Role            CollaboratorRole
	Status          CollaboratorStatus
	InvitedByUserID uuid.UUID
	Message         string
	InvitedAt       time.Time
	ExpiresAt       *time.Time
	AcceptedAt      *time.Time
	DeclinedAt      *time.Time
	RevokedAt       *time.Time
	RemovedAt       *time.Time
	LastSeenAt      *time.Time
	Permissions     map[string]any
	UpdatedAt       time.Time
}

type SharedTrip struct {
	Trip         Trip
	Collaborator TripCollaborator
}
