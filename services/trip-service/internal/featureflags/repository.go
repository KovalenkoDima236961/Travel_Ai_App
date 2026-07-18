package featureflags

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

// PostgresRepository keeps runtime overrides in the service database. The
// global scope is the only active scope in v1; the schema reserves workspace
// and user scopes without silently evaluating them.
type PostgresRepository struct {
	db *postgres.DB
}

func NewPostgresRepository(db *postgres.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetGlobalOverride(ctx context.Context, key, environment string) (*Override, error) {
	row := r.db.QueryRow(ctx, `
SELECT id, key, value_type, bool_value, environment, scope_type, scope_id,
       description, enabled, source, created_by_user_id, updated_by_user_id,
       created_at, updated_at
FROM feature_flags
WHERE key = $1
  AND scope_type = 'global'
  AND scope_id IS NULL
  AND (environment = $2 OR environment IS NULL)
ORDER BY CASE WHEN environment = $2 THEN 0 ELSE 1 END, updated_at DESC
LIMIT 1`, key, environment)
	override, err := scanOverride(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query feature flag override: %w", err)
	}
	return override, nil
}

func (r *PostgresRepository) SaveGlobalOverride(ctx context.Context, override Override, audit AuditEvent) (*Override, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin feature flag update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row := tx.QueryRow(ctx, `
INSERT INTO feature_flags (
    id, key, value_type, bool_value, environment, scope_type, scope_id,
    enabled, source, created_by_user_id, updated_by_user_id, created_at, updated_at
) VALUES ($1, $2, $3, $4, NULLIF($5, ''), 'global', NULL, $6, 'db', $7, $8, $9, $10)
ON CONFLICT DO UPDATE SET
    value_type = EXCLUDED.value_type,
    bool_value = EXCLUDED.bool_value,
    enabled = EXCLUDED.enabled,
    source = EXCLUDED.source,
    updated_by_user_id = EXCLUDED.updated_by_user_id,
    updated_at = EXCLUDED.updated_at
RETURNING id, key, value_type, bool_value, environment, scope_type, scope_id,
          description, enabled, source, created_by_user_id, updated_by_user_id,
          created_at, updated_at`,
		override.ID, override.Key, override.ValueType, override.BoolValue, overrideEnvironment(override.Environment),
		override.Enabled, override.CreatedBy, override.UpdatedBy, override.CreatedAt, override.UpdatedAt)
	saved, err := scanOverride(row)
	if err != nil {
		return nil, fmt.Errorf("save feature flag override: %w", err)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit feature flag update: %w", err)
	}
	return saved, nil
}

func (r *PostgresRepository) DeleteGlobalOverride(ctx context.Context, key, environment string, audit AuditEvent) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin feature flag reset: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM feature_flags WHERE key = $1 AND scope_type = 'global' AND scope_id IS NULL AND environment IS NOT DISTINCT FROM NULLIF($2, '')`, key, environment); err != nil {
		return fmt.Errorf("delete feature flag override: %w", err)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit feature flag reset: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListAudit(ctx context.Context, key string, limit int) ([]AuditEvent, error) {
	rows, err := r.db.Query(ctx, `
SELECT id, flag_key, environment, scope_type, scope_id, actor_user_id, action,
       old_value, new_value, reason, request_id, created_at
FROM feature_flag_audit_events
WHERE flag_key = $1
ORDER BY created_at DESC
LIMIT $2`, key, limit)
	if err != nil {
		return nil, fmt.Errorf("query feature flag audit: %w", err)
	}
	defer rows.Close()
	items := make([]AuditEvent, 0)
	for rows.Next() {
		var event AuditEvent
		var oldValue, newValue []byte
		if err := rows.Scan(&event.ID, &event.FlagKey, &event.Environment, &event.ScopeType, &event.ScopeID, &event.ActorUserID, &event.Action, &oldValue, &newValue, &event.Reason, &event.RequestID, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan feature flag audit: %w", err)
		}
		if len(oldValue) > 0 && json.Unmarshal(oldValue, &event.OldValue) != nil {
			return nil, fmt.Errorf("decode feature flag audit old value")
		}
		if len(newValue) > 0 && json.Unmarshal(newValue, &event.NewValue) != nil {
			return nil, fmt.Errorf("decode feature flag audit new value")
		}
		items = append(items, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feature flag audit: %w", err)
	}
	return items, nil
}

type scanner interface{ Scan(...any) error }

func scanOverride(row scanner) (*Override, error) {
	var override Override
	var valueType string
	if err := row.Scan(&override.ID, &override.Key, &valueType, &override.BoolValue, &override.Environment, &override.ScopeType, &override.ScopeID, &override.Description, &override.Enabled, &override.Source, &override.CreatedBy, &override.UpdatedBy, &override.CreatedAt, &override.UpdatedAt); err != nil {
		return nil, err
	}
	override.ValueType = ValueType(valueType)
	return &override, nil
}

func insertAudit(ctx context.Context, tx pgx.Tx, audit AuditEvent) error {
	oldValue, err := json.Marshal(audit.OldValue)
	if err != nil {
		return fmt.Errorf("encode feature flag old audit value: %w", err)
	}
	newValue, err := json.Marshal(audit.NewValue)
	if err != nil {
		return fmt.Errorf("encode feature flag new audit value: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO feature_flag_audit_events (
    id, flag_key, environment, scope_type, scope_id, actor_user_id, action,
    old_value, new_value, reason, request_id, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8::jsonb, 'null'::jsonb), NULLIF($9::jsonb, 'null'::jsonb), NULLIF($10, ''), NULLIF($11, ''), $12)`,
		audit.ID, audit.FlagKey, audit.Environment, audit.ScopeType, audit.ScopeID, audit.ActorUserID,
		audit.Action, string(oldValue), string(newValue), audit.Reason, audit.RequestID, audit.CreatedAt); err != nil {
		return fmt.Errorf("create feature flag audit event: %w", err)
	}
	return nil
}

func overrideEnvironment(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
