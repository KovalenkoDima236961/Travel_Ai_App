package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/dataexport"
)

type accountExportRepository interface {
	CreateAccountExportJob(context.Context, dataexport.Job) (*dataexport.Job, error)
	GetAccountExportJob(context.Context, uuid.UUID, uuid.UUID) (*dataexport.Job, error)
	CompleteAccountExportJob(context.Context, uuid.UUID, string, string, string, int64, string, time.Time) (*dataexport.Job, error)
	FailAccountExportJob(context.Context, uuid.UUID, string, string) error
	ExpireAccountExportJob(context.Context, uuid.UUID) error
	ListExpiredAccountExportJobs(context.Context, time.Time) ([]dataexport.Job, error)
	CreateAccountCleanupRequest(context.Context, uuid.UUID, uuid.UUID, *string, bool) error
}

type accountTripPackageProvider interface {
	BuildAccountTripPackage(context.Context, uuid.UUID, bool, bool) ([]byte, error)
}

type AccountExportSections struct {
	Profile                 bool `json:"profile"`
	Preferences             bool `json:"preferences"`
	PersonalTrips           bool `json:"personalTrips"`
	TripRecaps              bool `json:"tripRecaps"`
	Templates               bool `json:"templates"`
	Expenses                bool `json:"expenses"`
	Settlements             bool `json:"settlements"`
	Checklists              bool `json:"checklists"`
	Reminders               bool `json:"reminders"`
	PersonalizationFeedback bool `json:"personalizationFeedback"`
	NotificationPreferences bool `json:"notificationPreferences"`
	Notifications           bool `json:"notifications"`
}

type AccountExportRequest struct {
	Sections             AccountExportSections `json:"sections"`
	IncludeReceiptFiles  bool                  `json:"includeReceiptFiles"`
	IncludeWorkspaceData bool                  `json:"includeWorkspaceData"`
}

type AccountCleanupRequest struct {
	Reason               string `json:"reason"`
	ExportRequestedFirst bool   `json:"exportRequestedFirst"`
}

type ExportDownload struct {
	Reader      io.ReadCloser
	Filename    string
	ContentType string
	SizeBytes   int64
}

func WithDataExports(storage *dataexport.LocalStorage, cfg dataexport.Config) Option {
	return func(s *Service) { s.dataExportStorage = storage; s.dataExportConfig = cfg }
}

func WithAccountTripPackageProvider(provider accountTripPackageProvider) Option {
	return func(s *Service) { s.tripPackageProvider = provider }
}

func (s *Service) accountExports() (accountExportRepository, error) {
	repo, ok := s.repo.(accountExportRepository)
	if !ok {
		return nil, apperrs.NewInvalidInput("data export storage is unavailable")
	}
	if !s.dataExportConfig.Enabled || s.dataExportStorage == nil {
		return nil, apperrs.NewInvalidInput("data exports are disabled")
	}
	return repo, nil
}

