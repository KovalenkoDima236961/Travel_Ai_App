package workspacepolicies

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	storage "github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

const policyColumns = `id, workspace_id, name, description, rules_json, status,
	created_by_user_id, updated_by_user_id, created_at, updated_at,
	archived_at, archived_by_user_id`

type Repository interface {
	UpsertActive(context.Context, uuid.UUID, uuid.UUID, UpsertInput) (*Policy, error)
	GetActive(context.Context, uuid.UUID) (*Policy, error)
	GetByID(context.Context, uuid.UUID, uuid.UUID) (*Policy, error)
	ArchiveActive(context.Context, uuid.UUID, uuid.UUID) (*Policy, error)
}

type PostgresRepository struct {
	db *storage.DB
}

func NewRepository(db *storage.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) UpsertActive(
	ctx context.Context,
	workspaceID, actorUserID uuid.UUID,
	input UpsertInput,
) (*Policy, error) {
	raw, err := json.Marshal(input.Rules)
	if err != nil {
		return nil, fmt.Errorf("marshal workspace policy rules: %w", err)
	}
	const query = `INSERT INTO workspace_policies (
		id, workspace_id, name, description, rules_json, status,
		created_by_user_id, updated_by_user_id
	) VALUES ($1, $2, $3, $4, $5, 'active', $6, $6)
	ON CONFLICT (workspace_id) WHERE status = 'active'
	DO UPDATE SET name = EXCLUDED.name,
		description = EXCLUDED.description,
		rules_json = EXCLUDED.rules_json,
		updated_by_user_id = EXCLUDED.updated_by_user_id,
		updated_at = NOW()
	RETURNING ` + policyColumns
	return scanPolicy(r.db.QueryRow(
		ctx, query, uuid.New(), workspaceID, input.Name, input.Description, raw, actorUserID,
	))
}

func (r *PostgresRepository) GetActive(
	ctx context.Context,
	workspaceID uuid.UUID,
) (*Policy, error) {
	const query = `SELECT ` + policyColumns + `
		FROM workspace_policies WHERE workspace_id = $1 AND status = 'active'`
	return scanPolicy(r.db.QueryRow(ctx, query, workspaceID))
}

func (r *PostgresRepository) GetByID(
	ctx context.Context,
	workspaceID, policyID uuid.UUID,
) (*Policy, error) {
	const query = `SELECT ` + policyColumns + `
		FROM workspace_policies WHERE workspace_id = $1 AND id = $2`
	return scanPolicy(r.db.QueryRow(ctx, query, workspaceID, policyID))
}

func (r *PostgresRepository) ArchiveActive(
	ctx context.Context,
	workspaceID, actorUserID uuid.UUID,
) (*Policy, error) {
	const query = `UPDATE workspace_policies
		SET status = 'archived', archived_at = NOW(), archived_by_user_id = $2,
			updated_at = NOW(), updated_by_user_id = $2
		WHERE workspace_id = $1 AND status = 'active'
		RETURNING ` + policyColumns
	return scanPolicy(r.db.QueryRow(ctx, query, workspaceID, actorUserID))
}

func scanPolicy(row pgx.Row) (*Policy, error) {
	var policy Policy
	var raw []byte
	if err := row.Scan(
		&policy.ID,
		&policy.WorkspaceID,
		&policy.Name,
		&policy.Description,
		&raw,
		&policy.Status,
		&policy.CreatedByUserID,
		&policy.UpdatedByUserID,
		&policy.CreatedAt,
		&policy.UpdatedAt,
		&policy.ArchivedAt,
		&policy.ArchivedByUserID,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan workspace policy: %w", err)
	}
	if err := json.Unmarshal(raw, &policy.Rules); err != nil {
		return nil, fmt.Errorf("decode workspace policy rules: %w", err)
	}
	return &policy, nil
}
