package postgres

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
)

func (r *Repository) CreateWorkspaceBudget(ctx context.Context, b *entity.WorkspaceBudget) (*entity.WorkspaceBudget, error) {
	if b.IsPrimary {
		tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return nil, fmt.Errorf("begin create workspace budget tx: %w", err)
		}
		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback(ctx)
			}
		}()
		if err := r.clearPrimaryWorkspaceBudgets(ctx, tx, b.WorkspaceID); err != nil {
			return nil, err
		}
		created, err := r.createWorkspaceBudget(ctx, tx, b)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit create workspace budget tx: %w", err)
		}
		committed = true
		return created, nil
	}
	return r.createWorkspaceBudget(ctx, r.db, b)
}

func (r *Repository) GetWorkspaceBudgetByID(ctx context.Context, workspaceID, budgetID uuid.UUID) (*entity.WorkspaceBudget, error) {
	query, args, err := r.db.Builder.
		Select(dto.WorkspaceBudgetColumns).
		From("workspace_budgets").
		Where(sq.Eq{"workspace_id": dto.IDArg(workspaceID), "id": dto.IDArg(budgetID)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get workspace budget: %w", err)
	}
	return dto.ScanWorkspaceBudget(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListWorkspaceBudgetsByWorkspace(
	ctx context.Context,
	workspaceID uuid.UUID,
	status *entity.WorkspaceBudgetStatus,
) ([]entity.WorkspaceBudget, error) {
	builder := r.db.Builder.
		Select(dto.WorkspaceBudgetColumns).
		From("workspace_budgets").
		Where(sq.Eq{"workspace_id": dto.IDArg(workspaceID)})
	if status != nil {
		builder = builder.Where(sq.Eq{"status": string(*status)})
	}
	query, args, err := builder.
		OrderBy("is_primary DESC", "created_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list workspace budgets: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query workspace budgets: %w", err)
	}
	defer rows.Close()
	return dto.ScanWorkspaceBudgetRows(rows)
}

func (r *Repository) ListActiveWorkspaceBudgetsByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]entity.WorkspaceBudget, error) {
	status := entity.WorkspaceBudgetStatusActive
	return r.ListWorkspaceBudgetsByWorkspace(ctx, workspaceID, &status)
}

func (r *Repository) GetPrimaryWorkspaceBudget(ctx context.Context, workspaceID uuid.UUID) (*entity.WorkspaceBudget, error) {
	query, args, err := r.db.Builder.
		Select(dto.WorkspaceBudgetColumns).
		From("workspace_budgets").
		Where(sq.Eq{
			"workspace_id": dto.IDArg(workspaceID),
			"status":       string(entity.WorkspaceBudgetStatusActive),
			"is_primary":   true,
		}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get primary workspace budget: %w", err)
	}
	return dto.ScanWorkspaceBudget(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) UpdateWorkspaceBudget(ctx context.Context, b *entity.WorkspaceBudget) (*entity.WorkspaceBudget, error) {
	if b.IsPrimary && b.Status == entity.WorkspaceBudgetStatusActive {
		tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return nil, fmt.Errorf("begin update workspace budget tx: %w", err)
		}
		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback(ctx)
			}
		}()
		if err := r.clearPrimaryWorkspaceBudgets(ctx, tx, b.WorkspaceID); err != nil {
			return nil, err
		}
		updated, err := r.updateWorkspaceBudget(ctx, tx, b)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit update workspace budget tx: %w", err)
		}
		committed = true
		return updated, nil
	}
	return r.updateWorkspaceBudget(ctx, r.db, b)
}

