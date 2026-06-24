package entity

import (
	"time"

	"github.com/google/uuid"
)

type CollaboratorRole string

const (
	CollaboratorRoleViewer CollaboratorRole = "viewer"
	CollaboratorRoleEditor CollaboratorRole = "editor"
)

type CollaboratorStatus string

const (
	CollaboratorStatusPending  CollaboratorStatus = "pending"
	CollaboratorStatusAccepted CollaboratorStatus = "accepted"
	CollaboratorStatusRemoved  CollaboratorStatus = "removed"
)

type TripCollaborator struct {
	ID              uuid.UUID
	TripID          uuid.UUID
	UserID          uuid.UUID
	Role            CollaboratorRole
	Status          CollaboratorStatus
	InvitedByUserID uuid.UUID
	InvitedAt       time.Time
	AcceptedAt      *time.Time
	RemovedAt       *time.Time
	UpdatedAt       time.Time
}

type SharedTrip struct {
	Trip         Trip
	Collaborator TripCollaborator
}
