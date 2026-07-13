package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/request"
)

func (h *Handler) ListReminders(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	filters, err := parseReminderFilters(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	result, err := h.svc.ListTripReminders(r.Context(), id, filters)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) ListAssignedReminders(w http.ResponseWriter, r *http.Request) {
	filters, err := parseReminderFilters(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	result, err := h.svc.ListAssignedTripReminders(r.Context(), filters)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GenerateReminders(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.GenerateReminders
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.svc.GenerateTripReminders(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) CreateReminder(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.CreateReminder
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input, err := req.ToInput()
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	reminder, err := h.svc.CreateTripReminder(r.Context(), id, input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, reminder)
}

func (h *Handler) UpdateReminder(w http.ResponseWriter, r *http.Request) {
	id, reminderID, ok := h.parseReminderIDs(w, r)
	if !ok {
		return
	}
	var req request.UpdateReminder
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input, err := req.ToInput()
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	reminder, err := h.svc.UpdateTripReminder(r.Context(), id, reminderID, input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reminder)
}

func (h *Handler) CompleteReminder(w http.ResponseWriter, r *http.Request) {
	h.setReminderDoneState(w, r, true)
}

func (h *Handler) ReopenReminder(w http.ResponseWriter, r *http.Request) {
	h.setReminderDoneState(w, r, false)
}

func (h *Handler) DisableReminder(w http.ResponseWriter, r *http.Request) {
	id, reminderID, ok := h.parseReminderIDs(w, r)
	if !ok {
		return
	}
	reminder, err := h.svc.DisableTripReminder(r.Context(), id, reminderID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reminder)
}

func (h *Handler) EnableReminder(w http.ResponseWriter, r *http.Request) {
	id, reminderID, ok := h.parseReminderIDs(w, r)
	if !ok {
		return
	}
	reminder, err := h.svc.EnableTripReminder(r.Context(), id, reminderID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reminder)
}

func (h *Handler) DeleteReminder(w http.ResponseWriter, r *http.Request) {
	id, reminderID, ok := h.parseReminderIDs(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteTripReminder(r.Context(), id, reminderID); err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) ProcessDueReminders(w http.ResponseWriter, r *http.Request) {
	var req request.ProcessDueReminders
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.svc.ProcessDueTripReminders(r.Context(), req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) setReminderDoneState(w http.ResponseWriter, r *http.Request, done bool) {
	id, reminderID, ok := h.parseReminderIDs(w, r)
	if !ok {
		return
	}
	var (
		reminder appdto.TripReminderDTO
		err      error
	)
	if done {
		reminder, err = h.svc.CompleteTripReminder(r.Context(), id, reminderID)
	} else {
		reminder, err = h.svc.ReopenTripReminder(r.Context(), id, reminderID)
	}
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reminder)
}

func (h *Handler) parseReminderIDs(w http.ResponseWriter, r *http.Request) (tripID, reminderID uuid.UUID, ok bool) {
	tripID, ok = h.parseID(w, r)
	if !ok {
		return uuid.UUID{}, uuid.UUID{}, false
	}
	reminderID, ok = parseUUIDParam(w, r, "reminderId", "invalid reminder id")
	if !ok {
		return uuid.UUID{}, uuid.UUID{}, false
	}
	return tripID, reminderID, true
}

func parseReminderFilters(r *http.Request) (appdto.ReminderListFilters, error) {
	q := r.URL.Query()
	var filters appdto.ReminderListFilters
	if raw := strings.TrimSpace(q.Get("status")); raw != "" {
		status := entity.ReminderStatus(raw)
		if !knownReminderStatus(status) {
			return filters, apperrs.NewInvalidInput("status is invalid")
		}
		filters.Status = &status
	}
	if raw := strings.TrimSpace(q.Get("category")); raw != "" {
		category := entity.ReminderCategory(raw)
		if !knownReminderCategory(category) {
			return filters, apperrs.NewInvalidInput("category is invalid")
		}
		filters.Category = &category
	}
	filters.AssignedToMe = parseReminderBoolQuery(q.Get("assignedToMe"))
	filters.UpcomingOnly = parseReminderBoolQuery(q.Get("upcomingOnly"))
	filters.HighPriorityOnly = parseReminderBoolQuery(q.Get("highPriority"))
	fromDate, err := parseReminderQueryDate(q.Get("fromDate"), "fromDate")
	if err != nil {
		return filters, err
	}
	toDate, err := parseReminderQueryDate(q.Get("toDate"), "toDate")
	if err != nil {
		return filters, err
	}
	filters.FromDate = fromDate
	filters.ToDate = toDate
	return filters, nil
}

func parseReminderBoolQuery(raw string) bool {
	raw = strings.TrimSpace(strings.ToLower(raw))
	return raw == "true" || raw == "1" || raw == "yes"
}

func parseReminderQueryDate(raw, name string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, apperrs.NewInvalidInput("%s must be in YYYY-MM-DD format", name)
	}
	return &parsed, nil
}

func knownReminderStatus(status entity.ReminderStatus) bool {
	switch status {
	case entity.ReminderStatusPending,
		entity.ReminderStatusSent,
		entity.ReminderStatusCompleted,
		entity.ReminderStatusDisabled,
		entity.ReminderStatusCancelled,
		entity.ReminderStatusFailed:
		return true
	default:
		return false
	}
}

func knownReminderCategory(category entity.ReminderCategory) bool {
	switch category {
	case entity.ReminderCategoryDocuments,
		entity.ReminderCategoryPacking,
		entity.ReminderCategoryTransport,
		entity.ReminderCategoryAccommodation,
		entity.ReminderCategoryWeather,
		entity.ReminderCategoryActivities,
		entity.ReminderCategoryGroup,
		entity.ReminderCategoryChecklist,
		entity.ReminderCategoryBeforeDeparture,
		entity.ReminderCategoryRoute,
		entity.ReminderCategorySafety,
		entity.ReminderCategoryOther:
		return true
	default:
		return false
	}
}
