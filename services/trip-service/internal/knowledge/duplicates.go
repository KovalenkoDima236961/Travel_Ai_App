package knowledge

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Duplicate resolution is deliberately two-phase: jobs propose groups, humans
// dispose of them. An automatic merge would be hard to reverse once a canonical
// record has been reindexed and referenced by generated itineraries.

// Duplicate group states.
const (
	DuplicateGroupOpen     = "open"
	DuplicateGroupMerged   = "merged"
	DuplicateGroupRejected = "rejected"
	DuplicateGroupSplit    = "split"
)

// DuplicateGroupMember is one place inside a duplicate group.
type DuplicateGroupMember struct {
	PlaceID       uuid.UUID `json:"placeId"`
	CanonicalName string    `json:"canonicalName"`
	Category      string    `json:"category"`
	QualityScore  *float64  `json:"qualityScore,omitempty"`
	ReviewStatus  string    `json:"reviewStatus"`
	Confidence    float64   `json:"confidence"`
	Reason        string    `json:"reason,omitempty"`
}

// DuplicateGroup is a review unit for Ops.
type DuplicateGroup struct {
	ID               uuid.UUID              `json:"id"`
	DestinationID    uuid.UUID              `json:"destinationId"`
	CanonicalPlaceID *uuid.UUID             `json:"canonicalPlaceId,omitempty"`
	Status           string                 `json:"status"`
	Reason           string                 `json:"reason,omitempty"`
	Members          []DuplicateGroupMember `json:"members"`
}

// CreateDuplicateGroup records a proposed duplicate relationship. It is
// idempotent per destination + member set: re-running detection reuses an open
// group instead of creating a second one for the same places.
func (s *Store) CreateDuplicateGroup(ctx context.Context, destinationID uuid.UUID, pair DuplicatePair, reason string) (uuid.UUID, bool, error) {
	if s == nil || s.db == nil {
		return uuid.Nil, false, fmt.Errorf("knowledge store is required")
	}
	left, err := uuid.Parse(pair.PlaceID)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("parse duplicate place id: %w", err)
	}
	right, err := uuid.Parse(pair.OtherPlaceID)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("parse duplicate place id: %w", err)
	}

	transaction, err := s.db.BeginTx(ctx, pgxTxOptions())
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("begin duplicate group transaction: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	// An open group already containing both places is the same finding.
	var existing uuid.UUID
	err = transaction.QueryRow(ctx, `SELECT g.id FROM travel_place_duplicate_groups g
    WHERE g.destination_id = $1 AND g.status = 'open'
      AND EXISTS (SELECT 1 FROM travel_place_duplicate_group_members m WHERE m.group_id = g.id AND m.place_id = $2)
      AND EXISTS (SELECT 1 FROM travel_place_duplicate_group_members m WHERE m.group_id = g.id AND m.place_id = $3)
    LIMIT 1`, destinationID, left, right).Scan(&existing)
	switch {
	case err == nil:
		return existing, false, nil
	case err != pgx.ErrNoRows:
		return uuid.Nil, false, fmt.Errorf("look up duplicate group: %w", err)
	}

	var groupID uuid.UUID
	if err := transaction.QueryRow(ctx, `INSERT INTO travel_place_duplicate_groups
    (destination_id, status, reason) VALUES ($1, 'open', $2) RETURNING id`,
		destinationID, nullText(reason)).Scan(&groupID); err != nil {
		return uuid.Nil, false, fmt.Errorf("create duplicate group: %w", err)
	}

	for _, placeID := range []uuid.UUID{left, right} {
		if _, err := transaction.Exec(ctx, `INSERT INTO travel_place_duplicate_group_members
      (group_id, place_id, confidence, reason) VALUES ($1,$2,$3,$4)
      ON CONFLICT (group_id, place_id) DO NOTHING`,
			groupID, placeID, pair.Confidence, nullText(reason)); err != nil {
			return uuid.Nil, false, fmt.Errorf("add duplicate group member: %w", err)
		}
		if _, err := transaction.Exec(ctx, `UPDATE travel_places SET duplicate_group_id=$2, updated_at=NOW()
      WHERE id=$1 AND review_status NOT IN ('approved','rejected','merged')`, placeID, groupID); err != nil {
			return uuid.Nil, false, fmt.Errorf("link place to duplicate group: %w", err)
		}
	}

	if err := transaction.Commit(ctx); err != nil {
		return uuid.Nil, false, fmt.Errorf("commit duplicate group: %w", err)
	}
	return groupID, true, nil
}

