package workspaces

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"go.uber.org/zap"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/authusers"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/domain/errs"
	notify "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/notifications"
)

const (
	maxWorkspaceNameLength        = 80
	minWorkspaceNameLength        = 2
	maxWorkspaceDescriptionLength = 500
	maxSlugAttempts               = 20
)

var slugCollapsePattern = regexp.MustCompile(`-+`)

type repository interface {
	CreateWorkspaceWithOwner(ctx context.Context, workspace Workspace, owner WorkspaceMember) (*WorkspaceSummary, error)
	SlugExists(ctx context.Context, slug string) (bool, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]WorkspaceSummary, error)
	GetForUser(ctx context.Context, workspaceID, userID uuid.UUID) (*WorkspaceSummary, error)
	GetWorkspace(ctx context.Context, workspaceID uuid.UUID) (*Workspace, error)
	UpdateWorkspace(ctx context.Context, workspaceID uuid.UUID, name *string, description *string) (*Workspace, error)
	ArchiveWorkspace(ctx context.Context, workspaceID uuid.UUID) (*Workspace, error)
	GetMemberByWorkspaceUser(ctx context.Context, workspaceID, userID uuid.UUID) (*WorkspaceMember, error)
	GetMemberByID(ctx context.Context, workspaceID, memberID uuid.UUID) (*WorkspaceMember, error)
	ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]WorkspaceMember, error)
	CountActiveOwners(ctx context.Context, workspaceID uuid.UUID) (int, error)
	HasActiveMember(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error)
	UpsertInvitation(ctx context.Context, invitation WorkspaceInvitation) (*WorkspaceInvitation, error)
	UpsertInvitedMember(ctx context.Context, member WorkspaceMember) (*WorkspaceMember, error)
	ListInvitationsForUser(ctx context.Context, userID uuid.UUID, email string) ([]WorkspaceInvitation, error)
	GetInvitation(ctx context.Context, invitationID uuid.UUID) (*WorkspaceInvitation, error)
	AcceptInvitation(ctx context.Context, invitationID, userID uuid.UUID) (*WorkspaceInvitation, *WorkspaceMember, error)
	DeclineInvitation(ctx context.Context, invitationID uuid.UUID) (*WorkspaceInvitation, error)
	UpdateMemberRole(ctx context.Context, workspaceID, memberID uuid.UUID, role Role) (*WorkspaceMember, error)
	RemoveMember(ctx context.Context, workspaceID, memberID uuid.UUID) (*WorkspaceMember, error)
	AccessCheck(ctx context.Context, userID, workspaceID uuid.UUID) (*WorkspaceAccess, error)
	ListForUserInternal(ctx context.Context, userID uuid.UUID) ([]WorkspaceAccess, []uuid.UUID, error)
	BatchInfo(ctx context.Context, ids []uuid.UUID) ([]WorkspaceInfo, error)
}

type authUserLookup interface {
	LookupByEmail(ctx context.Context, email string) (*authusers.User, error)
	BatchByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]authusers.User, error)
}

type notifier interface {
	CreateNotifications(ctx context.Context, notifications []notify.CreateInput) error
}

type Service struct {
	repo                   repository
	userLookup             authUserLookup
	notifier               notifier
	notificationsEnabled   bool
	notificationsFailOpen  bool
	notificationWebBaseURL string
	log                    *zap.Logger
}

type Option func(*Service)

func WithUserLookup(lookup authUserLookup) Option {
	return func(s *Service) {
		s.userLookup = lookup
	}
}

func WithNotifications(client notifier, enabled, failOpen bool, publicWebBaseURL string) Option {
	return func(s *Service) {
		s.notifier = client
		s.notificationsEnabled = enabled
		s.notificationsFailOpen = failOpen
		s.notificationWebBaseURL = strings.TrimRight(strings.TrimSpace(publicWebBaseURL), "/")
	}
}

func NewService(repo repository, log *zap.Logger, opts ...Option) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	s := &Service{repo: repo, notificationsFailOpen: true, log: log}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*WorkspaceSummary, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	name, err := normalizeWorkspaceName(in.Name)
	if err != nil {
		return nil, err
	}
	description, err := normalizeDescription(in.Description)
	if err != nil {
		return nil, err
	}
	slug, err := s.uniqueSlug(ctx, name)
	if err != nil {
		return nil, err
	}
	workspaceID := uuid.New()
	return s.repo.CreateWorkspaceWithOwner(ctx, Workspace{
		ID:              workspaceID,
		Name:            name,
		Slug:            slug,
		Description:     description,
		CreatedByUserID: user.ID,
	}, WorkspaceMember{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		UserID:      user.ID,
		Role:        RoleOwner,
		Status:      MemberStatusActive,
	})
}

