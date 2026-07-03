package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/storage/postgres"
)

const WorkspaceBudgetColumns = "id, workspace_id, name, description, amount, currency, " +
	"period_start, period_end, status, is_primary, created_by_user_id, archived_by_user_id, " +
	"created_at, updated_at, archived_at"

func WorkspaceBudgetInsertColumns() []string {
	return []string{
		"id", "workspace_id", "name", "description", "amount", "currency",
		"period_start", "period_end", "status", "is_primary", "created_by_user_id",
	}
}

func WorkspaceBudgetInsertValues(b *entity.WorkspaceBudget) []any {
	return []any{
		IDArg(b.ID),
		IDArg(b.WorkspaceID),
		b.Name,
		toPgTextPtr(b.Description),
		NumericArg(&b.Amount),
		b.Currency,
		toPgDate(b.PeriodStart),
		toPgDate(b.PeriodEnd),
		string(b.Status),
		b.IsPrimary,
		IDArg(b.CreatedByUserID),
	}
}

func DateArg(t *time.Time) pgtype.Date {
	return toPgDate(t)
}

func ScanWorkspaceBudget(row pgx.Row) (*entity.WorkspaceBudget, error) {
	var (
		id, workspaceID, createdByUserID, archivedByUserID pgtype.UUID
		name, currency, status                             string
		description                                        pgtype.Text
		amount                                             pgtype.Numeric
		periodStart, periodEnd                             pgtype.Date
		isPrimary                                          bool
		createdAt, updatedAt, archivedAt                   pgtype.Timestamp
	)

	err := row.Scan(
		&id,
		&workspaceID,
		&name,
		&description,
		&amount,
		&currency,
		&periodStart,
		&periodEnd,
		&status,
		&isPrimary,
		&createdByUserID,
		&archivedByUserID,
		&createdAt,
		&updatedAt,
		&archivedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan workspace budget: %w", err)
	}

	amountValue := fromPgNumeric(amount)
	if amountValue == nil {
		return nil, fmt.Errorf("scan workspace budget: amount is null")
	}

	return &entity.WorkspaceBudget{
		ID:               uuid.UUID(id.Bytes),
		WorkspaceID:      uuid.UUID(workspaceID.Bytes),
		Name:             name,
		Description:      fromPgText(description),
		Amount:           *amountValue,
		Currency:         currency,
		PeriodStart:      fromPgDate(periodStart),
		PeriodEnd:        fromPgDate(periodEnd),
		Status:           entity.WorkspaceBudgetStatus(status),
		IsPrimary:        isPrimary,
		CreatedByUserID:  uuid.UUID(createdByUserID.Bytes),
		ArchivedByUserID: fromPgUUID(archivedByUserID),
		CreatedAt:        createdAt.Time,
		UpdatedAt:        updatedAt.Time,
		ArchivedAt:       fromPgTimestampPtr(archivedAt),
	}, nil
}

func ScanWorkspaceBudgetRows(rows pgx.Rows) ([]entity.WorkspaceBudget, error) {
	out := make([]entity.WorkspaceBudget, 0)
	for rows.Next() {
		budget, err := ScanWorkspaceBudget(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *budget)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace budgets: %w", err)
	}
	return out, nil
}