// ListDuplicateGroups returns groups with their members for Ops review.
func (s *Store) ListDuplicateGroups(ctx context.Context, destinationID *uuid.UUID, status string, limit int) ([]DuplicateGroup, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("knowledge store is required")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if status == "" {
		status = DuplicateGroupOpen
	}
	rows, err := s.db.Query(ctx, `SELECT id, destination_id, canonical_place_id, status, COALESCE(reason,'')
    FROM travel_place_duplicate_groups
    WHERE ($1::uuid IS NULL OR destination_id = $1) AND status = $2
    ORDER BY created_at DESC, id
    LIMIT $3`, destinationID, status, limit)
	if err != nil {
		return nil, fmt.Errorf("list duplicate groups: %w", err)
	}
	defer rows.Close()

	groups := make([]DuplicateGroup, 0, limit)
	for rows.Next() {
		var group DuplicateGroup
		if err := rows.Scan(&group.ID, &group.DestinationID, &group.CanonicalPlaceID, &group.Status, &group.Reason); err != nil {
			return nil, fmt.Errorf("scan duplicate group: %w", err)
		}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for index := range groups {
		members, err := s.duplicateGroupMembers(ctx, groups[index].ID)
		if err != nil {
			return nil, err
		}
		groups[index].Members = members
	}
	return groups, nil
}

func (s *Store) duplicateGroupMembers(ctx context.Context, groupID uuid.UUID) ([]DuplicateGroupMember, error) {
	rows, err := s.db.Query(ctx, `SELECT m.place_id, p.canonical_name, p.category, p.quality_score,
      p.review_status, m.confidence, COALESCE(m.reason,'')
    FROM travel_place_duplicate_group_members m
    JOIN travel_places p ON p.id = m.place_id
    WHERE m.group_id = $1
    ORDER BY p.quality_score DESC NULLS LAST, p.canonical_name, m.place_id`, groupID)
	if err != nil {
		return nil, fmt.Errorf("list duplicate group members: %w", err)
	}
	defer rows.Close()

	members := make([]DuplicateGroupMember, 0, 4)
	for rows.Next() {
		var member DuplicateGroupMember
		if err := rows.Scan(&member.PlaceID, &member.CanonicalName, &member.Category, &member.QualityScore,
			&member.ReviewStatus, &member.Confidence, &member.Reason); err != nil {
			return nil, fmt.Errorf("scan duplicate group member: %w", err)
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

// MergeDuplicateGroup makes one member canonical and marks the rest merged.
// Merged records keep their row (for audit and for existing itinerary
// references) but are excluded from grounding retrieval and scored to zero.
func (s *Store) MergeDuplicateGroup(
	ctx context.Context,
	groupID uuid.UUID,
	canonicalPlaceID uuid.UUID,
	actorUserID *uuid.UUID,
	reason string,
) (MergeResolution, error) {
	if s == nil || s.db == nil {
		return MergeResolution{}, fmt.Errorf("knowledge store is required")
	}

	transaction, err := s.db.BeginTx(ctx, pgxTxOptions())
	if err != nil {
		return MergeResolution{}, fmt.Errorf("begin merge transaction: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	var groupStatus string
	if err := transaction.QueryRow(ctx, `SELECT status FROM travel_place_duplicate_groups WHERE id=$1 FOR UPDATE`,
		groupID).Scan(&groupStatus); err != nil {
		if err == pgx.ErrNoRows {
			return MergeResolution{}, ErrDuplicateGroupNotFound
		}
		return MergeResolution{}, fmt.Errorf("load duplicate group %s: %w", groupID, err)
	}
	if groupStatus != DuplicateGroupOpen {
		return MergeResolution{}, fmt.Errorf("%w: group is %s", ErrMergeConflict, groupStatus)
	}

	candidates, err := loadMergeCandidates(ctx, transaction, groupID)
	if err != nil {
		return MergeResolution{}, err
	}
	if len(candidates) < 2 {
		return MergeResolution{}, fmt.Errorf("%w: group needs at least two members", ErrMergeConflict)
	}
	canonicalFound := false
	for _, candidate := range candidates {
		if candidate.PlaceID == canonicalPlaceID.String() {
			canonicalFound = true
			break
		}
	}
	if !canonicalFound {
		return MergeResolution{}, fmt.Errorf("%w: canonical place is not a member of the group", ErrMergeConflict)
	}

	resolution := ResolveMerge(canonicalPlaceID.String(), candidates)

	aliases, err := json.Marshal(resolution.Aliases)
	if err != nil {
		return MergeResolution{}, fmt.Errorf("marshal merged aliases: %w", err)
	}
	tags, err := json.Marshal(resolution.Tags)
	if err != nil {
		return MergeResolution{}, fmt.Errorf("marshal merged tags: %w", err)
	}
	providerRefs, err := json.Marshal(resolution.ProviderRefs)
	if err != nil {
		return MergeResolution{}, fmt.Errorf("marshal merged provider refs: %w", err)
	}

	if _, err := transaction.Exec(ctx, `UPDATE travel_places SET
      category=$2, latitude=COALESCE($3, latitude), longitude=COALESCE($4, longitude),
      address=COALESCE($5, address), website=COALESCE($6, website),
      opening_hours=COALESCE($7, opening_hours),
      aliases=$8, tags=$9, provider_refs=$10,
      duplicate_group_id=NULL, canonical_place_id=NULL, merged_into_place_id=NULL,
      updated_at=NOW()
    WHERE id=$1`, canonicalPlaceID, resolution.Category, resolution.Latitude, resolution.Longitude,
		nullText(resolution.Address), nullText(resolution.Website), optionalBytes(resolution.OpeningHours),
		aliases, tags, providerRefs); err != nil {
		return MergeResolution{}, fmt.Errorf("update canonical place: %w", err)
	}

	for _, mergedID := range resolution.MergedPlaceIDs {
		placeID, parseErr := uuid.Parse(mergedID)
		if parseErr != nil {
			return MergeResolution{}, fmt.Errorf("parse merged place id: %w", parseErr)
		}
		// Merged records are excluded from grounding by review_status and by a
		// zeroed quality score, so neither retrieval path can surface them.
		if _, err := transaction.Exec(ctx, `UPDATE travel_places SET
        review_status='merged', merged_into_place_id=$2, canonical_place_id=$2,
        quality_score=0, duplicate_group_id=NULL, status='archived', updated_at=NOW()
      WHERE id=$1`, placeID, canonicalPlaceID); err != nil {
			return MergeResolution{}, fmt.Errorf("mark place %s merged: %w", placeID, err)
		}
		// Observations follow their place so provider evidence stays attached
		// to the surviving record.
		if _, err := transaction.Exec(ctx, `UPDATE travel_provider_place_observations
        SET matched_place_id=$2, match_status='duplicate', updated_at=NOW()
      WHERE matched_place_id=$1`, placeID, canonicalPlaceID); err != nil {
			return MergeResolution{}, fmt.Errorf("repoint observations for %s: %w", placeID, err)
		}
	}

	if _, err := transaction.Exec(ctx, `UPDATE travel_place_duplicate_groups SET
      status='merged', canonical_place_id=$2, resolved_by_user_id=$3, resolved_at=NOW(), updated_at=NOW()
    WHERE id=$1`, groupID, canonicalPlaceID, actorUserID); err != nil {
		return MergeResolution{}, fmt.Errorf("close duplicate group: %w", err)
	}

	if err := insertReviewEvent(ctx, transaction, &canonicalPlaceID, &groupID, actorUserID, "merged",
		map[string]any{"mergedPlaceIds": resolution.MergedPlaceIDs},
		map[string]any{"canonicalPlaceId": canonicalPlaceID.String(), "fieldSources": resolution.FieldSources},
		reason); err != nil {
		return MergeResolution{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return MergeResolution{}, fmt.Errorf("commit merge: %w", err)
	}
	return resolution, nil
}

// RejectDuplicateGroup records that the members are genuinely distinct places.
func (s *Store) RejectDuplicateGroup(ctx context.Context, groupID uuid.UUID, actorUserID *uuid.UUID, reason string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("knowledge store is required")
	}
	transaction, err := s.db.BeginTx(ctx, pgxTxOptions())
	if err != nil {
		return fmt.Errorf("begin reject transaction: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	tag, err := transaction.Exec(ctx, `UPDATE travel_place_duplicate_groups SET
      status='rejected', resolved_by_user_id=$2, resolved_at=NOW(), reason=COALESCE($3, reason), updated_at=NOW()
    WHERE id=$1 AND status='open'`, groupID, actorUserID, nullText(reason))
	if err != nil {
		return fmt.Errorf("reject duplicate group: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDuplicateGroupNotFound
	}
	if _, err := transaction.Exec(ctx, `UPDATE travel_places SET duplicate_group_id=NULL, updated_at=NOW()
    WHERE duplicate_group_id=$1`, groupID); err != nil {
		return fmt.Errorf("unlink places from duplicate group: %w", err)
	}
	if err := insertReviewEvent(ctx, transaction, nil, &groupID, actorUserID, "duplicate_rejected",
		map[string]any{"status": DuplicateGroupOpen},
		map[string]any{"status": DuplicateGroupRejected}, reason); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit duplicate rejection: %w", err)
	}
	return nil
}

func loadMergeCandidates(ctx context.Context, transaction pgx.Tx, groupID uuid.UUID) ([]MergeCandidate, error) {
	rows, err := transaction.Query(ctx, `SELECT p.id, COALESCE(s.trust_level, 'unknown'), COALESCE(p.quality_score, 0),
      COALESCE(EXTRACT(EPOCH FROM p.last_provider_refresh_at), EXTRACT(EPOCH FROM p.updated_at), 0),
      p.category, p.latitude, p.longitude, COALESCE(p.address,''), COALESCE(p.website,''),
      p.opening_hours, p.aliases, p.tags, p.provider_refs
    FROM travel_place_duplicate_group_members m
    JOIN travel_places p ON p.id = m.place_id
    LEFT JOIN travel_knowledge_sources s ON s.id = p.source_id
    WHERE m.group_id = $1
    ORDER BY p.id`, groupID)
	if err != nil {
		return nil, fmt.Errorf("load merge candidates: %w", err)
	}
	defer rows.Close()

	candidates := make([]MergeCandidate, 0, 4)
	for rows.Next() {
		var (
			candidate    MergeCandidate
			placeID      uuid.UUID
			observedAt   float64
			openingHours []byte
			aliasesJSON  []byte
			tagsJSON     []byte
			refsJSON     []byte
		)
		if err := rows.Scan(&placeID, &candidate.TrustLevel, &candidate.QualityScore, &observedAt,
			&candidate.Category, &candidate.Latitude, &candidate.Longitude, &candidate.Address,
			&candidate.Website, &openingHours, &aliasesJSON, &tagsJSON, &refsJSON); err != nil {
			return nil, fmt.Errorf("scan merge candidate: %w", err)
		}
		candidate.PlaceID = placeID.String()
		candidate.ObservedAt = int64(observedAt)
		candidate.OpeningHours = openingHours
		candidate.Aliases = decodeStringArray(aliasesJSON)
		candidate.Tags = decodeStringArray(tagsJSON)
		candidate.ProviderRefs = decodeProviderRefs(refsJSON)
		candidates = append(candidates, candidate)
	}
	return candidates, rows.Err()
}

// insertReviewEvent writes the audit trail. Values are summaries only: no
// secrets, no raw provider payloads, no private user content.
func insertReviewEvent(
	ctx context.Context,
	transaction pgx.Tx,
	placeID *uuid.UUID,
	groupID *uuid.UUID,
	actorUserID *uuid.UUID,
	action string,
	oldValues, newValues map[string]any,
	reason string,
) error {
	oldEncoded, err := json.Marshal(oldValues)
	if err != nil {
		return fmt.Errorf("marshal audit old values: %w", err)
	}
	newEncoded, err := json.Marshal(newValues)
	if err != nil {
		return fmt.Errorf("marshal audit new values: %w", err)
	}
	if _, err := transaction.Exec(ctx, `INSERT INTO travel_knowledge_review_events
    (place_id, duplicate_group_id, actor_user_id, action, old_values, new_values, reason)
    VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		placeID, groupID, actorUserID, action, oldEncoded, newEncoded, nullText(reason)); err != nil {
		return fmt.Errorf("write review audit event: %w", err)
	}
	return nil
}

func pgxTxOptions() pgx.TxOptions { return pgx.TxOptions{} }

func optionalBytes(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return value
}