func (s *Service) List(ctx context.Context) ([]WorkspaceSummary, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListForUser(ctx, user.ID)
}

func (s *Service) Get(ctx context.Context, workspaceID uuid.UUID) (*WorkspaceSummary, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.GetForUser(ctx, workspaceID, user.ID)
}

func (s *Service) Update(ctx context.Context, workspaceID uuid.UUID, in UpdateInput) (*WorkspaceSummary, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	member, err := s.requireActiveMember(ctx, workspaceID, user.ID)
	if err != nil {
		return nil, err
	}
	if !CanManageWorkspace(member.Role) {
		return nil, ErrForbidden
	}
	var name *string
	if in.Name != nil {
		normalized, err := normalizeWorkspaceName(*in.Name)
		if err != nil {
			return nil, err
		}
		name = &normalized
	}
	description, err := normalizeDescription(in.Description)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.UpdateWorkspace(ctx, workspaceID, name, description); err != nil {
		return nil, err
	}
	return s.repo.GetForUser(ctx, workspaceID, user.ID)
}

func (s *Service) Archive(ctx context.Context, workspaceID uuid.UUID) (*WorkspaceSummary, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	member, err := s.requireActiveMember(ctx, workspaceID, user.ID)
	if err != nil {
		return nil, err
	}
	if !CanArchiveWorkspace(member.Role) {
		return nil, ErrForbidden
	}
	if _, err := s.repo.ArchiveWorkspace(ctx, workspaceID); err != nil {
		return nil, err
	}
	return s.repo.GetForUser(ctx, workspaceID, user.ID)
}

func (s *Service) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]WorkspaceMemberInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireActiveMember(ctx, workspaceID, user.ID); err != nil {
		return nil, err
	}
	members, err := s.repo.ListMembers(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	usersByID := map[uuid.UUID]authusers.User{}
	if s.userLookup != nil {
		ids := make([]uuid.UUID, 0, len(members))
		for _, member := range members {
			ids = append(ids, member.UserID)
		}
		usersByID, err = s.userLookup.BatchByIDs(ctx, ids)
		if err != nil {
			s.log.Warn("workspace member user enrichment failed", zap.Error(err))
		}
	}
	out := make([]WorkspaceMemberInfo, 0, len(members))
	for _, member := range members {
		info := WorkspaceMemberInfo{Member: member}
		if resolved, ok := usersByID[member.UserID]; ok {
			info.Email = optionalString(resolved.Email)
			info.DisplayName = optionalString(resolved.DisplayName)
		}
		out = append(out, info)
	}
	return out, nil
}

