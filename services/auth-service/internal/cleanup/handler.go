// Package cleanup exposes bounded, internal-only lifecycle actions for Auth
// Service. Worker Service orchestrates these actions but never reaches into
// the Auth database directly.
package cleanup

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/pkg/storage/postgres"
)

type Handler struct {
	db  *postgres.DB
	cfg config.CleanupConfig
	log *zap.Logger
}

func New(db *postgres.DB, cfg config.CleanupConfig, log *zap.Logger) *Handler {
	if log == nil {
		log = zap.NewNop()
	}
	return &Handler{db: db, cfg: cfg, log: log}
}

func (h *Handler) RegisterRoutes(r chi.Router) { r.Post("/internal/cleanup/{taskName}", h.run) }

type request struct {
	DryRun     bool `json:"dryRun"`
	BatchSize  int  `json:"batchSize"`
	MaxBatches int  `json:"maxBatches"`
}

type result struct {
	TaskName         string   `json:"taskName"`
	DryRun           bool     `json:"dryRun"`
	ScannedCount     int64    `json:"scannedCount"`
	DeletedCount     int64    `json:"deletedCount"`
	ArchivedCount    int64    `json:"archivedCount"`
	SkippedCount     int64    `json:"skippedCount"`
	ErrorCount       int64    `json:"errorCount"`
	FileDeletedCount int64    `json:"fileDeletedCount"`
	BytesFreed       int64    `json:"bytesFreed"`
	DurationMS       int64    `json:"durationMs"`
	Warnings         []string `json:"warnings,omitempty"`
}

func (h *Handler) run(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		writeError(w, http.StatusServiceUnavailable, "cleanup_disabled")
		return
	}
	var req request
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cleanup_invalid_scope")
		return
	}
	if req.BatchSize < 1 || req.BatchSize > 1000 || req.MaxBatches < 1 || req.MaxBatches > 100 {
		writeError(w, http.StatusBadRequest, "cleanup_invalid_scope")
		return
	}

	task := chi.URLParam(r, "taskName")
	days, column := 0, ""
	switch task {
	case "expired_refresh_tokens":
		days, column = h.cfg.ExpiredRefreshTokensDays, "expires_at"
	case "revoked_refresh_tokens":
		days, column = h.cfg.RevokedRefreshTokensDays, "revoked_at"
	default:
		writeError(w, http.StatusNotFound, "cleanup_task_not_found")
		return
	}

	started := time.Now()
	out := result{TaskName: task, DryRun: req.DryRun}
	cutoff := started.UTC().AddDate(0, 0, -days)
	if req.DryRun {
		count, err := h.eligible(r, column, cutoff, req.BatchSize*req.MaxBatches)
		if err != nil {
			h.log.Warn("auth cleanup dry-run failed", zap.String("task", task), zap.Error(err))
			writeError(w, http.StatusInternalServerError, "cleanup_internal_error")
			return
		}
		out.ScannedCount = count
	} else {
		for i := 0; i < req.MaxBatches; i++ {
			deleted, err := h.deleteBatch(r, column, cutoff, req.BatchSize)
			if err != nil {
				h.log.Warn("auth cleanup failed", zap.String("task", task), zap.Error(err))
				writeError(w, http.StatusInternalServerError, "cleanup_internal_error")
				return
			}
			out.ScannedCount += deleted
			out.DeletedCount += deleted
			if deleted < int64(req.BatchSize) {
				break
			}
		}
	}
	out.DurationMS = time.Since(started).Milliseconds()
	h.log.Info("auth_cleanup", zap.String("task", task), zap.Bool("dryRun", req.DryRun), zap.Int64("scannedCount", out.ScannedCount), zap.Int64("deletedCount", out.DeletedCount), zap.Int64("durationMs", out.DurationMS))
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) eligible(r *http.Request, column string, cutoff time.Time, limit int) (int64, error) {
	var count int64
	query := fmt.Sprintf("SELECT COUNT(*) FROM (SELECT id FROM refresh_tokens WHERE %s IS NOT NULL AND %s < $1 ORDER BY %s, id LIMIT $2) candidates", column, column, column)
	err := h.db.QueryRow(r.Context(), query, cutoff, limit).Scan(&count)
	return count, err
}

func (h *Handler) deleteBatch(r *http.Request, column string, cutoff time.Time, batch int) (int64, error) {
	query := fmt.Sprintf("WITH ids AS (SELECT id FROM refresh_tokens WHERE %s IS NOT NULL AND %s < $1 ORDER BY %s, id LIMIT $2) DELETE FROM refresh_tokens WHERE id IN (SELECT id FROM ids)", column, column, column)
	tag, err := h.db.Exec(r.Context(), query, cutoff, batch)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": strings.TrimSpace(code)})
}
