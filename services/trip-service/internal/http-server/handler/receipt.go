package handler

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/request"
	tripsecurity "github.com/KovalenkoDima236961/Travel_Ai_App/internal/security"
)

func (h *Handler) UploadReceipt(w http.ResponseWriter, r *http.Request) {
	tripID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	user, _ := auth.UserFromContext(r.Context())
	if !h.receiptUploadLimiter.Allow(user.ID.String() + ":" + requestClientKey(r)) {
		tripsecurity.ReceiptUploadRejected.WithLabelValues("rate_limited").Inc()
		h.auditSecurity("receipt_upload", "trip", tripID.String(), "rate_limited")
		writeRateLimited(w)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, h.svc.ReceiptMaxUploadBytes()+(1<<20))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()
	var expenseID *uuid.UUID
	if raw := strings.TrimSpace(r.FormValue("expenseId")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid expenseId")
			return
		}
		expenseID = &parsed
	}
	runOCR := true
	if raw := strings.TrimSpace(r.FormValue("runOcr")); raw != "" {
		parsed, ok := parseBoolValue(raw)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid runOcr")
			return
		}
		runOCR = parsed
	}
	receipt, err := h.svc.UploadReceipt(r.Context(), tripID, appdto.UploadReceiptInput{
		OriginalFilename: header.Filename,
		ContentType:      header.Header.Get("Content-Type"),
		SizeBytes:        header.Size,
		ExpenseID:        expenseID,
		RunOCR:           runOCR,
		File:             file,
	})
	if err != nil {
		tripsecurity.ReceiptUploadRejected.WithLabelValues("validation_or_access").Inc()
		h.auditSecurity("receipt_upload", "trip", tripID.String(), "denied")
		h.writeServiceError(w, err)
		return
	}
	h.auditSecurity("receipt_upload", "trip", tripID.String(), "success")
	writeJSON(w, http.StatusCreated, receipt)
}

func (h *Handler) ListReceipts(w http.ResponseWriter, r *http.Request) {
	tripID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	filters, ok := parseReceiptFilters(w, r)
	if !ok {
		return
	}
	receipts, err := h.svc.ListReceipts(r.Context(), tripID, filters)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, receipts)
}

func (h *Handler) GetReceipt(w http.ResponseWriter, r *http.Request) {
	tripID, receiptID, ok := h.parseReceiptIDs(w, r)
	if !ok {
		return
	}
	receipt, err := h.svc.GetReceipt(r.Context(), tripID, receiptID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, receipt)
}

func (h *Handler) GetReceiptFile(w http.ResponseWriter, r *http.Request) {
	tripID, receiptID, ok := h.parseReceiptIDs(w, r)
	if !ok {
		return
	}
	file, err := h.svc.OpenReceiptFile(r.Context(), tripID, receiptID)
	if err != nil {
		tripsecurity.ReceiptDownloadDenied.WithLabelValues("access_or_missing").Inc()
		h.auditSecurity("receipt_download", "receipt", receiptID.String(), "denied")
		h.writeServiceError(w, err)
		return
	}
	defer file.Reader.Close()
	h.auditSecurity("receipt_download", "receipt", receiptID.String(), "success")
	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(file.SizeBytes, 10))
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": file.Filename}))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	if _, err := io.Copy(w, file.Reader); err != nil {
		h.log.Warn("stream receipt file failed")
	}
}

func (h *Handler) ExtractReceipt(w http.ResponseWriter, r *http.Request) {
	tripID, receiptID, ok := h.parseReceiptIDs(w, r)
	if !ok {
		return
	}
	var req request.ExtractReceipt
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input, err := req.ToInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	receipt, err := h.svc.ExtractReceipt(r.Context(), tripID, receiptID, input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, receipt)
}

func (h *Handler) CreateExpenseFromReceipt(w http.ResponseWriter, r *http.Request) {
	tripID, receiptID, ok := h.parseReceiptIDs(w, r)
	if !ok {
		return
	}
	var req request.CreateTripExpense
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input, err := req.ToInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	expense, err := h.svc.CreateExpenseFromReceipt(r.Context(), tripID, receiptID, input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, expense)
}

func (h *Handler) AttachReceiptToExpense(w http.ResponseWriter, r *http.Request) {
	tripID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	expenseID, ok := parseUUIDParam(w, r, "expenseId", "invalid expense id")
	if !ok {
		return
	}
	var req request.AttachReceipt
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input, err := req.ToInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	receipt, err := h.svc.AttachReceiptToExpense(r.Context(), tripID, expenseID, input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, receipt)
}

func (h *Handler) DeleteReceipt(w http.ResponseWriter, r *http.Request) {
	tripID, receiptID, ok := h.parseReceiptIDs(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteReceipt(r.Context(), tripID, receiptID); err != nil {
		h.auditSecurity("receipt_delete", "receipt", receiptID.String(), "denied")
		h.writeServiceError(w, err)
		return
	}
	h.auditSecurity("receipt_delete", "receipt", receiptID.String(), "success")
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) parseReceiptIDs(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	tripID, ok := h.parseID(w, r)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	receiptID, ok := parseUUIDParam(w, r, "receiptId", "invalid receipt id")
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	return tripID, receiptID, true
}

func parseReceiptFilters(w http.ResponseWriter, r *http.Request) (appdto.ListReceiptsInput, bool) {
	var filters appdto.ListReceiptsInput
	query := r.URL.Query()
	if raw := strings.TrimSpace(query.Get("expenseId")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid expenseId")
			return filters, false
		}
		filters.ExpenseID = &parsed
	}
	if raw := strings.TrimSpace(query.Get("status")); raw != "" {
		status := entity.ReceiptStatus(raw)
		switch status {
		case entity.ReceiptStatusUploaded,
			entity.ReceiptStatusProcessing,
			entity.ReceiptStatusExtracted,
			entity.ReceiptStatusExtractionFailed,
			entity.ReceiptStatusAttached:
			filters.Status = &status
		default:
			writeError(w, http.StatusBadRequest, "invalid receipt status")
			return filters, false
		}
	}
	unlinkedOnly, ok := parseBoolQuery(w, r, "unlinkedOnly")
	if !ok {
		return filters, false
	}
	filters.UnlinkedOnly = unlinkedOnly
	filters.Limit, ok = parseQueryInt(w, r, "limit")
	if !ok {
		return filters, false
	}
	filters.Offset, ok = parseQueryInt(w, r, "offset")
	if !ok {
		return filters, false
	}
	return filters, true
}

func parseBoolValue(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes":
		return true, true
	case "false", "0", "no":
		return false, true
	default:
		return false, false
	}
}