func (r *Repository) ArchiveWorkspaceBudget(ctx context.Context, workspaceID, budgetID, actorUserID uuid.UUID) (*entity.WorkspaceBudget, error) {
	query, args, err := r.db.Builder.
		Update("workspace_budgets").
		Set("status", string(entity.WorkspaceBudgetStatusArchived)).
		Set("is_primary", false).
		Set("archived_by_user_id", dto.IDArg(actorUserID)).
		Set("archived_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"workspace_id": dto.IDArg(workspaceID),
			"id":           dto.IDArg(budgetID),
		}).
		Where(sq.NotEq{"status": string(entity.WorkspaceBudgetStatusArchived)}).
		Suffix("RETURNING " + dto.WorkspaceBudgetColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build archive workspace budget: %w", err)
	}
	return dto.ScanWorkspaceBudget(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) SetWorkspaceBudgetPrimary(ctx context.Context, workspaceID, budgetID uuid.UUID) (*entity.WorkspaceBudget, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin set workspace budget primary tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	if err := r.clearPrimaryWorkspaceBudgets(ctx, tx, workspaceID); err != nil {
		return nil, err
	}
	query, args, err := r.db.Builder.
		Update("workspace_budgets").
		Set("is_primary", true).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"workspace_id": dto.IDArg(workspaceID),
			"id":           dto.IDArg(budgetID),
			"status":       string(entity.WorkspaceBudgetStatusActive),
		}).
		Suffix("RETURNING " + dto.WorkspaceBudgetColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build set workspace budget primary: %w", err)
	}
	updated, err := dto.ScanWorkspaceBudget(tx.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit set workspace budget primary tx: %w", err)
	}
	committed = true
	return updated, nil
}

func (r *Repository) ClearPrimaryWorkspaceBudgets(ctx context.Context, workspaceID uuid.UUID) error {
	return r.clearPrimaryWorkspaceBudgets(ctx, r.db, workspaceID)
}

func (r *Repository) CountWorkspaceBudgets(ctx context.Context, workspaceID uuid.UUID, status *entity.WorkspaceBudgetStatus) (int, error) {
	builder := r.db.Builder.
		Select("COUNT(*)").
		From("workspace_budgets").
		Where(sq.Eq{"workspace_id": dto.IDArg(workspaceID)})
	if status != nil {
		builder = builder.Where(sq.Eq{"status": string(*status)})
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count workspace budgets: %w", err)
	}
	var count int
	if err := r.db.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count workspace budgets: %w", err)
	}
	return count, nil
}

func (r *Repository) createWorkspaceBudget(ctx context.Context, q rowQuerier, b *entity.WorkspaceBudget) (*entity.WorkspaceBudget, error) {
	query, args, err := r.db.Builder.
		Insert("workspace_budgets").
		Columns(dto.WorkspaceBudgetInsertColumns()...).
		Values(dto.WorkspaceBudgetInsertValues(b)...).
		Suffix("RETURNING " + dto.WorkspaceBudgetColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create workspace budget: %w", err)
	}
	return dto.ScanWorkspaceBudget(q.QueryRow(ctx, query, args...))
}

func (r *Repository) updateWorkspaceBudget(ctx context.Context, q rowQuerier, b *entity.WorkspaceBudget) (*entity.WorkspaceBudget, error) {
	query, args, err := r.db.Builder.
		Update("workspace_budgets").
		Set("name", b.Name).
		Set("description", dto.TextArg(valueOrEmpty(b.Description))).
		Set("amount", dto.NumericArg(&b.Amount)).
		Set("currency", b.Currency).
		Set("period_start", dto.DateArg(b.PeriodStart)).
		Set("period_end", dto.DateArg(b.PeriodEnd)).
		Set("is_primary", b.IsPrimary).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"workspace_id": dto.IDArg(b.WorkspaceID),
			"id":           dto.IDArg(b.ID),
			"status":       string(entity.WorkspaceBudgetStatusActive),
		}).
		Suffix("RETURNING " + dto.WorkspaceBudgetColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update workspace budget: %w", err)
	}
	return dto.ScanWorkspaceBudget(q.QueryRow(ctx, query, args...))
}

func (r *Repository) clearPrimaryWorkspaceBudgets(ctx context.Context, q interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, workspaceID uuid.UUID) error {
	query, args, err := r.db.Builder.
		Update("workspace_budgets").
		Set("is_primary", false).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"workspace_id": dto.IDArg(workspaceID),
			"status":       string(entity.WorkspaceBudgetStatusActive),
			"is_primary":   true,
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build clear primary workspace budgets: %w", err)
	}
	if _, err := q.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("clear primary workspace budgets: %w", err)
	}
	return nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