// CreateAccountExport creates a short-lived package for the data this service
// owns. Trip data stays in Trip Service and is deliberately not fetched by
// impersonating the user or widening workspace access. The manifest and README
// make that boundary explicit rather than presenting an incomplete package as a
// compliance claim.
func (s *Service) CreateAccountExport(ctx context.Context, request AccountExportRequest) (*dataexport.Job, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := s.accountExports()
	if err != nil {
		return nil, err
	}
	if !anyAccountSection(request.Sections) {
		request.Sections.Profile, request.Sections.Preferences = true, true
	}
	scope, _ := json.Marshal(request)
	job, err := repo.CreateAccountExportJob(ctx, dataexport.Job{ID: uuid.New(), UserID: user.ID, ExportType: dataexport.TypeAccount, Status: dataexport.Queued, Scope: scope})
	if err != nil {
		return nil, err
	}
	packageBytes, err := s.buildAccountExport(ctx, user.ID, request)
	if err != nil {
		_ = repo.FailAccountExportJob(ctx, job.ID, "export_generation_failed", "We could not create this export. Please try again.")
		job.Status = dataexport.Failed
		job.ErrorCode = accountStringPtr("export_generation_failed")
		job.ErrorMessageSafe = accountStringPtr("We could not create this export. Please try again.")
		accountDataExportJobs.WithLabelValues("generation_failed").Inc()
		return job, nil
	}
	if max := s.dataExportConfig.MaxAccountBytes; max > 0 && int64(len(packageBytes)) > max {
		_ = repo.FailAccountExportJob(ctx, job.ID, "export_too_large", "This export is too large to create safely.")
		job.Status = dataexport.Failed
		job.ErrorCode = accountStringPtr("export_too_large")
		job.ErrorMessageSafe = accountStringPtr("This export is too large to create safely.")
		accountDataExportJobs.WithLabelValues("too_large").Inc()
		return job, nil
	}
	key := "account-exports/" + job.ID.String() + ".zip"
	size, checksum, err := s.dataExportStorage.Save(key, packageBytes)
	if err != nil {
		_ = repo.FailAccountExportJob(ctx, job.ID, "export_storage_failed", "We could not store this export. Please try again.")
		return nil, err
	}
	completed, err := repo.CompleteAccountExportJob(ctx, job.ID, key, "travel-data-export-"+time.Now().UTC().Format("2006-01-02")+".zip", "application/zip", size, checksum, time.Now().UTC().Add(s.dataExportConfig.TTL))
	if err != nil {
		_ = s.dataExportStorage.Delete(key)
		return nil, err
	}
	s.log.Info("account export completed", zap.String("export_id", completed.ID.String()), zap.String("user_id", user.ID.String()), zap.Int64("size_bytes", size))
	accountDataExportJobs.WithLabelValues("completed").Inc()
	accountDataExportBytes.Observe(float64(size))
	return completed, nil
}

func (s *Service) GetAccountExport(ctx context.Context, exportID uuid.UUID) (*dataexport.Job, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := s.accountExports()
	if err != nil {
		return nil, err
	}
	job, err := repo.GetAccountExportJob(ctx, exportID, user.ID)
	if err != nil {
		return nil, err
	}
	if job.Status == dataexport.Completed && job.ExpiresAt != nil && !job.ExpiresAt.After(time.Now().UTC()) {
		if err := repo.ExpireAccountExportJob(ctx, job.ID); err != nil {
			return nil, err
		}
		job.Status, job.FilePath = dataexport.Expired, nil
	}
	return job, nil
}

func (s *Service) OpenAccountExport(ctx context.Context, exportID uuid.UUID) (*ExportDownload, error) {
	job, err := s.GetAccountExport(ctx, exportID)
	if err != nil {
		return nil, err
	}
	if job.Status != dataexport.Completed || job.FilePath == nil || job.FileName == nil || job.ExpiresAt == nil || !job.ExpiresAt.After(time.Now().UTC()) {
		return nil, apperrs.NewInvalidInput("export is not available for download")
	}
	reader, err := s.dataExportStorage.Open(*job.FilePath)
	if err != nil {
		return nil, apperrs.NewInvalidInput("export file is no longer available")
	}
	return &ExportDownload{Reader: reader, Filename: *job.FileName, ContentType: "application/zip", SizeBytes: valueOrZero(job.SizeBytes)}, nil
}

func (s *Service) RequestAccountCleanup(ctx context.Context, request AccountCleanupRequest) error {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(request.Reason)) > 1000 {
		return apperrs.NewInvalidInput("reason must be at most 1000 characters")
	}
	repo, ok := s.repo.(interface {
		CreateAccountCleanupRequest(context.Context, uuid.UUID, uuid.UUID, *string, bool) error
	})
	if !ok {
		return apperrs.NewInvalidInput("account cleanup requests are unavailable")
	}
	var reason *string
	if cleaned := strings.TrimSpace(request.Reason); cleaned != "" {
		reason = &cleaned
	}
	return repo.CreateAccountCleanupRequest(ctx, uuid.New(), user.ID, reason, request.ExportRequestedFirst)
}

