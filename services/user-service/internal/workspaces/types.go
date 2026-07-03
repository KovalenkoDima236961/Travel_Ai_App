package workspaces

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleViewer Role = "viewer"
)

type MemberStatus string

const (
	MemberStatusActive  MemberStatus = "active"
	MemberStatusInvited MemberStatus = "invited"
	MemberStatusRemoved MemberStatus = "removed"
)

type InvitationStatus string

const (
	InvitationStatusPending  InvitationStatus = "pending"
	InvitationStatusAccepted InvitationStatus = "accepted"
	InvitationStatusDeclined InvitationStatus = "declined"
	InvitationStatusRevoked  InvitationStatus = "revoked"
	InvitationStatusExpired  InvitationStatus = "expired"
)

const (
	NotificationWorkspaceInvited            = "workspace_invited"
	NotificationWorkspaceInvitationAccepted = "workspace_invitation_accepted"
	NotificationWorkspaceInvitationDeclined = "workspace_invitation_declined"
	NotificationWorkspaceMemberRemoved      = "workspace_member_removed"
	NotificationWorkspaceRoleChanged        = "workspace_role_changed"
	NotificationWorkspaceTripCreated        = "workspace_trip_created"

	EntityWorkspace = "workspace"
)

var (
	ErrForbidden      = errors.New("workspace permission denied")
	ErrConflict       = errors.New("workspace conflict")
	ErrAlreadyMember  = errors.New("user is already a workspace member")
	ErrLastOwner      = errors.New("workspace must keep at least one owner")
	ErrInvalidInvitee = errors.New("invitation does not belong to current user")
)

type Workspace struct {
	ID              uuid.UUID
	Name            string
	Slug            string
	Description     *string
	CreatedByUserID uuid.UUID
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ArchivedAt      *time.Time
}

func (w Workspace) Archived() bool {
	return w.ArchivedAt != nil
}

type WorkspaceSummary struct {
	Workspace       Workspace
	CurrentUserRole Role
	MemberCount     int
}

type WorkspaceMember struct {
	ID              uuid.UUID
	WorkspaceID     uuid.UUID
	UserID          uuid.UUID
	Role            Role
	Status          MemberStatus
	InvitedByUserID *uuid.UUID
	InvitedAt       *time.Time
	JoinedAt        *time.Time
	RemovedAt       *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type WorkspaceMemberInfo struct {
	Member      WorkspaceMember
	Email       *string
	DisplayName *string
}

type WorkspaceInvitation struct {
	ID              uuid.UUID
	WorkspaceID     uuid.UUID
	WorkspaceName   string
	Email           string
	InvitedUserID   *uuid.UUID
	Role            Role
	Status          InvitationStatus
	InvitedByUserID uuid.UUID
	ExpiresAt       *time.Time
	AcceptedAt      *time.Time
	DeclinedAt      *time.Time
	RevokedAt       *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type WorkspaceAccess struct {
	HasAccess         bool
	Role              Role
	Status            MemberStatus
	WorkspaceArchived bool
}

type WorkspaceInfo struct {
	ID       uuid.UUID
	Name     string
	Slug     string
	Archived bool
}

type CreateInput struct {
	Name        string
	Description *string
}

type UpdateInput struct {
	Name        *string
	Description *string
}

type InviteInput struct {
	Email string
	Role  Role
}

type UpdateMemberInput struct {
	Role Role
}

func CanManageWorkspace(role Role) bool {
	return role == RoleOwner || role == RoleAdmin
}

func CanArchiveWorkspace(role Role) bool {
	return role == RoleOwner
}

func CanInviteMembers(role Role) bool {
	return role == RoleOwner || role == RoleAdmin
}

func CanCreateWorkspaceTrip(role Role) bool {
	return role == RoleOwner || role == RoleAdmin || role == RoleMember
}

func CanEditWorkspaceTrip(role Role) bool {
	return role == RoleOwner || role == RoleAdmin || role == RoleMember
}

func CanViewWorkspaceTrip(role Role) bool {
	return role == RoleOwner || role == RoleAdmin || role == RoleMember || role == RoleViewer
}
