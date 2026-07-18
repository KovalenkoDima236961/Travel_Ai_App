package handler

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/dataexport"
)

func (h *Handler) CreateTripArchiveExport(w http.ResponseWriter, r *http.Request) {
	tripID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var input service.TripArchiveExportInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil && err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	job, err := h.svc.CreateTripArchiveExport(r.Context(), tripID, input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, tripExportResponse(tripID, job))
}

func (h *Handler) GetTripExport(w http.ResponseWriter, r *http.Request) {
	tripID, exportID, ok := parseTripExportIDs(w, r)
	if !ok {
		return
	}
	job, err := h.svc.GetTripExport(r.Context(), tripID, exportID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tripExportResponse(tripID, job))
}

func (h *Handler) DownloadTripExport(w http.ResponseWriter, r *http.Request) {
	tripID, exportID, ok := parseTripExportIDs(w, r)
	if !ok {
		return
	}
	file, err := h.svc.OpenTripExport(r.Context(), tripID, exportID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	defer file.Reader.Close()
	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": file.Filename}))
	w.Header().Set("Cache-Control", "private, no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if file.SizeBytes > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(file.SizeBytes, 10))
	}
	if _, err := io.Copy(w, file.Reader); err != nil {
		h.log.Warn("stream trip export failed")
	}
}

func (h *Handler) ExportExpensesCSV(w http.ResponseWriter, r *http.Request) {
	h.writeTripCSV(w, r, "expenses")
}
func (h *Handler) ExportSettlementsCSV(w http.ResponseWriter, r *http.Request) {
	h.writeTripCSV(w, r, "settlements")
}
func (h *Handler) ExportBudgetCSV(w http.ResponseWriter, r *http.Request) {
	h.writeTripCSV(w, r, "budget")
}
func (h *Handler) ExportReceiptMetadataCSV(w http.ResponseWriter, r *http.Request) {
	h.writeTripCSV(w, r, "receipt-metadata")
}

func (h *Handler) writeTripCSV(w http.ResponseWriter, r *http.Request, kind string) {
	tripID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	contents, filename, err := h.svc.ExportTripCSV(r.Context(), tripID, kind)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Cache-Control", "private, no-store, max-age=0")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(contents)
}

func parseTripExportIDs(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	tripID, ok := parseUUIDParam(w, r, "id", "invalid trip id")
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	exportID, err := uuid.Parse(chi.URLParam(r, "exportId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid export id")
		return uuid.Nil, uuid.Nil, false
	}
	return tripID, exportID, true
}

func tripExportResponse(tripID uuid.UUID, job *dataexport.Job) map[string]any {
	result := map[string]any{"exportId": job.ID.String(), "status": job.Status, "createdAt": job.CreatedAt}
	if job.FileName != nil {
		result["fileName"] = *job.FileName
	}
	if job.SizeBytes != nil {
		result["sizeBytes"] = *job.SizeBytes
	}
	if job.ChecksumSHA256 != nil {
		result["checksumSha256"] = *job.ChecksumSHA256
	}
	if job.ExpiresAt != nil {
		result["expiresAt"] = *job.ExpiresAt
	}
	if job.ErrorCode != nil {
		result["errorCode"] = *job.ErrorCode
	}
	if job.ErrorMessageSafe != nil {
		result["errorMessageSafe"] = *job.ErrorMessageSafe
	}
	if job.Status == dataexport.StatusCompleted {
		result["downloadUrl"] = "/trips/" + tripID.String() + "/export/" + job.ID.String() + "/download"
	}
	return result
}