func (s *Service) CleanupExpiredAccountExports(ctx context.Context) (int, error) {
	repo, err := s.accountExports()
	if err != nil {
		return 0, err
	}
	jobs, err := repo.ListExpiredAccountExportJobs(ctx, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	count := 0
	for _, job := range jobs {
		if job.FilePath != nil {
			_ = s.dataExportStorage.Delete(*job.FilePath)
		}
		if err := repo.ExpireAccountExportJob(ctx, job.ID); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func StartAccountExportCleanupLoop(parent context.Context, svc *Service, interval time.Duration, log *zap.Logger) func(context.Context) error {
	if svc == nil || !svc.dataExportConfig.Enabled || svc.dataExportStorage == nil {
		return func(context.Context) error { return nil }
	}
	if interval <= 0 {
		interval = time.Hour
	}
	if log == nil {
		log = zap.NewNop()
	}
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	go func() {
		defer close(done)
		cleanup := func() {
			if count, err := svc.CleanupExpiredAccountExports(ctx); err != nil {
				log.Warn("account export cleanup failed", zap.Error(err))
			} else if count > 0 {
				log.Info("account exports expired", zap.Int("count", count))
			}
		}
		cleanup()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanup()
			}
		}
	}()
	return func(stopCtx context.Context) error {
		cancel()
		select {
		case <-done:
			return nil
		case <-stopCtx.Done():
			return stopCtx.Err()
		}
	}
}

func (s *Service) buildAccountExport(ctx context.Context, userID uuid.UUID, request AccountExportRequest) ([]byte, error) {
	profile, err := s.GetProfile(ctx)
	if err != nil {
		return nil, err
	}
	preferences, err := s.GetPreferences(ctx)
	if err != nil {
		return nil, err
	}
	root := "account-export"
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	addJSON := func(name string, value any) error {
		raw, marshalErr := json.MarshalIndent(value, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		file, createErr := writer.Create(root + "/" + name)
		if createErr != nil {
			return createErr
		}
		_, writeErr := file.Write(raw)
		return writeErr
	}
	includeTripData := request.Sections.PersonalTrips || request.Sections.TripRecaps || request.Sections.Templates || request.Sections.Expenses || request.Sections.Settlements || request.Sections.Checklists || request.Sections.Reminders
	manifest := map[string]any{"schemaVersion": "account_export_v1", "exportType": "account", "createdAt": time.Now().UTC().Format(time.RFC3339), "userId": userID.String(), "sections": request.Sections, "includedReceiptFiles": request.IncludeReceiptFiles, "tripPackageIncluded": false, "excluded": []string{"passwords", "tokens", "provider credentials", "raw internal logs", "raw receipt OCR"}, "expiresAt": time.Now().UTC().Add(s.dataExportConfig.TTL).Format(time.RFC3339)}
	var tripPackage []byte
	if includeTripData {
		if s.tripPackageProvider == nil {
			return nil, fmt.Errorf("trip export handoff is unavailable")
		}
		tripPackage, err = s.tripPackageProvider.BuildAccountTripPackage(ctx, userID, request.IncludeWorkspaceData, request.IncludeReceiptFiles)
		if err != nil {
			return nil, fmt.Errorf("build account trip package: %w", err)
		}
		manifest["tripPackageIncluded"] = true
	}
	if err := addJSON("manifest.json", manifest); err != nil {
		return nil, err
	}
	if request.Sections.Profile {
		if err := addJSON("profile.json", safeProfile(profile)); err != nil {
			return nil, err
		}
	}
	if request.Sections.Preferences {
		if err := addJSON("preferences.json", safePreferences(preferences)); err != nil {
			return nil, err
		}
	}
	if len(tripPackage) > 0 {
		file, createErr := writer.Create(root + "/trip-data.zip")
		if createErr != nil {
			return nil, createErr
		}
		if _, writeErr := file.Write(tripPackage); writeErr != nil {
			return nil, writeErr
		}
	}
	readme, err := writer.Create(root + "/README.txt")
	if err != nil {
		return nil, err
	}
	_, _ = io.WriteString(readme, "This private export may contain sensitive travel data. It expires after 24 hours.\nWhen selected, trip-data.zip contains nested archives for personal and permitted workspace trips where the user is an owner or editor. Read-only shared trips are not included.\nThis package excludes passwords, authentication tokens, provider credentials, internal logs, raw AI traces, raw receipt OCR, and other users' private data.\n")
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finish account export: %w", err)
	}
	return output.Bytes(), nil
}

func anyAccountSection(s AccountExportSections) bool {
	return s.Profile || s.Preferences || s.PersonalTrips || s.TripRecaps || s.Templates || s.Expenses || s.Settlements || s.Checklists || s.Reminders || s.PersonalizationFeedback || s.NotificationPreferences || s.Notifications
}
func valueOrZero(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
func accountStringPtr(value string) *string { return &value }
func safeProfile(profile any) any           { return profile }
func safePreferences(preferences any) any   { return preferences }
