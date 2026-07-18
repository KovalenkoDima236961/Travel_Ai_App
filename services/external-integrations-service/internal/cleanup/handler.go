// Package cleanup owns retention of persisted integration metadata. It does
// not inspect provider payloads or tokens and leaves process-local caches to
// their TTL cache implementation.
package cleanup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/storage/postgres"
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
type spec struct {
	table, where, timestamp string
	cutoff                  any
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
	started := time.Now()
	out := result{TaskName: task, DryRun: req.DryRun}
	if task == "provider_cache" {
		out.Warnings = []string{"provider caches are in-memory in v1 and evict at their own TTL"}
		out.DurationMS = time.Since(started).Milliseconds()
		writeJSON(w, http.StatusOK, out)
		return
	}
	s, ok := h.taskSpec(task, started.UTC())
	if !ok {
		writeError(w, http.StatusNotFound, "cleanup_task_not_found")
		return
	}
	if req.DryRun {
		count, err := h.eligible(r.Context(), s, req.BatchSize*req.MaxBatches)
		if err != nil {
			h.fail(w, task, err)
			return
		}
		out.ScannedCount = count
	} else {
		for i := 0; i < req.MaxBatches; i++ {
			deleted, err := h.deleteBatch(r.Context(), s, req.BatchSize)
			if err != nil {
				h.fail(w, task, err)
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
	h.log.Info("integration_cleanup", zap.String("task", task), zap.Bool("dryRun", req.DryRun), zap.Int64("scannedCount", out.ScannedCount), zap.Int64("deletedCount", out.DeletedCount), zap.Int64("durationMs", out.DurationMS))
	writeJSON(w, http.StatusOK, out)
}
func (h *Handler) taskSpec(task string, now time.Time) (spec, bool) {
	switch task {
	case "oauth_states":
		return spec{"calendar_oauth_states", "expires_at < $1", "expires_at", now.AddDate(0, 0, -h.cfg.OAuthStatesDays)}, true
	case "provider_quota_counters":
		return spec{"provider_daily_usage", "usage_date < $1", "usage_date", now.AddDate(0, 0, -h.cfg.QuotaCountersDays).Format("2006-01-02")}, true
	default:
		return spec{}, false
	}
}
func (h *Handler) eligible(ctx context.Context, s spec, limit int) (int64, error) {
	var count int64
	q := fmt.Sprintf("SELECT COUNT(*) FROM (SELECT id FROM %s WHERE %s ORDER BY %s, id LIMIT $2) candidates", s.table, s.where, s.timestamp)
	err := h.db.QueryRow(ctx, q, s.cutoff, limit).Scan(&count)
	return count, err
}
func (h *Handler) deleteBatch(ctx context.Context, s spec, batch int) (int64, error) {
	q := fmt.Sprintf("WITH ids AS (SELECT id FROM %s WHERE %s ORDER BY %s, id LIMIT $2) DELETE FROM %s WHERE id IN (SELECT id FROM ids)", s.table, s.where, s.timestamp, s.table)
	tag, err := h.db.Exec(ctx, q, s.cutoff, batch)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
func (h *Handler) fail(w http.ResponseWriter, task string, err error) {
	h.log.Warn("integration cleanup failed", zap.String("task", task), zap.Error(err))
	writeError(w, http.StatusInternalServerError, "cleanup_internal_error")
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": strings.TrimSpace(code)})
}