func (s *Service) ListMembersInternal(ctx context.Context, workspaceID uuid.UUID) ([]WorkspaceMember, error) {
	members, err := s.repo.ListMembers(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	out := make([]WorkspaceMember, 0, len(members))
	for _, member := range members {
		if member.Status != MemberStatusActive {
			continue
		}
		out = append(out, member)
	}
	return out, nil
}

func (s *Service) InviteMember(ctx context.Context, workspaceID uuid.UUID, in InviteInput) (*WorkspaceInvitation, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	actor, err := s.requireActiveMember(ctx, workspaceID, user.ID)
	if err != nil {
		return nil, err
	}
	if !CanInviteMembers(actor.Role) {
		return nil, ErrForbidden
	}
	role, err := normalizeInviteRole(in.Role)
	if err != nil {
		return nil, err
	}
	if actor.Role == RoleAdmin && role == RoleAdmin {
		return nil, ErrForbidden
	}
	email, err := normalizeEmail(in.Email)
	if err != nil {
		return nil, err
	}

	var invitedUserID *uuid.UUID
	if s.userLookup != nil {
		resolved, lookupErr := s.userLookup.LookupByEmail(ctx, email)
		if lookupErr != nil && !errors.Is(lookupErr, domainerrs.ErrNotFound) {
			return nil, lookupErr
		}
		if resolved != nil {
			if resolved.UserID == user.ID {
				return nil, apperrs.NewInvalidInput("cannot invite yourself")
			}
			active, err := s.repo.HasActiveMember(ctx, workspaceID, resolved.UserID)
			if err != nil {
				return nil, err
			}
			if active {
				return nil, ErrAlreadyMember
			}
			invitedUserID = &resolved.UserID
		}
	}

	invitation, err := s.repo.UpsertInvitation(ctx, WorkspaceInvitation{
		ID:              uuid.New(),
		WorkspaceID:     workspaceID,
		Email:           email,
		InvitedUserID:   invitedUserID,
		Role:            role,
		Status:          InvitationStatusPending,
		InvitedByUserID: user.ID,
	})
	if err != nil {
		return nil, err
	}
	if invitedUserID != nil {
		if _, err := s.repo.UpsertInvitedMember(ctx, WorkspaceMember{
			ID:              uuid.New(),
			WorkspaceID:     workspaceID,
			UserID:          *invitedUserID,
			Role:            role,
			Status:          MemberStatusInvited,
			InvitedByUserID: &user.ID,
		}); err != nil {
			return nil, err
		}
		s.notify(ctx, []notify.CreateInput{workspaceNotification(
			*invitedUserID,
			&user.ID,
			NotificationWorkspaceInvited,
			"Workspace invitation",
			fmt.Sprintf("You were invited to join %s.", invitation.WorkspaceName),
			invitation.WorkspaceID,
			map[string]any{
				"workspaceId":   invitation.WorkspaceID.String(),
				"workspaceName": invitation.WorkspaceName,
				"role":          string(role),
				"url":           workspaceInvitationsURL(s.notificationWebBaseURL),
			},
		)})
	}
	return invitation, nil
}

func (s *Service) ListInvitations(ctx context.Context) ([]WorkspaceInvitation, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListInvitationsForUser(ctx, user.ID, user.Email)
}

func (s *Service) AcceptInvitation(ctx context.Context, invitationID uuid.UUID) (*WorkspaceSummary, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	invitation, err := s.repo.GetInvitation(ctx, invitationID)
	if err != nil {
		return nil, err
	}
	if err := validateInvitationForUser(invitation, user); err != nil {
		return nil, err
	}
	accepted, _, err := s.repo.AcceptInvitation(ctx, invitationID, user.ID)
	if err != nil {
		return nil, err
	}
	s.notify(ctx, []notify.CreateInput{workspaceNotification(
		accepted.InvitedByUserID,
		&user.ID,
		NotificationWorkspaceInvitationAccepted,
		"Workspace invitation accepted",
		fmt.Sprintf("%s joined %s.", userDisplay(user), accepted.WorkspaceName),
		accepted.WorkspaceID,
		map[string]any{
			"workspaceId":   accepted.WorkspaceID.String(),
			"workspaceName": accepted.WorkspaceName,
			"role":          string(accepted.Role),
			"url":           workspaceURL(s.notificationWebBaseURL, accepted.WorkspaceID),
		},
	)})
	return s.repo.GetForUser(ctx, accepted.WorkspaceID, user.ID)
}

func (s *Service) DeclineInvitation(ctx context.Context, invitationID uuid.UUID) error {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return err
	}
	invitation, err := s.repo.GetInvitation(ctx, invitationID)
	if err != nil {
		return err
	}
	if err := validateInvitationForUser(invitation, user); err != nil {
		return err
	}
	declined, err := s.repo.DeclineInvitation(ctx, invitationID)
	if err != nil {
		return err
	}
	s.notify(ctx, []notify.CreateInput{workspaceNotification(
		declined.InvitedByUserID,
		&user.ID,
		NotificationWorkspaceInvitationDeclined,
		"Workspace invitation declined",
		fmt.Sprintf("%s declined the invitation to %s.", userDisplay(user), declined.WorkspaceName),
		declined.WorkspaceID,
		map[string]any{
			"workspaceId":   declined.WorkspaceID.String(),
			"workspaceName": declined.WorkspaceName,
		},
	)})
	return nil
}

