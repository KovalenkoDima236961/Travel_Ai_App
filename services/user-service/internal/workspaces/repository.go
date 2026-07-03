package workspaces

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/domain/errs"
	storage "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/pkg/storage/postgres"
)

const (
	workspaceColumns       = "id, name, slug, description, created_by_user_id, created_at, updated_at, archived_at"
	memberColumns          = "id, workspace_id, user_id, role, status, invited_by_user_id, invited_at, joined_at, removed_at, created_at, updated_at"
	inviteSelectColumns    = "i.id, i.workspace_id, w.name, i.email, i.invited_user_id, i.role, i.status, i.invited_by_user_id, i.expires_at, i.accepted_at, i.declined_at, i.revoked_at, i.created_at, i.updated_at"
	inviteReturningColumns = "id, workspace_id, (SELECT name FROM workspaces WHERE id = workspace_invitations.workspace_id), email, invited_user_id, role, status, invited_by_user_id, expires_at, accepted_at, declined_at, revoked_at, created_at, updated_at"
)

type Repository struct {
	db *storage.DB
}

type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func NewRepository(db *storage.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateWorkspaceWithOwner(ctx context.Context, workspace Workspace, owner WorkspaceMember) (*WorkspaceSummary, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin workspace tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	query, args, err := r.db.Builder.
		Insert("workspaces").
		Columns("id", "name", "slug", "description", "created_by_user_id").
		Values(uuidArg(workspace.ID), workspace.Name, workspace.Slug, textPtrArg(workspace.Description), uuidArg(workspace.CreatedByUserID)).
		Suffix("RETURNING " + workspaceColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build workspace insert: %w", err)
	}
	created, err := scanWorkspace(tx.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, mapWriteError(err)
	}

	query, args, err = r.db.Builder.
		Insert("workspace_members").
		Columns("id", "workspace_id", "user_id", "role", "status", "joined_at").
		Values(uuidArg(owner.ID), uuidArg(owner.WorkspaceID), uuidArg(owner.UserID), string(owner.Role), string(owner.Status), sq.Expr("NOW()")).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build owner member insert: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return nil, mapWriteError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit workspace tx: %w", err)
	}
	committed = true

	return &WorkspaceSummary{Workspace: *created, CurrentUserRole: RoleOwner, MemberCount: 1}, nil
}

func (r *Repository) SlugExists(ctx context.Context, slug string) (bool, error) {
	query, args, err := r.db.Builder.
		Select("1").
		From("workspaces").
		Where(sq.Eq{"slug": slug}).
		Limit(1).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build slug check: %w", err)
	}
	var one int
	err = r.db.QueryRow(ctx, query, args...).Scan(&one)
	if storage.NoRowsFound(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("scan slug check: %w", err)
	}
	return true, nil
}

func (r *Repository) ListForUser(ctx context.Context, userID uuid.UUID) ([]WorkspaceSummary, error) {
	query, args, err := r.db.Builder.
		Select(
			"w.id", "w.name", "w.slug", "w.description", "w.created_by_user_id", "w.created_at", "w.updated_at", "w.archived_at",
			"m.role",
			"COUNT(active_members.id)",
		).
		From("workspaces w").
		Join("workspace_members m ON m.workspace_id = w.id").
		LeftJoin("workspace_members active_members ON active_members.workspace_id = w.id AND active_members.status = 'active'").
		Where(sq.Eq{"m.user_id": uuidArg(userID), "m.status": string(MemberStatusActive)}).
		Where("w.archived_at IS NULL").
		GroupBy("w.id", "m.role").
		OrderBy("w.created_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list workspaces: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query workspaces: %w", err)
	}
	defer rows.Close()

	out := make([]WorkspaceSummary, 0)
	for rows.Next() {
		summary, err := scanWorkspaceSummary(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspaces: %w", err)
	}
	return out, nil
}

