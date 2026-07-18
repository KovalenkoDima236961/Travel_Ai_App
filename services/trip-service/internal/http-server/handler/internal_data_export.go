package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
)

type accountTripPackageRequest struct {
	UserID               string `json:"userId"`
	IncludeWorkspaceData bool   `json:"includeWorkspaceData"`
	IncludeReceiptFiles  bool   `json:"includeReceiptFiles"`
}

// BuildAccountTripPackage runs only in the internal service-token router
// group. It writes no durable file: User Service stores the result in the
// already authenticated account-export job.
func (h *Handler) BuildAccountTripPackage(w http.ResponseWriter, r *http.Request) {
	var req accountTripPackageRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	contents, err := h.svc.BuildAccountTripPackage(r.Context(), userID, req.IncludeWorkspaceData, req.IncludeReceiptFiles)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(contents)
}