func (s *Service) UpdateMember(ctx context.Context, workspaceID, memberID uuid.UUID, in UpdateMemberInput) (*WorkspaceMemberInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	actor, err := s.requireActiveMember(ctx, workspaceID, user.ID)
	if err != nil {
		return nil, err
	}
	target, err := s.repo.GetMemberByID(ctx, workspaceID, memberID)
	if err != nil {
		return nil, err
	}
	role, err := normalizeMemberRole(in.Role)
	if err != nil {
		return nil, err
	}
	if err := canChangeRole(actor, target, role); err != nil {
		return nil, err
	}
	updated, err := s.repo.UpdateMemberRole(ctx, workspaceID, memberID, role)
	if err != nil {
		return nil, err
	}
	workspace, _ := s.repo.GetWorkspace(ctx, workspaceID)
	workspaceName := "workspace"
	if workspace != nil {
		workspaceName = workspace.Name
	}
	s.notify(ctx, []notify.CreateInput{workspaceNotification(
		updated.UserID,
		&user.ID,
		NotificationWorkspaceRoleChanged,
		"Workspace role changed",
		fmt.Sprintf("Your role for %s changed to %s.", workspaceName, role),
		workspaceID,
		map[string]any{
			"workspaceId":   workspaceID.String(),
			"workspaceName": workspaceName,
			"oldRole":       string(target.Role),
			"newRole":       string(role),
			"url":           workspaceURL(s.notificationWebBaseURL, workspaceID),
		},
	)})
	return &WorkspaceMemberInfo{Member: *updated}, nil
}

func (s *Service) RemoveMember(ctx context.Context, workspaceID, memberID uuid.UUID) error {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return err
	}
	actor, err := s.requireActiveMember(ctx, workspaceID, user.ID)
	if err != nil {
		return err
	}
	target, err := s.repo.GetMemberByID(ctx, workspaceID, memberID)
	if err != nil {
		return err
	}
	if err := s.canRemoveMember(ctx, actor, target); err != nil {
		return err
	}
	removed, err := s.repo.RemoveMember(ctx, workspaceID, memberID)
	if err != nil {
		return err
	}
	if removed.UserID != user.ID {
		workspace, _ := s.repo.GetWorkspace(ctx, workspaceID)
		workspaceName := "workspace"
		if workspace != nil {
			workspaceName = workspace.Name
		}
		s.notify(ctx, []notify.CreateInput{workspaceNotification(
			removed.UserID,
			&user.ID,
			NotificationWorkspaceMemberRemoved,
			"Workspace access removed",
			fmt.Sprintf("You were removed from %s.", workspaceName),
			workspaceID,
			map[string]any{
				"workspaceId":   workspaceID.String(),
				"workspaceName": workspaceName,
			},
		)})
	}
	return nil
}

func (s *Service) AccessCheck(ctx context.Context, userID, workspaceID uuid.UUID) (*WorkspaceAccess, error) {
	return s.repo.AccessCheck(ctx, userID, workspaceID)
}

func (s *Service) ListForUserInternal(ctx context.Context, userID uuid.UUID) ([]struct {
	ID   uuid.UUID
	Role Role
}, error) {
	accesses, ids, err := s.repo.ListForUserInternal(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]struct {
		ID   uuid.UUID
		Role Role
	}, 0, len(ids))
	for i, id := range ids {
		out = append(out, struct {
			ID   uuid.UUID
			Role Role
		}{ID: id, Role: accesses[i].Role})
	}
	return out, nil
}

func (s *Service) BatchInfo(ctx context.Context, ids []uuid.UUID) ([]WorkspaceInfo, error) {
	return s.repo.BatchInfo(ctx, ids)
}

func (s *Service) requireActiveMember(ctx context.Context, workspaceID, userID uuid.UUID) (*WorkspaceMember, error) {
	workspace, err := s.repo.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if workspace.Archived() {
		return nil, domainerrs.ErrNotFound
	}
	member, err := s.repo.GetMemberByWorkspaceUser(ctx, workspaceID, userID)
	if err != nil {
		return nil, err
	}
	if member.Status != MemberStatusActive {
		return nil, domainerrs.ErrNotFound
	}
	return member, nil
}

func (s *Service) canRemoveMember(ctx context.Context, actor, target *WorkspaceMember) error {
	if actor.UserID == target.UserID {
		if target.Role == RoleOwner {
			count, err := s.repo.CountActiveOwners(ctx, target.WorkspaceID)
			if err != nil {
				return err
			}
			if count <= 1 {
				return ErrLastOwner
			}
		}
		return nil
	}
	if !CanManageWorkspace(actor.Role) {
		return ErrForbidden
	}
	if target.Role == RoleOwner {
		return ErrForbidden
	}
	if actor.Role == RoleAdmin && (target.Role == RoleAdmin || target.Role == RoleOwner) {
		return ErrForbidden
	}
	return nil
}

func canChangeRole(actor, target *WorkspaceMember, role Role) error {
	if !CanManageWorkspace(actor.Role) {
		return ErrForbidden
	}
	if target.Role == RoleOwner || role == RoleOwner {
		return ErrForbidden
	}
	if actor.Role == RoleAdmin && (target.Role == RoleAdmin || role == RoleAdmin) {
		return ErrForbidden
	}
	return nil
}

