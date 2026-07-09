package workspacepolicies

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspaces"
)

type policyTestRepository struct {
	active  *Policy
	upserts int
}

func (r *policyTestRepository) UpsertActive(
	_ context.Context,
	workspaceID, actorUserID uuid.UUID,
	input UpsertInput,
) (*Policy, error) {
	r.upserts++
	r.active = &Policy{
		ID: uuid.New(), WorkspaceID: workspaceID, Name: input.Name,
		Description: input.Description, Rules: input.Rules, Status: "active",
		CreatedByUserID: actorUserID,
	}
	return r.active, nil
}

func (r *policyTestRepository) GetActive(context.Context, uuid.UUID) (*Policy, error) {
	if r.active == nil {
		return nil, domainerrs.ErrNotFound
	}
	return r.active, nil
}

func (r *policyTestRepository) GetByID(
	context.Context,
	uuid.UUID,
	uuid.UUID,
) (*Policy, error) {
	return r.GetActive(context.Background(), uuid.Nil)
}

func (r *policyTestRepository) ArchiveActive(
	_ context.Context,
	_ uuid.UUID,
	actorUserID uuid.UUID,
) (*Policy, error) {
	if r.active == nil {
		return nil, domainerrs.ErrNotFound
	}
	r.active.Status = "archived"
	r.active.ArchivedByUserID = &actorUserID
	return r.active, nil
}

type policyTestAccess struct {
	roles map[uuid.UUID]workspaces.Role
}

func (a policyTestAccess) AccessCheck(
	_ context.Context,
	userID, _ uuid.UUID,
) (*workspaces.Access, error) {
	role, ok := a.roles[userID]
	return &workspaces.Access{HasAccess: ok, Role: role, Status: "active"}, nil
}

func TestServicePermissionsAndLifecycle(t *testing.T) {
	ownerID, memberID, outsiderID := uuid.New(), uuid.New(), uuid.New()
	workspaceID := uuid.New()
	repository := &policyTestRepository{}
	service := New(repository, policyTestAccess{roles: map[uuid.UUID]workspaces.Role{
		ownerID: workspaces.RoleOwner, memberID: workspaces.RoleMember,
	}})

	memberContext := auth.WithUser(context.Background(), auth.AuthenticatedUser{ID: memberID})
	response, err := service.Get(memberContext, workspaceID)
	if err != nil || response.Policy != nil || response.Defaults == nil {
		t.Fatalf("member get defaults: response=%#v err=%v", response, err)
	}
	input := UpsertInput{Name: "Default policy", Rules: DefaultRules()}
	if _, err := service.Upsert(memberContext, workspaceID, input); !errors.Is(err, apperrs.ErrForbidden) {
		t.Fatalf("member upsert error = %v, want forbidden", err)
	}

	ownerContext := auth.WithUser(context.Background(), auth.AuthenticatedUser{ID: ownerID})
	policy, err := service.Upsert(ownerContext, workspaceID, input)
	if err != nil || policy.Status != "active" || repository.upserts != 1 {
		t.Fatalf("owner upsert: policy=%#v err=%v", policy, err)
	}
	if _, err := service.Archive(ownerContext, workspaceID); err != nil {
		t.Fatalf("owner archive: %v", err)
	}

	outsiderContext := auth.WithUser(context.Background(), auth.AuthenticatedUser{ID: outsiderID})
	if _, err := service.Get(outsiderContext, workspaceID); !errors.Is(err, apperrs.ErrForbidden) {
		t.Fatalf("outsider get error = %v, want forbidden", err)
	}
}
