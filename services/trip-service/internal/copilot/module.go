package copilot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/aiobservability"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	tripsecurity "github.com/KovalenkoDima236961/Travel_Ai_App/internal/security"
)

type Handler struct {
	svc     *service.Service
	cfg     Config
	ai      *AIClient
	tracer  *aiobservability.Service
	limiter *tripsecurity.RateLimiter
	log     *zap.Logger
}

func NewHandler(
	svc *service.Service,
	cfg Config,
	aiPlanningServiceURL string,
	tracer *aiobservability.Service,
	log *zap.Logger,
) (*Handler, error) {
	cfg = NormalizeConfig(cfg)
	if log == nil {
		log = zap.NewNop()
	}
	handler := &Handler{
		svc:     svc,
		cfg:     cfg,
		tracer:  tracer,
		limiter: tripsecurity.NewRateLimiter(cfg.RateLimitPerMinute, time.Minute),
		log:     log,
	}
	if cfg.Enabled && cfg.Mode == "ai" {
		client, err := NewAIClient(aiPlanningServiceURL, int(cfg.Timeout.Seconds()))
		if err != nil {
			return nil, err
		}
		handler.ai = client
	}
	return handler, nil
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/trips/{tripId}/copilot/chat", h.Chat)
}

func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svc == nil || !h.cfg.Enabled {
		writeError(w, http.StatusServiceUnavailable, "copilot_unavailable", "Copilot is unavailable")
		return
	}
	tripID, err := uuid.Parse(chi.URLParam(r, "tripId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_trip_id", "Trip id must be a UUID")
		return
	}
	var input ChatRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, int64(h.cfg.MaxMessageChars+8*1024)))
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid Copilot request")
		return
	}
	input.Message = strings.TrimSpace(input.Message)
	if input.Message == "" || len(input.Message) > h.cfg.MaxMessageChars {
		writeError(w, http.StatusBadRequest, "invalid_message", "Message must be between 1 and the configured maximum length")
		return
	}
	conversationID, err := normalizedConversationID(input.ConversationID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_conversation_id", "conversationId must be a UUID")
		return
	}
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required")
		return
	}
	if !h.limiter.Allow(user.ID.String() + ":" + tripID.String()) {
		writeError(w, http.StatusTooManyRequests, "copilot_rate_limited", "Please wait a moment before asking another question")
		return
	}

	intent := ClassifyIntent(input.Message)
	started := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.Timeout)
	defer cancel()
	safeContext, access, err := BuildSafeContext(ctx, h.svc, tripID, input.ClientContext)
	if err != nil {
		writeError(w, http.StatusNotFound, "trip_not_found", "Trip not found")
		return
	}
	role := access.Role()
	available := AvailableActions(tripID, access, input.ClientContext)
	language := h.svc.CopilotPreferredLanguage(ctx, r.Header.Get("Accept-Language"))
	trace := h.startTrace(ctx, tripID, user.ID, intent, input.Message, safeContext)

	result := fallbackResponse(intent, safeContext, available, language)
	status := "fallback"
	failClosed := false
	failureCode := ""
	if intent == IntentUnsafeMutationRequest {
		unsafeRequestsTotal.WithLabelValues(string(intent), role).Inc()
		status = "unsafe_refusal"
	} else if intent == IntentOutOfScope {
		status = "out_of_scope"
	} else if h.cfg.Mode == "ai" && h.ai != nil {
		if response, aiErr := h.ai.Respond(ctx, input.Message, language, intent, compactContext(safeContext, h.cfg.MaxContextChars), available, permissionSummary(access)); aiErr == nil {
			if validated, validationErr := validateAIResponse(response, available); validationErr == nil {
				result = validated
				status = "success"
			} else {
				validationFailuresTotal.WithLabelValues(string(intent), h.cfg.Mode).Inc()
				h.log.Warn("copilot response rejected", zap.String("trip_id", tripID.String()), zap.String("intent", string(intent)))
				failClosed = !h.cfg.FailOpen
				failureCode = "response_validation_failed"
				if !failClosed {
					fallbacksTotal.WithLabelValues(string(intent), h.cfg.Mode).Inc()
				}
			}
		} else {
			aiFailuresTotal.WithLabelValues(string(intent), h.cfg.Mode).Inc()
			h.log.Warn("copilot AI call failed", zap.String("trip_id", tripID.String()), zap.String("intent", string(intent)), zap.Error(aiErr))
			failClosed = !h.cfg.FailOpen
			failureCode = "ai_provider_unavailable"
			if !failClosed {
				fallbacksTotal.WithLabelValues(string(intent), h.cfg.Mode).Inc()
			}
		}
	} else {
		fallbacksTotal.WithLabelValues(string(intent), h.cfg.Mode).Inc()
	}
	if failClosed {
		status = "ai_unavailable"
		requestsTotal.WithLabelValues(string(intent), h.cfg.Mode, status, role).Inc()
		durationSeconds.WithLabelValues(string(intent), h.cfg.Mode, status, role).Observe(time.Since(started).Seconds())
		h.failTrace(ctx, trace, status, failureCode)
		writeError(w, http.StatusServiceUnavailable, "copilot_unavailable", "Copilot is temporarily unavailable")
		return
	}

	response := ChatResponse{
		ConversationID:  conversationID,
		MessageID:       uuid.NewString(),
		Answer:          result.Answer,
		Actions:         result.Actions,
		Sources:         responseSources(tripID, result.SourceTypes),
		Warnings:        result.Warnings,
		PermissionNotes: permissionNotes(access, language),
		Metadata: ResponseMetadata{
			Mode:            h.cfg.Mode,
			Intent:          intent,
			SafeContextUsed: contextSections(safeContext),
		},
	}
	if len(response.Actions) > 0 {
		actionSuggestionsTotal.WithLabelValues(string(intent), h.cfg.Mode, role).Add(float64(len(response.Actions)))
	}
	requestsTotal.WithLabelValues(string(intent), h.cfg.Mode, status, role).Inc()
	durationSeconds.WithLabelValues(string(intent), h.cfg.Mode, status, role).Observe(time.Since(started).Seconds())
	h.completeTrace(ctx, trace, intent, status, response)
	writeJSON(w, http.StatusOK, response)
}

