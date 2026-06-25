package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/http-server/dto/request"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
)

// internalService is the create port used by trusted internal callers.
type internalService interface {
	CreateBatch(ctx context.Context, inputs []notifications.CreateInput) (int, error)
}

// InternalHandler serves service-to-service endpoints. Its routes must be
// mounted behind the internal service-token middleware — never exposed to
// browsers and never requiring a user JWT.
type InternalHandler struct {
	svc internalService
	log *zap.Logger
}

// NewInternal constructs the internal notification HTTP handler.
func NewInternal(svc internalService, log *zap.Logger) *InternalHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &InternalHandler{svc: svc, log: log}
}

// RegisterRoutes mounts the internal routes. The caller wraps these in the
// internal service-token middleware.
func (h *InternalHandler) RegisterRoutes(r chi.Router) {
	r.Route("/internal/notifications", func(r chi.Router) {
		r.Post("/batch", h.CreateBatch)
	})
}

// CreateBatch handles POST /internal/notifications/batch. It trusts the caller
// to provide recipient user ids; it skips self-notifications and returns the
// number of notifications actually created.
func (h *InternalHandler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	var req request.CreateNotificationsBatch
	if !decodeJSON(w, r, &req) {
		return
	}

	inputs, err := req.ToInputs()
	if err != nil {
		writeServiceError(w, h.log, err)
		return
	}

	created, err := h.svc.CreateBatch(r.Context(), inputs)
	if err != nil {
		writeServiceError(w, h.log, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]int{"created": created})
}
