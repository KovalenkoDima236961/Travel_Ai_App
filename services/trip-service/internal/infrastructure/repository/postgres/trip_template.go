package postgres

import (
	"context"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
)

func (r *Repository) CreateTripTemplate(ctx context.Context, t *entity.TripTemplate) (*entity.TripTemplate, error) {
	query, args, err := r.db.Builder.
		Insert("trip_templates").
		Columns(dto.TripTemplateInsertColumns()...).
		Values(dto.TripTemplateInsertValues(t)...).
		Suffix("RETURNING " + dto.TripTemplateColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build insert trip template: %w", err)
	}

	return dto.ScanTripTemplate(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetTripTemplateByID(ctx context.Context, id uuid.UUID) (*entity.TripTemplate, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripTemplateColumns).
		From("trip_templates").
		Where(sq.Eq{"id": dto.IDArg(id)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get trip template: %w", err)
	}

	return dto.ScanTripTemplate(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListTripTemplates(
	ctx context.Context,
	userID uuid.UUID,
	workspaceIDs []uuid.UUID,
	in appdto.ListTripTemplatesInput,
) ([]entity.TripTemplate, error) {
	builder := r.db.Builder.
		Select(dto.TripTemplateSummaryColumns).
		From("trip_templates")

	status := string(in.Status)
	if status == "" {
		status = string(entity.TripTemplateStatusActive)
	}
	builder = builder.Where(sq.Eq{"status": status})

	switch in.Visibility {
	case entity.TripTemplateVisibilityPrivate:
		builder = builder.Where(sq.Eq{
			"visibility":         string(entity.TripTemplateVisibilityPrivate),
			"created_by_user_id": dto.IDArg(userID),
		})
	case entity.TripTemplateVisibilityWorkspace:
		ids := filterWorkspaceIDs(workspaceIDs, in.WorkspaceID)
		if len(ids) == 0 {
			return []entity.TripTemplate{}, nil
		}
		builder = builder.Where(sq.Eq{
			"visibility":   string(entity.TripTemplateVisibilityWorkspace),
			"workspace_id": ids,
		})
	default:
		ids := filterWorkspaceIDs(workspaceIDs, in.WorkspaceID)
		if in.WorkspaceID != nil && len(ids) == 0 {
			return []entity.TripTemplate{}, nil
		}
		access := sq.Or{
			sq.And{
				sq.Eq{"visibility": string(entity.TripTemplateVisibilityPrivate)},
				sq.Eq{"created_by_user_id": dto.IDArg(userID)},
			},
		}
		if len(ids) > 0 {
			access = append(access, sq.And{
				sq.Eq{"visibility": string(entity.TripTemplateVisibilityWorkspace)},
				sq.Eq{"workspace_id": ids},
			})
		}
		builder = builder.Where(access)
	}

	if tag := strings.TrimSpace(in.Tag); tag != "" {
		builder = builder.Where(sq.Expr("? = ANY(tags)", tag))
	}
	if q := strings.TrimSpace(in.Query); q != "" {
		like := "%" + q + "%"
		builder = builder.Where(sq.Or{
			sq.ILike{"title": like},
			sq.ILike{"destination_hint": like},
		})
	}

	query, args, err := builder.
		OrderBy("created_at DESC", "id DESC").
		Limit(uint64(in.Limit)).
		Offset(uint64(in.Offset)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trip templates: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query trip templates: %w", err)
	}
	defer rows.Close()

	return dto.ScanTripTemplateRows(rows)
}

func (r *Repository) UpdateTripTemplateMetadata(ctx context.Context, t *entity.TripTemplate) (*entity.TripTemplate, error) {
	query, args, err := r.db.Builder.
		Update("trip_templates").
		Set("title", t.Title).
		Set("description", dto.TextArg(valueOrEmpty(t.Description))).
		Set("destination_hint", dto.TextArg(valueOrEmpty(t.DestinationHint))).
		Set("default_currency", dto.TextArg(valueOrEmpty(t.DefaultCurrency))).
		Set("tags", t.Tags).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": dto.IDArg(t.ID)}).
		Suffix("RETURNING " + dto.TripTemplateColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update trip template metadata: %w", err)
	}

	return dto.ScanTripTemplate(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ArchiveTripTemplate(ctx context.Context, id, actorUserID uuid.UUID) (*entity.TripTemplate, error) {
	query, args, err := r.db.Builder.
		Update("trip_templates").
		Set("status", string(entity.TripTemplateStatusArchived)).
		Set("archived_at", sq.Expr("NOW()")).
		Set("archived_by_user_id", dto.IDArg(actorUserID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": dto.IDArg(id)}).
		Where(sq.NotEq{"status": string(entity.TripTemplateStatusArchived)}).
		Suffix("RETURNING " + dto.TripTemplateColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build archive trip template: %w", err)
	}

	return dto.ScanTripTemplate(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) CountTripTemplates(
	ctx context.Context,
	userID uuid.UUID,
	workspaceIDs []uuid.UUID,
	in appdto.ListTripTemplatesInput,
) (int, error) {
	builder := r.db.Builder.
		Select("COUNT(*)").
		From("trip_templates")

	status := string(in.Status)
	if status == "" {
		status = string(entity.TripTemplateStatusActive)
	}
	builder = builder.Where(sq.Eq{"status": status})

	switch in.Visibility {
	case entity.TripTemplateVisibilityPrivate:
		builder = builder.Where(sq.Eq{
			"visibility":         string(entity.TripTemplateVisibilityPrivate),
			"created_by_user_id": dto.IDArg(userID),
		})
	case entity.TripTemplateVisibilityWorkspace:
		ids := filterWorkspaceIDs(workspaceIDs, in.WorkspaceID)
		if len(ids) == 0 {
			return 0, nil
		}
		builder = builder.Where(sq.Eq{
			"visibility":   string(entity.TripTemplateVisibilityWorkspace),
			"workspace_id": ids,
		})
	default:
		ids := filterWorkspaceIDs(workspaceIDs, in.WorkspaceID)
		if in.WorkspaceID != nil && len(ids) == 0 {
			return 0, nil
		}
		access := sq.Or{
			sq.And{
				sq.Eq{"visibility": string(entity.TripTemplateVisibilityPrivate)},
				sq.Eq{"created_by_user_id": dto.IDArg(userID)},
			},
		}
		if len(ids) > 0 {
			access = append(access, sq.And{
				sq.Eq{"visibility": string(entity.TripTemplateVisibilityWorkspace)},
				sq.Eq{"workspace_id": ids},
			})
		}
		builder = builder.Where(access)
	}

	if tag := strings.TrimSpace(in.Tag); tag != "" {
		builder = builder.Where(sq.Expr("? = ANY(tags)", tag))
	}
	if q := strings.TrimSpace(in.Query); q != "" {
		like := "%" + q + "%"
		builder = builder.Where(sq.Or{
			sq.ILike{"title": like},
			sq.ILike{"destination_hint": like},
		})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count trip templates: %w", err)
	}
	var count int
	if err := r.db.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count trip templates: %w", err)
	}
	return count, nil
}