func normalizedConversationID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return uuid.NewString(), nil
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return "", err
	}
	return parsed.String(), nil
}

func permissionSummary(access service.TripAccess) PermissionSummary {
	return PermissionSummary{
		Role:             access.Role(),
		CanEditItinerary: access.CanEdit(),
		CanEditRoute:     access.CanEdit(),
		CanManageShare:   access.CanManageShare(),
		CanUploadReceipt: access.Allows(tripsecurity.PermissionReceiptsUpload),
		CanComment:       access.Allows(tripsecurity.PermissionCommentsCreate),
		CanVote:          access.CanView(),
	}
}

func permissionNotes(access service.TripAccess, language string) []string {
	if access.CanEdit() {
		return []string{}
	}
	return []string{fallbackText(language, "permission_note")}
}

func responseSources(tripID uuid.UUID, sourceTypes []string) []Source {
	out := make([]Source, 0, len(sourceTypes))
	for _, sourceType := range sourceTypes {
		info, ok := sourceDefinition(sourceType)
		if !ok {
			continue
		}
		out = append(out, Source{Type: sourceType, Label: info.label, Href: actionHref(tripID, "source", info.tab, ClientContext{})})
	}
	if len(out) == 0 {
		info := sourceDefinitions["command_center"]
		out = append(out, Source{Type: "command_center", Label: info.label, Href: actionHref(tripID, "source", info.tab, ClientContext{})})
	}
	return out
}

func contextSections(context SafeContext) []string {
	sections := []string{"trip"}
	for _, entry := range []struct {
		name  string
		value map[string]any
	}{
		{"command_center", context.CommandCenter}, {"trip_health", context.Health}, {"budget_confidence", context.Budget},
		{"group_readiness", context.Group}, {"route_summary", context.Route}, {"itinerary_summary", context.Itinerary}, {"travel_day", context.TravelDay},
		{"checklist_summary", context.Checklist}, {"reminders_summary", context.Reminders}, {"expenses_summary", context.Expenses},
		{"approval_status", context.Approval}, {"policy_evaluation", context.Policy},
		{"generation_quality", context.Generation}, {"personalization", context.Personalization},
	} {
		if entry.value != nil {
			sections = append(sections, entry.name)
		}
	}
	return sections
}

func compactContext(value SafeContext, maxChars int) SafeContext {
	encoded, err := json.Marshal(value)
	if err == nil && len(encoded) <= maxChars {
		return value
	}
	if value.Health != nil {
		delete(value.Health, "topIssues")
		delete(value.Health, "topFixes")
	}
	if value.Budget != nil {
		delete(value.Budget, "issues")
		delete(value.Budget, "recommendations")
	}
	if value.Route != nil {
		delete(value.Route, "legs")
		delete(value.Route, "selectedLeg")
	}
	value.Unavailable = append(value.Unavailable, "copilot_context_truncated")
	return value
}

func (h *Handler) startTrace(ctx context.Context, tripID, userID uuid.UUID, intent Intent, message string, safeContext SafeContext) *aiobservability.TraceContext {
	if h.tracer == nil {
		return nil
	}
	digest := sha256.Sum256([]byte(message))
	input, _ := json.Marshal(map[string]any{
		"intent":          intent,
		"messageHash":     hex.EncodeToString(digest[:]),
		"contextSections": contextSections(safeContext),
	})
	trace, err := h.tracer.StartTrace(ctx, aiobservability.StartTraceInput{
		TripID:         &tripID,
		UserID:         &userID,
		GenerationType: "copilot_response",
		Source:         "trip_copilot",
		Provider:       h.cfg.Mode,
		AIMode:         h.cfg.Mode,
		PromptVersion:  "copilot_v1",
		InputSummary:   input,
	})
	if err != nil {
		h.log.Warn("copilot trace unavailable", zap.Error(err))
		return nil
	}
	return trace
}

func (h *Handler) completeTrace(ctx context.Context, trace *aiobservability.TraceContext, intent Intent, status string, response ChatResponse) {
	if trace == nil || !trace.Active || h.tracer == nil {
		return
	}
	sourceTypes := make([]string, 0, len(response.Sources))
	for _, source := range response.Sources {
		sourceTypes = append(sourceTypes, source.Type)
	}
	output, _ := json.Marshal(map[string]any{
		"intent":      intent,
		"status":      status,
		"actionCount": len(response.Actions),
		"sourceTypes": sourceTypes,
	})
	if err := h.tracer.CompleteTrace(ctx, trace.TraceID, aiobservability.CompleteTraceInput{
		Status:        aiobservability.StatusCompleted,
		QualityStatus: status,
		OutputSummary: output,
	}); err != nil {
		h.log.Warn("copilot trace completion failed", zap.Error(err))
	}
}

func (h *Handler) failTrace(ctx context.Context, trace *aiobservability.TraceContext, status, code string) {
	if trace == nil || !trace.Active || h.tracer == nil {
		return
	}
	if err := h.tracer.FailTrace(ctx, trace.TraceID, aiobservability.FailTraceInput{
		Status:        aiobservability.StatusFailed,
		QualityStatus: status,
		ErrorCode:     code,
	}); err != nil {
		h.log.Warn("copilot trace failure completion failed", zap.Error(err))
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": code, "message": message})
}
