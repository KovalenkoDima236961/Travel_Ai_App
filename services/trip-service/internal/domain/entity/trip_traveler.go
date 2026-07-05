package entity

import (
	"time"

	"github.com/google/uuid"
)

type TripTravelerRole string

const (
	TripTravelerRoleOrganizer TripTravelerRole = "organizer"
	TripTravelerRoleTraveler  TripTravelerRole = "traveler"
)

type TripTravelerStatus string

const (
	TripTravelerStatusActive  TripTravelerStatus = "active"
	TripTravelerStatusRemoved TripTravelerStatus = "removed"
)

// TripTraveler represents a person included in planning cost allocation. It is
// intentionally separate from trip collaborators and does not grant app access.
type TripTraveler struct {
	ID              uuid.UUID
	TripID          uuid.UUID
	Name            string
	Email           *string
	LinkedUserID    *uuid.UUID
	Role            TripTravelerRole
	Status          TripTravelerStatus
	CreatedByUserID uuid.UUID
	CreatedAt       time.Time
	UpdatedAt       time.Time
	RemovedAt       *time.Time
}
