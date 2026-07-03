package workspaces

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/application/errs"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/domain/errs"
)

type Handler struct {
	svc *Service
	log *zap.Logger
}

func NewHandler(svc *Service, log *zap.Logger) *Handler {
	if log == nil {
		log = zap.NewNop()
	}
	return &Handler{svc: svc, log: log}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/workspaces", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/{workspaceId}", h.Get)
		r.Patch("/{workspaceId}", h.Update)
		r.Delete("/{workspaceId}", h.Archive)
		r.Get("/{workspaceId}/members", h.ListMembers)
		r.Post("/{workspaceId}/members/invite", h.InviteMember)
		r.Patch("/{workspaceId}/members/{memberId}", h.UpdateMember)
		r.Delete("/{workspaceId}/members/{memberId}", h.RemoveMember)
	})
	r.Route("/workspace-invitations", func(r chi.Router) {
		r.Get("/", h.ListInvitations)
		r.Post("/{invitationId}/accept", h.AcceptInvitation)
		r.Post("/{invitationId}/decline", h.DeclineInvitation)
	})
}

func (h *Handler) RegisterInternalRoutes(r chi.Router) {
	r.Route("/internal/workspaces", func(r chi.Router) {
		r.Post("/access-check", h.InternalAccessCheck)
		r.Post("/list-for-user", h.InternalListForUser)
		r.Post("/list-members", h.InternalListMembers)
		r.Post("/batch", h.InternalBatchInfo)
	})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createWorkspaceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	summary, err := h.svc.Create(r.Context(), CreateInput{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, newWorkspaceResponse(*summary))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	workspaces, err := h.svc.List(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	items := make([]workspaceResponse, 0, len(workspaces))
	for _, item := range workspaces {
		items = append(items, newWorkspaceResponse(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspaces": items})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := parseUUIDParam(w, r, "workspaceId", "invalid workspace id")
	if !ok {
		return
	}
	summary, err := h.svc.Get(r.Context(), workspaceID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newWorkspaceResponse(*summary))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := parseUUIDParam(w, r, "workspaceId", "invalid workspace id")
	if !ok {
		return
	}
	var req updateWorkspaceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	summary, err := h.svc.Update(r.Context(), workspaceID, UpdateInput{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newWorkspaceResponse(*summary))
}

func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := parseUUIDParam(w, r, "workspaceId", "invalid workspace id")
	if !ok {
		return
	}
	summary, err := h.svc.Archive(r.Context(), workspaceID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newWorkspaceResponse(*summary))
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := parseUUIDParam(w, r, "workspaceId", "invalid workspace id")
	if !ok {
		return
	}
	members, err := h.svc.ListMembers(r.Context(), workspaceID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	out := make([]memberResponse, 0, len(members))
	for _, member := range members {
		out = append(out, newMemberResponse(member))
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": out})
}

func (h *Handler) InviteMember(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := parseUUIDParam(w, r, "workspaceId", "invalid workspace id")
	if !ok {
		return
	}
	var req inviteMemberRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	invitation, err := h.svc.InviteMember(r.Context(), workspaceID, InviteInput{
		Email: req.Email,
		Role:  Role(req.Role),
	})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, newInvitationResponse(*invitation))
}

func (h *Handler) ListInvitations(w http.ResponseWriter, r *http.Request) {
	invitations, err := h.svc.ListInvitations(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	out := make([]invitationResponse, 0, len(invitations))
	for _, invitation := range invitations {
		out = append(out, newInvitationResponse(invitation))
	}
	writeJSON(w, http.StatusOK, map[string]any{"invitations": out})
}

func (h *Handler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	invitationID, ok := parseUUIDParam(w, r, "invitationId", "invalid invitation id")
	if !ok {
		return
	}
	workspace, err := h.svc.AcceptInvitation(r.Context(), invitationID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newWorkspaceResponse(*workspace))
}

func (h *Handler) DeclineInvitation(w http.ResponseWriter, r *http.Request) {
	invitationID, ok := parseUUIDParam(w, r, "invitationId", "invalid invitation id")
	if !ok {
		return
	}
	if err := h.svc.DeclineInvitation(r.Context(), invitationID); err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := parseUUIDParam(w, r, "workspaceId", "invalid workspace id")
	if !ok {
		return
	}
	memberID, ok := parseUUIDParam(w, r, "memberId", "invalid member id")
	if !ok {
		return
	}
	var req updateMemberRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	member, err := h.svc.UpdateMember(r.Context(), workspaceID, memberID, UpdateMemberInput{
		Role: Role(req.Role),
	})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newMemberResponse(*member))
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := parseUUIDParam(w, r, "workspaceId", "invalid workspace id")
	if !ok {
		return
	}
	memberID, ok := parseUUIDParam(w, r, "memberId", "invalid member id")
	if !ok {
		return
	}
	if err := h.svc.RemoveMember(r.Context(), workspaceID, memberID); err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) InternalAccessCheck(w http.ResponseWriter, r *http.Request) {
	var req internalAccessCheckRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, err := uuid.Parse(strings.TrimSpace(req.UserID))
	if err != nil {
		writeError(w, http.StatusBadRequest, "userId must be a valid uuid")
		return
	}
	workspaceID, err := uuid.Parse(strings.TrimSpace(req.WorkspaceID))
	if err != nil {
		writeError(w, http.StatusBadRequest, "workspaceId must be a valid uuid")
		return
	}
	access, err := h.svc.AccessCheck(r.Context(), userID, workspaceID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, internalAccessCheckResponse{
		HasAccess:         access.HasAccess,
		Role:              string(access.Role),
		Status:            string(access.Status),
		WorkspaceArchived: access.WorkspaceArchived,
	})
}

func (h *Handler) InternalListForUser(w http.ResponseWriter, r *http.Request) {
	var req internalListForUserRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, err := uuid.Parse(strings.TrimSpace(req.UserID))
	if err != nil {
		writeError(w, http.StatusBadRequest, "userId must be a valid uuid")
		return
	}
	rows, err := h.svc.ListForUserInternal(r.Context(), userID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	items := make([]internalWorkspaceRoleResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, internalWorkspaceRoleResponse{ID: row.ID.String(), Role: string(row.Role)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspaces": items})
}

func (h *Handler) InternalListMembers(w http.ResponseWriter, r *http.Request) {
	var req internalListMembersRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	workspaceID, err := uuid.Parse(strings.TrimSpace(req.WorkspaceID))
	if err != nil {
		writeError(w, http.StatusBadRequest, "workspaceId must be a valid uuid")
		return
	}
	members, err := h.svc.ListMembersInternal(r.Context(), workspaceID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	items := make([]internalWorkspaceMemberResponse, 0, len(members))
	for _, member := range members {
		items = append(items, internalWorkspaceMemberResponse{
			ID:          member.ID.String(),
			WorkspaceID: member.WorkspaceID.String(),
			UserID:      member.UserID.String(),
			Role:        string(member.Role),
			Status:      string(member.Status),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": items})
}

func (h *Handler) InternalBatchInfo(w http.ResponseWriter, r *http.Request) {
	var req internalBatchRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	ids := make([]uuid.UUID, 0, len(req.WorkspaceIDs))
	for _, raw := range req.WorkspaceIDs {
		id, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil {
			writeError(w, http.StatusBadRequest, "workspaceIds must be valid uuids")
			return
		}
		ids = append(ids, id)
	}
	infos, err := h.svc.BatchInfo(r.Context(), ids)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	items := make([]internalWorkspaceInfoResponse, 0, len(infos))
	for _, info := range infos {
		items = append(items, internalWorkspaceInfoResponse{
			ID:       info.ID.String(),
			Name:     info.Name,
			Slug:     info.Slug,
			Archived: info.Archived,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspaces": items})
}

func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	var invalid *apperrs.InvalidInputError
	switch {
	case errors.As(err, &invalid):
		writeError(w, http.StatusBadRequest, invalid.Error())
	case errors.Is(err, ErrAlreadyMember):
		writeError(w, http.StatusConflict, "user is already a workspace member")
	case errors.Is(err, ErrConflict):
		writeError(w, http.StatusConflict, "workspace conflict")
	case errors.Is(err, ErrLastOwner):
		writeError(w, http.StatusBadRequest, "workspace must keep at least one owner")
	case errors.Is(err, ErrInvalidInvitee):
		writeError(w, http.StatusForbidden, "invitation does not belong to current user")
	case errors.Is(err, ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, domainerrs.ErrNotFound):
		writeError(w, http.StatusNotFound, "workspace resource not found")
	default:
		h.log.Error("unhandled workspace service error", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

type createWorkspaceRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type updateWorkspaceRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type inviteMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type updateMemberRequest struct {
	Role string `json:"role"`
}

type internalAccessCheckRequest struct {
	UserID      string `json:"userId"`
	WorkspaceID string `json:"workspaceId"`
}

type internalListForUserRequest struct {
	UserID string `json:"userId"`
}

type internalListMembersRequest struct {
	WorkspaceID string `json:"workspaceId"`
}

type internalBatchRequest struct {
	WorkspaceIDs []string `json:"workspaceIds"`
}

type workspaceResponse struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Slug            string     `json:"slug"`
	Description     *string    `json:"description,omitempty"`
	CurrentUserRole string     `json:"currentUserRole"`
	MemberCount     int        `json:"memberCount"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	ArchivedAt      *time.Time `json:"archivedAt,omitempty"`
}

type memberResponse struct {
	ID              string     `json:"id"`
	WorkspaceID     string     `json:"workspaceId"`
	UserID          string     `json:"userId"`
	Email           *string    `json:"email,omitempty"`
	DisplayName     *string    `json:"displayName,omitempty"`
	Role            string     `json:"role"`
	Status          string     `json:"status"`
	InvitedByUserID *string    `json:"invitedByUserId,omitempty"`
	InvitedAt       *time.Time `json:"invitedAt,omitempty"`
	JoinedAt        *time.Time `json:"joinedAt,omitempty"`
	RemovedAt       *time.Time `json:"removedAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type invitationResponse struct {
	ID              string     `json:"id"`
	WorkspaceID     string     `json:"workspaceId"`
	WorkspaceName   string     `json:"workspaceName"`
	Email           string     `json:"email"`
	InvitedUserID   *string    `json:"invitedUserId,omitempty"`
	Role            string     `json:"role"`
	Status          string     `json:"status"`
	InvitedByUserID string     `json:"invitedByUserId"`
	ExpiresAt       *time.Time `json:"expiresAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type internalAccessCheckResponse struct {
	HasAccess         bool   `json:"hasAccess"`
	Role              string `json:"role,omitempty"`
	Status            string `json:"status,omitempty"`
	WorkspaceArchived bool   `json:"workspaceArchived"`
}

type internalWorkspaceRoleResponse struct {
	ID   string `json:"id"`
	Role string `json:"role"`
}

type internalWorkspaceMemberResponse struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	UserID      string `json:"userId"`
	Role        string `json:"role"`
	Status      string `json:"status"`
}

type internalWorkspaceInfoResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Archived bool   `json:"archived"`
}

func newWorkspaceResponse(summary WorkspaceSummary) workspaceResponse {
	workspace := summary.Workspace
	return workspaceResponse{
		ID:              workspace.ID.String(),
		Name:            workspace.Name,
		Slug:            workspace.Slug,
		Description:     workspace.Description,
		CurrentUserRole: string(summary.CurrentUserRole),
		MemberCount:     summary.MemberCount,
		CreatedAt:       workspace.CreatedAt,
		UpdatedAt:       workspace.UpdatedAt,
		ArchivedAt:      workspace.ArchivedAt,
	}
}

func newMemberResponse(info WorkspaceMemberInfo) memberResponse {
	member := info.Member
	return memberResponse{
		ID:              member.ID.String(),
		WorkspaceID:     member.WorkspaceID.String(),
		UserID:          member.UserID.String(),
		Email:           info.Email,
		DisplayName:     info.DisplayName,
		Role:            string(member.Role),
		Status:          string(member.Status),
		InvitedByUserID: uuidPtrString(member.InvitedByUserID),
		InvitedAt:       member.InvitedAt,
		JoinedAt:        member.JoinedAt,
		RemovedAt:       member.RemovedAt,
		CreatedAt:       member.CreatedAt,
		UpdatedAt:       member.UpdatedAt,
	}
}

func newInvitationResponse(invitation WorkspaceInvitation) invitationResponse {
	return invitationResponse{
		ID:              invitation.ID.String(),
		WorkspaceID:     invitation.WorkspaceID.String(),
		WorkspaceName:   invitation.WorkspaceName,
		Email:           invitation.Email,
		InvitedUserID:   uuidPtrString(invitation.InvitedUserID),
		Role:            string(invitation.Role),
		Status:          string(invitation.Status),
		InvitedByUserID: invitation.InvitedByUserID.String(),
		ExpiresAt:       invitation.ExpiresAt,
		CreatedAt:       invitation.CreatedAt,
		UpdatedAt:       invitation.UpdatedAt,
	}
}

func uuidPtrString(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	value := id.String()
	return &value
}

func parseUUIDParam(w http.ResponseWriter, r *http.Request, name, message string) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, name)))
	if err != nil {
		writeError(w, http.StatusBadRequest, message)
		return uuid.Nil, false
	}
	return id, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

type errorBody struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorBody{Error: message})
}
