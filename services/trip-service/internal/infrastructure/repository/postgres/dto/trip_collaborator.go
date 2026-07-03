package dto

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/storage/postgres"
)

const TripCollaboratorColumns = "id, trip_id, user_id, role, status, invited_by_user_id, invited_at, accepted_at, removed_at, updated_at"

const TripColumnsWithAlias = "t.id, t.user_id, t.destination, t.start_date, t.days, t.budget_amount, " +
	"t.budget_currency, t.travelers, t.interests, t.pace, t.status, t.itinerary, t.itinerary_revision, t.accommodation, t.workspace_id, t.created_at, t.updated_at"

const TripCollaboratorColumnsWithAlias = "c.id, c.trip_id, c.user_id, c.role, c.status, c.invited_by_user_id, " +
	"c.invited_at, c.accepted_at, c.removed_at, c.updated_at"

func ScanTripCollaborator(row pgx.Row) (*entity.TripCollaborator, error) {
	var (
		id, tripID, userID, invitedByUserID pgtype.UUID
		role, status                        string
		invitedAt, acceptedAt               pgtype.Timestamp
		removedAt, updatedAt                pgtype.Timestamp
	)

	err := row.Scan(
		&id,
		&tripID,
		&userID,
		&role,
		&status,
		&invitedByUserID,
		&invitedAt,
		&acceptedAt,
		&removedAt,
		&updatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip collaborator: %w", err)
	}

	return tripCollaboratorFromScannedValues(
		id,
		tripID,
		userID,
		role,
		status,
		invitedByUserID,
		invitedAt.Time,
		timestampPtr(acceptedAt),
		timestampPtr(removedAt),
		updatedAt.Time,
	), nil
}

func ScanTripCollaboratorRows(rows pgx.Rows) ([]entity.TripCollaborator, error) {
	collaborators := make([]entity.TripCollaborator, 0)
	for rows.Next() {
		collaborator, err := ScanTripCollaborator(rows)
		if err != nil {
			return nil, err
		}
		collaborators = append(collaborators, *collaborator)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip collaborators: %w", err)
	}
	return collaborators, nil
}

func ScanSharedTrip(row pgx.Row) (*entity.SharedTrip, error) {
	var (
		tripID, tripUserID, workspaceID pgtype.UUID
		destination                     string
		startDate                       pgtype.Date
		days                            int32
		budgetAmount                    pgtype.Numeric
		budgetCurrency                  pgtype.Text
		travelers                       pgtype.Int4
		interestsRaw                    []byte
		pace, tripStatus                string
		itineraryRaw                    []byte
		accommodationRaw                []byte
		itineraryRevision               int32
		tripCreatedAt, tripUpdatedAt    pgtype.Timestamp
		id, collaboratorTripID, userID  pgtype.UUID
		role, status                    string
		invitedByUserID                 pgtype.UUID
		invitedAt, acceptedAt           pgtype.Timestamp
		removedAt, updatedAt            pgtype.Timestamp
	)

	err := row.Scan(
		&tripID,
		&tripUserID,
		&destination,
		&startDate,
		&days,
		&budgetAmount,
		&budgetCurrency,
		&travelers,
		&interestsRaw,
		&pace,
		&tripStatus,
		&itineraryRaw,
		&itineraryRevision,
		&accommodationRaw,
		&workspaceID,
		&tripCreatedAt,
		&tripUpdatedAt,
		&id,
		&collaboratorTripID,
		&userID,
		&role,
		&status,
		&invitedByUserID,
		&invitedAt,
		&acceptedAt,
		&removedAt,
		&updatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan shared trip: %w", err)
	}

	interests, err := unmarshalInterests(interestsRaw)
	if err != nil {
		return nil, err
	}

	trip := entity.Trip{
		ID:                uuid.UUID(tripID.Bytes),
		UserID:            fromPgUUID(tripUserID),
		WorkspaceID:       fromPgUUID(workspaceID),
		Destination:       destination,
		StartDate:         fromPgDate(startDate),
		Days:              days,
		BudgetAmount:      fromPgNumeric(budgetAmount),
		BudgetCurrency:    budgetCurrency.String,
		Travelers:         travelers.Int32,
		Interests:         interests,
		Pace:              pace,
		Status:            entity.Status(tripStatus),
		ItineraryRevision: int(itineraryRevision),
		CreatedAt:         tripCreatedAt.Time,
		UpdatedAt:         tripUpdatedAt.Time,
	}
	if len(itineraryRaw) > 0 {
		trip.Itinerary = json.RawMessage(itineraryRaw)
	}
	if len(accommodationRaw) > 0 {
		accommodation, err := unmarshalAccommodation(accommodationRaw)
		if err != nil {
			return nil, err
		}
		trip.Accommodation = accommodation
	}

	collaborator := tripCollaboratorFromScannedValues(
		id,
		collaboratorTripID,
		userID,
		role,
		status,
		invitedByUserID,
		invitedAt.Time,
		timestampPtr(acceptedAt),
		timestampPtr(removedAt),
		updatedAt.Time,
	)

	return &entity.SharedTrip{Trip: trip, Collaborator: *collaborator}, nil
}

func tripCollaboratorFromScannedValues(
	id, tripID, userID pgtype.UUID,
	role, status string,
	invitedByUserID pgtype.UUID,
	invitedAt time.Time,
	acceptedAt *time.Time,
	removedAt *time.Time,
	updatedAt time.Time,
) *entity.TripCollaborator {
	return &entity.TripCollaborator{
		ID:              uuid.UUID(id.Bytes),
		TripID:          uuid.UUID(tripID.Bytes),
		UserID:          uuid.UUID(userID.Bytes),
		Role:            entity.CollaboratorRole(role),
		Status:          entity.CollaboratorStatus(status),
		InvitedByUserID: uuid.UUID(invitedByUserID.Bytes),
		InvitedAt:       invitedAt,
		AcceptedAt:      acceptedAt,
		RemovedAt:       removedAt,
		UpdatedAt:       updatedAt,
	}
}

func timestampPtr(ts pgtype.Timestamp) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}
