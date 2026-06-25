package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"go.uber.org/zap"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/editlocks"
)

type editLockRequest struct {
	Scope string `json:"scope"`
}

func (h *Handler) GetEditLock(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	scope, ok := h.editLockScopeFromQuery(w, r)
	if !ok {
		return
	}
	if _, err := h.svc.GetTripAccess(r.Context(), id); err != nil {
		h.writeServiceError(w, err)
		return
	}

	if !h.editLocksEnabled() {
		view := editlocks.NewUnlockedView(id, scope)
		view.Disabled = true
		writeJSON(w, http.StatusOK, view)
		return
	}

	lock, err := h.editLocks.Get(r.Context(), id, scope, user.ID)
	if err != nil {
		h.writeEditLockError(w, err)
		return
	}
	if lock == nil {
		writeJSON(w, http.StatusOK, editlocks.NewUnlockedView(id, scope))
		return
	}
	writeJSON(w, http.StatusOK, lock)
}

func (h *Handler) AcquireEditLock(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	scope, ok := h.editLockScopeFromBody(w, r)
	if !ok {
		return
	}
	access, err := h.svc.GetTripAccess(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if !access.CanEdit() {
		h.writeServiceError(w, apperrs.ErrForbidden)
		return
	}

	if !h.editLocksEnabled() {
		writeJSON(w, http.StatusOK, editlocks.AcquireEditLockResponse{
			Acquired: true,
			Disabled: true,
		})
		return
	}

	result, err := h.editLocks.AcquireOrRenew(r.Context(), editlocks.AcquireLockInput{
		TripID: id,
		Scope:  scope,
		UserID: user.ID,
		Role:   string(access.Level),
		TTL:    h.editLockCfg.TTL,
	})
	if err != nil {
		h.writeEditLockError(w, err)
		return
	}
	if result.BlockedByOther {
		writeJSON(w, http.StatusConflict, editlocks.AcquireEditLockResponse{
			Error:    "edit_lock_conflict",
			Message:  "Another user is already editing this itinerary.",
			Acquired: false,
			Reason:   "locked_by_other_user",
			Lock:     result.Lock,
		})
		return
	}

	writeJSON(w, http.StatusOK, editlocks.AcquireEditLockResponse{
		Acquired: result.Acquired,
		Renewed:  result.Renewed,
		Lock:     result.Lock,
	})
}

func (h *Handler) ReleaseEditLock(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	scope, ok := h.editLockScopeFromBody(w, r)
	if !ok {
		return
	}
	access, err := h.svc.GetTripAccess(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if !access.CanEdit() {
		h.writeServiceError(w, apperrs.ErrForbidden)
		return
	}

	if !h.editLocksEnabled() {
		writeJSON(w, http.StatusOK, editlocks.ReleaseEditLockResponse{Released: false})
		return
	}

	lock, err := h.editLocks.Get(r.Context(), id, scope, user.ID)
	if err != nil {
		h.writeEditLockError(w, err)
		return
	}
	if lock != nil && lock.Locked && !lock.LockedByCurrentUser {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error":   "not_lock_owner",
			"message": "You do not own the active edit lock.",
		})
		return
	}

	released, err := h.editLocks.Release(r.Context(), id, scope, user.ID)
	if err != nil {
		h.writeEditLockError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, editlocks.ReleaseEditLockResponse{Released: released})
}

func (h *Handler) editLockScopeFromQuery(w http.ResponseWriter, r *http.Request) (string, bool) {
	scope, err := editlocks.NormalizeScope(r.URL.Query().Get("scope"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid edit lock scope")
		return "", false
	}
	return scope, true
}

func (h *Handler) editLockScopeFromBody(w http.ResponseWriter, r *http.Request) (string, bool) {
	var req editLockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return "", false
	}
	scope, err := editlocks.NormalizeScope(req.Scope)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid edit lock scope")
		return "", false
	}
	return scope, true
}

func (h *Handler) editLocksEnabled() bool {
	return h.editLocks != nil && h.editLockCfg.Enabled
}

func (h *Handler) writeEditLockError(w http.ResponseWriter, err error) {
	if errors.Is(err, editlocks.ErrInvalidScope) {
		writeError(w, http.StatusBadRequest, "invalid edit lock scope")
		return
	}
	h.log.Error("edit lock error", zap.Error(err))
	writeError(w, http.StatusInternalServerError, "internal server error")
}