func (r *Repository) GetForUser(ctx context.Context, workspaceID, userID uuid.UUID) (*WorkspaceSummary, error) {
	query, args, err := r.db.Builder.
		Select(
			"w.id", "w.name", "w.slug", "w.description", "w.created_by_user_id", "w.created_at", "w.updated_at", "w.archived_at",
			"m.role",
			"(SELECT COUNT(*) FROM workspace_members active_members WHERE active_members.workspace_id = w.id AND active_members.status = 'active')",
		).
		From("workspaces w").
		Join("workspace_members m ON m.workspace_id = w.id").
		Where(sq.Eq{"w.id": uuidArg(workspaceID), "m.user_id": uuidArg(userID), "m.status": string(MemberStatusActive)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get workspace: %w", err)
	}
	return scanWorkspaceSummary(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetWorkspace(ctx context.Context, workspaceID uuid.UUID) (*Workspace, error) {
	query, args, err := r.db.Builder.
		Select(workspaceColumns).
		From("workspaces").
		Where(sq.Eq{"id": uuidArg(workspaceID)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get workspace by id: %w", err)
	}
	return scanWorkspace(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) UpdateWorkspace(ctx context.Context, workspaceID uuid.UUID, name *string, description *string) (*Workspace, error) {
	builder := r.db.Builder.
		Update("workspaces").
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": uuidArg(workspaceID)}).
		Where("archived_at IS NULL")
	if name != nil {
		builder = builder.Set("name", strings.TrimSpace(*name))
	}
	if description != nil {
		builder = builder.Set("description", textPtrArg(description))
	}
	query, args, err := builder.Suffix("RETURNING " + workspaceColumns).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build workspace update: %w", err)
	}
	return scanWorkspace(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ArchiveWorkspace(ctx context.Context, workspaceID uuid.UUID) (*Workspace, error) {
	query, args, err := r.db.Builder.
		Update("workspaces").
		Set("archived_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": uuidArg(workspaceID)}).
		Where("archived_at IS NULL").
		Suffix("RETURNING " + workspaceColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build workspace archive: %w", err)
	}
	return scanWorkspace(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetMemberByWorkspaceUser(ctx context.Context, workspaceID, userID uuid.UUID) (*WorkspaceMember, error) {
	query, args, err := r.db.Builder.
		Select(memberColumns).
		From("workspace_members").
		Where(sq.Eq{"workspace_id": uuidArg(workspaceID), "user_id": uuidArg(userID)}).
		Where(sq.NotEq{"status": string(MemberStatusRemoved)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get member by user: %w", err)
	}
	return scanMember(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetMemberByID(ctx context.Context, workspaceID, memberID uuid.UUID) (*WorkspaceMember, error) {
	query, args, err := r.db.Builder.
		Select(memberColumns).
		From("workspace_members").
		Where(sq.Eq{"workspace_id": uuidArg(workspaceID), "id": uuidArg(memberID)}).
		Where(sq.NotEq{"status": string(MemberStatusRemoved)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get member by id: %w", err)
	}
	return scanMember(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]WorkspaceMember, error) {
	query, args, err := r.db.Builder.
		Select(memberColumns).
		From("workspace_members").
		Where(sq.Eq{"workspace_id": uuidArg(workspaceID)}).
		Where(sq.NotEq{"status": string(MemberStatusRemoved)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list members: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query members: %w", err)
	}
	defer rows.Close()
	out := make([]WorkspaceMember, 0)
	for rows.Next() {
		member, err := scanMember(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate members: %w", err)
	}
	return out, nil
}

func (r *Repository) CountActiveOwners(ctx context.Context, workspaceID uuid.UUID) (int, error) {
	query, args, err := r.db.Builder.
		Select("COUNT(*)").
		From("workspace_members").
		Where(sq.Eq{"workspace_id": uuidArg(workspaceID), "role": string(RoleOwner), "status": string(MemberStatusActive)}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build owner count: %w", err)
	}
	var count int
	if err := r.db.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("scan owner count: %w", err)
	}
	return count, nil
}

func (r *Repository) HasActiveMember(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	query, args, err := r.db.Builder.
		Select("1").
		From("workspace_members").
		Where(sq.Eq{"workspace_id": uuidArg(workspaceID), "user_id": uuidArg(userID), "status": string(MemberStatusActive)}).
		Limit(1).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build active member check: %w", err)
	}
	var one int
	err = r.db.QueryRow(ctx, query, args...).Scan(&one)
	if storage.NoRowsFound(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("scan active member check: %w", err)
	}
	return true, nil
}

func (r *Repository) UpsertInvitation(ctx context.Context, invitation WorkspaceInvitation) (*WorkspaceInvitation, error) {
	query, args, err := r.db.Builder.
		Insert("workspace_invitations").
		Columns("id", "workspace_id", "email", "invited_user_id", "role", "status", "invited_by_user_id", "expires_at").
		Values(
			uuidArg(invitation.ID),
			uuidArg(invitation.WorkspaceID),
			invitation.Email,
			uuidPtrArg(invitation.InvitedUserID),
			string(invitation.Role),
			string(InvitationStatusPending),
			uuidArg(invitation.InvitedByUserID),
			timePtrArg(invitation.ExpiresAt),
		).
		Suffix(
			"ON CONFLICT (workspace_id, lower(email)) WHERE status = 'pending' DO UPDATE SET " +
				"role = EXCLUDED.role, invited_user_id = EXCLUDED.invited_user_id, invited_by_user_id = EXCLUDED.invited_by_user_id, updated_at = NOW() " +
				"RETURNING " + inviteReturningColumns,
		).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build upsert invitation: %w", err)
	}
	return scanInvitation(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) UpsertInvitedMember(ctx context.Context, member WorkspaceMember) (*WorkspaceMember, error) {
	query, args, err := r.db.Builder.
		Insert("workspace_members").
		Columns("id", "workspace_id", "user_id", "role", "status", "invited_by_user_id", "invited_at", "removed_at").
		Values(
			uuidArg(member.ID),
			uuidArg(member.WorkspaceID),
			uuidArg(member.UserID),
			string(member.Role),
			string(MemberStatusInvited),
			uuidPtrArg(member.InvitedByUserID),
			sq.Expr("NOW()"),
			nil,
		).
		Suffix(
			"ON CONFLICT (workspace_id, user_id) WHERE status IN ('active', 'invited') DO UPDATE SET " +
				"role = EXCLUDED.role, status = CASE WHEN workspace_members.status = 'active' THEN 'active' ELSE 'invited' END, " +
				"invited_by_user_id = EXCLUDED.invited_by_user_id, invited_at = COALESCE(workspace_members.invited_at, NOW()), removed_at = NULL, updated_at = NOW() " +
				"RETURNING " + memberColumns,
		).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build upsert invited member: %w", err)
	}
	return scanMember(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListInvitationsForUser(ctx context.Context, userID uuid.UUID, email string) ([]WorkspaceInvitation, error) {
	builder := r.db.Builder.
		Select(inviteSelectColumns).
		From("workspace_invitations i").
		Join("workspaces w ON w.id = i.workspace_id").
		Where(sq.Eq{"i.status": string(InvitationStatusPending)}).
		Where("w.archived_at IS NULL").
		OrderBy("i.created_at DESC")
	if strings.TrimSpace(email) != "" {
		builder = builder.Where(sq.Or{
			sq.Eq{"i.invited_user_id": uuidArg(userID)},
			sq.Eq{"lower(i.email)": strings.ToLower(strings.TrimSpace(email))},
		})
	} else {
		builder = builder.Where(sq.Eq{"i.invited_user_id": uuidArg(userID)})
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list invitations: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query invitations: %w", err)
	}
	defer rows.Close()
	out := make([]WorkspaceInvitation, 0)
	for rows.Next() {
		invitation, err := scanInvitation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *invitation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invitations: %w", err)
	}
	return out, nil
}

func (r *Repository) GetInvitation(ctx context.Context, invitationID uuid.UUID) (*WorkspaceInvitation, error) {
	query, args, err := r.db.Builder.
		Select(inviteSelectColumns).
		From("workspace_invitations i").
		Join("workspaces w ON w.id = i.workspace_id").
		Where(sq.Eq{"i.id": uuidArg(invitationID)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get invitation: %w", err)
	}
	return scanInvitation(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) AcceptInvitation(ctx context.Context, invitationID, userID uuid.UUID) (*WorkspaceInvitation, *WorkspaceMember, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("begin accept invitation tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	query, args, err := r.db.Builder.
		Update("workspace_invitations").
		Set("status", string(InvitationStatusAccepted)).
		Set("invited_user_id", uuidArg(userID)).
		Set("accepted_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": uuidArg(invitationID), "status": string(InvitationStatusPending)}).
		Suffix("RETURNING " + inviteReturningColumns).
		ToSql()
	if err != nil {
		return nil, nil, fmt.Errorf("build accept invitation: %w", err)
	}
	invitation, err := scanInvitation(tx.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, nil, err
	}

	query, args, err = r.db.Builder.
		Insert("workspace_members").
		Columns("id", "workspace_id", "user_id", "role", "status", "invited_by_user_id", "invited_at", "joined_at", "removed_at").
		Values(uuidArg(uuid.New()), uuidArg(invitation.WorkspaceID), uuidArg(userID), string(invitation.Role), string(MemberStatusActive), uuidArg(invitation.InvitedByUserID), sq.Expr("NOW()"), sq.Expr("NOW()"), nil).
		Suffix(
			"ON CONFLICT (workspace_id, user_id) WHERE status IN ('active', 'invited') DO UPDATE SET " +
				"role = EXCLUDED.role, status = 'active', joined_at = COALESCE(workspace_members.joined_at, NOW()), removed_at = NULL, updated_at = NOW() " +
				"RETURNING " + memberColumns,
		).
		ToSql()
	if err != nil {
		return nil, nil, fmt.Errorf("build accept member upsert: %w", err)
	}
	member, err := scanMember(tx.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit accept invitation tx: %w", err)
	}
	committed = true
	return invitation, member, nil
}

func (r *Repository) DeclineInvitation(ctx context.Context, invitationID uuid.UUID) (*WorkspaceInvitation, error) {
	query, args, err := r.db.Builder.
		Update("workspace_invitations").
		Set("status", string(InvitationStatusDeclined)).
		Set("declined_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": uuidArg(invitationID), "status": string(InvitationStatusPending)}).
		Suffix("RETURNING " + inviteReturningColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build decline invitation: %w", err)
	}
	return scanInvitation(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) UpdateMemberRole(ctx context.Context, workspaceID, memberID uuid.UUID, role Role) (*WorkspaceMember, error) {
	query, args, err := r.db.Builder.
		Update("workspace_members").
		Set("role", string(role)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"workspace_id": uuidArg(workspaceID), "id": uuidArg(memberID)}).
		Where(sq.NotEq{"status": string(MemberStatusRemoved)}).
		Suffix("RETURNING " + memberColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build member role update: %w", err)
	}
	return scanMember(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) RemoveMember(ctx context.Context, workspaceID, memberID uuid.UUID) (*WorkspaceMember, error) {
	query, args, err := r.db.Builder.
		Update("workspace_members").
		Set("status", string(MemberStatusRemoved)).
		Set("removed_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"workspace_id": uuidArg(workspaceID), "id": uuidArg(memberID)}).
		Where(sq.NotEq{"status": string(MemberStatusRemoved)}).
		Suffix("RETURNING " + memberColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build member remove: %w", err)
	}
	return scanMember(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) AccessCheck(ctx context.Context, userID, workspaceID uuid.UUID) (*WorkspaceAccess, error) {
	query, args, err := r.db.Builder.
		Select("m.role", "m.status", "w.archived_at").
		From("workspace_members m").
		Join("workspaces w ON w.id = m.workspace_id").
		Where(sq.Eq{"m.workspace_id": uuidArg(workspaceID), "m.user_id": uuidArg(userID)}).
		Where(sq.NotEq{"m.status": string(MemberStatusRemoved)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build access check: %w", err)
	}
	var role string
	var status string
	var archivedAt pgtype.Timestamp
	err = r.db.QueryRow(ctx, query, args...).Scan(&role, &status, &archivedAt)
	if storage.NoRowsFound(err) {
		return &WorkspaceAccess{HasAccess: false}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan access check: %w", err)
	}
	archived := archivedAt.Valid
	return &WorkspaceAccess{
		HasAccess:         status == string(MemberStatusActive) && !archived,
		Role:              Role(role),
		Status:            MemberStatus(status),
		WorkspaceArchived: archived,
	}, nil
}

func (r *Repository) ListForUserInternal(ctx context.Context, userID uuid.UUID) ([]WorkspaceAccess, []uuid.UUID, error) {
	query, args, err := r.db.Builder.
		Select("w.id", "m.role", "m.status", "w.archived_at").
		From("workspace_members m").
		Join("workspaces w ON w.id = m.workspace_id").
		Where(sq.Eq{"m.user_id": uuidArg(userID), "m.status": string(MemberStatusActive)}).
		Where("w.archived_at IS NULL").
		ToSql()
	if err != nil {
		return nil, nil, fmt.Errorf("build internal list workspaces: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("query internal workspaces: %w", err)
	}
	defer rows.Close()
	accesses := make([]WorkspaceAccess, 0)
	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id pgtype.UUID
		var role, status string
		var archivedAt pgtype.Timestamp
		if err := rows.Scan(&id, &role, &status, &archivedAt); err != nil {
			return nil, nil, fmt.Errorf("scan internal workspace: %w", err)
		}
		ids = append(ids, uuid.UUID(id.Bytes))
		accesses = append(accesses, WorkspaceAccess{
			HasAccess:         true,
			Role:              Role(role),
			Status:            MemberStatus(status),
			WorkspaceArchived: archivedAt.Valid,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate internal workspaces: %w", err)
	}
	return accesses, ids, nil
}

func (r *Repository) BatchInfo(ctx context.Context, ids []uuid.UUID) ([]WorkspaceInfo, error) {
	if len(ids) == 0 {
		return []WorkspaceInfo{}, nil
	}
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, uuidArg(id))
	}
	query, args, err := r.db.Builder.
		Select("id", "name", "slug", "archived_at").
		From("workspaces").
		Where(sq.Eq{"id": args}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build batch workspace info: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query batch workspace info: %w", err)
	}
	defer rows.Close()
	out := make([]WorkspaceInfo, 0)
	for rows.Next() {
		var id pgtype.UUID
		var name, slug string
		var archivedAt pgtype.Timestamp
		if err := rows.Scan(&id, &name, &slug, &archivedAt); err != nil {
			return nil, fmt.Errorf("scan workspace info: %w", err)
		}
		out = append(out, WorkspaceInfo{
			ID:       uuid.UUID(id.Bytes),
			Name:     name,
			Slug:     slug,
			Archived: archivedAt.Valid,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace info: %w", err)
	}
	return out, nil
}

func scanWorkspace(row pgx.Row) (*Workspace, error) {
	var id, createdBy pgtype.UUID
	var name, slug string
	var description pgtype.Text
	var createdAt, updatedAt, archivedAt pgtype.Timestamp
	if err := row.Scan(&id, &name, &slug, &description, &createdBy, &createdAt, &updatedAt, &archivedAt); err != nil {
		if storage.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan workspace: %w", err)
	}
	return &Workspace{
		ID:              uuid.UUID(id.Bytes),
		Name:            name,
		Slug:            slug,
		Description:     textPtr(description),
		CreatedByUserID: uuid.UUID(createdBy.Bytes),
		CreatedAt:       createdAt.Time,
		UpdatedAt:       updatedAt.Time,
		ArchivedAt:      timePtr(archivedAt),
	}, nil
}

func scanWorkspaceSummary(row pgx.Row) (*WorkspaceSummary, error) {
	var id, createdBy pgtype.UUID
	var name, slug, role string
	var description pgtype.Text
	var createdAt, updatedAt, archivedAt pgtype.Timestamp
	var memberCount int
	if err := row.Scan(&id, &name, &slug, &description, &createdBy, &createdAt, &updatedAt, &archivedAt, &role, &memberCount); err != nil {
		if storage.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan workspace summary: %w", err)
	}
	return &WorkspaceSummary{
		Workspace: Workspace{
			ID:              uuid.UUID(id.Bytes),
			Name:            name,
			Slug:            slug,
			Description:     textPtr(description),
			CreatedByUserID: uuid.UUID(createdBy.Bytes),
			CreatedAt:       createdAt.Time,
			UpdatedAt:       updatedAt.Time,
			ArchivedAt:      timePtr(archivedAt),
		},
		CurrentUserRole: Role(role),
		MemberCount:     memberCount,
	}, nil
}

func scanMember(row pgx.Row) (*WorkspaceMember, error) {
	var id, workspaceID, userID, invitedBy pgtype.UUID
	var role, status string
	var invitedAt, joinedAt, removedAt, createdAt, updatedAt pgtype.Timestamp
	if err := row.Scan(&id, &workspaceID, &userID, &role, &status, &invitedBy, &invitedAt, &joinedAt, &removedAt, &createdAt, &updatedAt); err != nil {
		if storage.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan workspace member: %w", err)
	}
	return &WorkspaceMember{
		ID:              uuid.UUID(id.Bytes),
		WorkspaceID:     uuid.UUID(workspaceID.Bytes),
		UserID:          uuid.UUID(userID.Bytes),
		Role:            Role(role),
		Status:          MemberStatus(status),
		InvitedByUserID: uuidPtr(invitedBy),
		InvitedAt:       timePtr(invitedAt),
		JoinedAt:        timePtr(joinedAt),
		RemovedAt:       timePtr(removedAt),
		CreatedAt:       createdAt.Time,
		UpdatedAt:       updatedAt.Time,
	}, nil
}

func scanInvitation(row pgx.Row) (*WorkspaceInvitation, error) {
	var id, workspaceID, invitedUserID, invitedBy pgtype.UUID
	var workspaceName, email, role, status string
	var expiresAt, acceptedAt, declinedAt, revokedAt, createdAt, updatedAt pgtype.Timestamp
	if err := row.Scan(&id, &workspaceID, &workspaceName, &email, &invitedUserID, &role, &status, &invitedBy, &expiresAt, &acceptedAt, &declinedAt, &revokedAt, &createdAt, &updatedAt); err != nil {
		if storage.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan workspace invitation: %w", err)
	}
	return &WorkspaceInvitation{
		ID:              uuid.UUID(id.Bytes),
		WorkspaceID:     uuid.UUID(workspaceID.Bytes),
		WorkspaceName:   workspaceName,
		Email:           email,
		InvitedUserID:   uuidPtr(invitedUserID),
		Role:            Role(role),
		Status:          InvitationStatus(status),
		InvitedByUserID: uuid.UUID(invitedBy.Bytes),
		ExpiresAt:       timePtr(expiresAt),
		AcceptedAt:      timePtr(acceptedAt),
		DeclinedAt:      timePtr(declinedAt),
		RevokedAt:       timePtr(revokedAt),
		CreatedAt:       createdAt.Time,
		UpdatedAt:       updatedAt.Time,
	}, nil
}

func mapWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrConflict
	}
	return err
}

func uuidArg(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func uuidPtrArg(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	return uuidArg(*id)
}

func uuidPtr(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}
	id := uuid.UUID(value.Bytes)
	return &id
}

func textPtrArg(value *string) pgtype.Text {
	if value == nil || strings.TrimSpace(*value) == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: strings.TrimSpace(*value), Valid: true}
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	v := value.String
	return &v
}

func timePtrArg(value *time.Time) pgtype.Timestamp {
	if value == nil {
		return pgtype.Timestamp{Valid: false}
	}
	return pgtype.Timestamp{Time: *value, Valid: true}
}

func timePtr(value pgtype.Timestamp) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}