func validateInvitationForUser(invitation *WorkspaceInvitation, user auth.AuthenticatedUser) error {
	if invitation.Status != InvitationStatusPending {
		return domainerrs.ErrNotFound
	}
	if invitation.ExpiresAt != nil && invitation.ExpiresAt.Before(time.Now().UTC()) {
		return domainerrs.ErrNotFound
	}
	if invitation.InvitedUserID != nil && *invitation.InvitedUserID == user.ID {
		return nil
	}
	if strings.TrimSpace(user.Email) != "" && strings.EqualFold(invitation.Email, user.Email) {
		return nil
	}
	return ErrInvalidInvitee
}

func normalizeWorkspaceName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	length := utf8.RuneCountInString(trimmed)
	if length < minWorkspaceNameLength || length > maxWorkspaceNameLength {
		return "", apperrs.NewInvalidInput("name must be between %d and %d characters", minWorkspaceNameLength, maxWorkspaceNameLength)
	}
	return trimmed, nil
}

func normalizeDescription(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(trimmed) > maxWorkspaceDescriptionLength {
		return nil, apperrs.NewInvalidInput("description must be at most %d characters", maxWorkspaceDescriptionLength)
	}
	return &trimmed, nil
}

func normalizeInviteRole(role Role) (Role, error) {
	switch role {
	case RoleAdmin, RoleMember, RoleViewer:
		return role, nil
	default:
		return "", apperrs.NewInvalidInput("role must be one of: admin member viewer")
	}
}

func normalizeMemberRole(role Role) (Role, error) {
	switch role {
	case RoleAdmin, RoleMember, RoleViewer:
		return role, nil
	default:
		return "", apperrs.NewInvalidInput("role must be one of: admin member viewer")
	}
}

func normalizeEmail(email string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return "", apperrs.NewInvalidInput("email is required")
	}
	if _, err := mail.ParseAddress(normalized); err != nil {
		return "", apperrs.NewInvalidInput("email must be a valid email address")
	}
	return normalized, nil
}

func (s *Service) uniqueSlug(ctx context.Context, name string) (string, error) {
	base := slugify(name)
	if base == "" {
		base = "workspace"
	}
	for attempt := 0; attempt < maxSlugAttempts; attempt++ {
		candidate := base
		if attempt > 0 {
			candidate = fmt.Sprintf("%s-%d", base, attempt+1)
		}
		exists, err := s.repo.SlugExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", ErrConflict
}

func slugify(value string) string {
	var b strings.Builder
	lastHyphen := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastHyphen = false
		case r == '-' || unicode.IsSpace(r):
			if !lastHyphen {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	return strings.Trim(slugCollapsePattern.ReplaceAllString(b.String(), "-"), "-")
}

func (s *Service) notify(ctx context.Context, inputs []notify.CreateInput) {
	if !s.notificationsEnabled || s.notifier == nil || len(inputs) == 0 {
		return
	}
	if err := s.notifier.CreateNotifications(ctx, inputs); err != nil {
		if s.notificationsFailOpen {
			s.log.Warn("workspace notification failed", zap.Error(err))
			return
		}
		s.log.Error("workspace notification failed", zap.Error(err))
	}
}

func workspaceNotification(userID uuid.UUID, actorUserID *uuid.UUID, notificationType, title, message string, workspaceID uuid.UUID, metadata map[string]any) notify.CreateInput {
	entityType := EntityWorkspace
	return notify.CreateInput{
		UserID:      userID,
		ActorUserID: actorUserID,
		Type:        notificationType,
		Title:       title,
		Message:     message,
		EntityType:  &entityType,
		EntityID:    &workspaceID,
		Metadata:    metadata,
	}
}

func workspaceURL(base string, workspaceID uuid.UUID) string {
	if strings.TrimSpace(base) == "" {
		return "/workspaces/" + workspaceID.String()
	}
	return strings.TrimRight(base, "/") + "/workspaces/" + workspaceID.String()
}

func workspaceInvitationsURL(base string) string {
	if strings.TrimSpace(base) == "" {
		return "/workspace-invitations"
	}
	return strings.TrimRight(base, "/") + "/workspace-invitations"
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func userDisplay(user auth.AuthenticatedUser) string {
	if strings.TrimSpace(user.Email) != "" {
		return user.Email
	}
	return "A member"
}
