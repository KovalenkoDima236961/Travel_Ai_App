package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/dataexport"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type tripExportRepository interface {
	CreateDataExportJob(context.Context, dataexport.Job) (*dataexport.Job, error)
	GetDataExportJob(context.Context, uuid.UUID, uuid.UUID) (*dataexport.Job, error)
	CompleteDataExportJob(context.Context, uuid.UUID, string, string, string, int64, string, time.Time) (*dataexport.Job, error)
	FailDataExportJob(context.Context, uuid.UUID, string, string) error
	ListExpiredDataExportJobs(context.Context, time.Time) ([]dataexport.Job, error)
	ExpireDataExportJob(context.Context, uuid.UUID) error
}

type TripArchiveExportInput struct {
	IncludeReceiptFiles bool `json:"includeReceiptFiles"`
	IncludeRecapPDF     bool `json:"includeRecapPdf"`
	IncludePrivateNotes bool `json:"includePrivateNotes"`
}

type ExportFile struct {
	Reader      io.ReadCloser
	Filename    string
	ContentType string
	SizeBytes   int64
}

// BuildAccountTripPackage is an internal, service-token-protected handoff for
// User Service. It applies the same owner/editor rule as browser archive
// exports, so a read-only collaboration never becomes export authority.
func (s *Service) BuildAccountTripPackage(ctx context.Context, userID uuid.UUID, includeWorkspaceData, includeReceiptFiles bool) ([]byte, error) {
	ctx = auth.WithUser(ctx, auth.AuthenticatedUser{ID: userID})
	scope := appdto.TripListScopePersonal
	if includeWorkspaceData {
		scope = appdto.TripListScopeAll
	}

	type archive struct {
		TripID string
		Bytes  []byte
	}
	archives := make([]archive, 0)
	skippedReadOnlyTripIDs := make([]string, 0)
	for offset := 0; ; offset += 100 {
		trips, _, _, err := s.ListWithFilters(ctx, appdto.ListTripsInput{Limit: 100, Offset: offset, Scope: scope, IncludeArchived: true})
		if err != nil {
			return nil, err
		}
		for _, trip := range trips {
			_, access, accessErr := s.requireViewerEditorOrOwner(ctx, trip.ID, userID)
			if accessErr != nil || (access.Level != AccessLevelOwner && access.Level != AccessLevelEditor) {
				skippedReadOnlyTripIDs = append(skippedReadOnlyTripIDs, trip.ID.String())
				continue
			}
			contents, _, archiveErr := s.buildTripArchive(ctx, trip.ID, includeReceiptFiles)
			if archiveErr != nil {
				return nil, archiveErr
			}
			archives = append(archives, archive{TripID: trip.ID.String(), Bytes: contents})
		}
		if len(trips) < 100 {
			break
		}
	}

	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	manifest, err := json.MarshalIndent(map[string]any{
		"schemaVersion":          "account_trip_package_v1",
		"createdAt":              time.Now().UTC().Format(time.RFC3339),
		"includedTripCount":      len(archives),
		"skippedReadOnlyTripIds": skippedReadOnlyTripIDs,
		"includedReceiptFiles":   includeReceiptFiles,
	}, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := addZipFile(writer, "manifest.json", manifest); err != nil {
		return nil, err
	}
	for _, archive := range archives {
		if err := addZipFile(writer, "trips/"+archive.TripID+".zip", archive.Bytes); err != nil {
			return nil, err
		}
	}
	if err := addZipFile(writer, "README.txt", []byte("This private package contains nested trip archives authorized for the requesting user. Read-only shared trips are not included. Receipt files are included only when explicitly requested.\n")); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finish account trip package: %w", err)
	}
	return output.Bytes(), nil
}

// WithDataExports wires private storage and lifecycle controls. It deliberately
// does not enable public storage, signed URLs, or account deletion.
func WithDataExports(storage *dataexport.LocalStorage, cfg dataexport.Config) Option {
	return func(s *Service) {
		s.dataExportStorage = storage
		s.dataExportConfig = cfg
	}
}

func (s *Service) dataExportRepo() (tripExportRepository, error) {
	repo, ok := s.repo.(tripExportRepository)
	if !ok {
		return nil, apperrs.NewDependencyError("data export storage is not configured")
	}
	if !s.dataExportConfig.Enabled || s.dataExportStorage == nil {
		return nil, apperrs.NewDependencyError("data exports are disabled")
	}
	return repo, nil
}

// CreateTripArchiveExport writes a private, short-lived package synchronously.
// Jobs are persisted first so failure is auditable and never leaves an exposed
// file. The API keeps the job shape so larger packages can move to a worker
// without changing clients.
func (s *Service) CreateTripArchiveExport(ctx context.Context, tripID uuid.UUID, input TripArchiveExportInput) (*dataexport.Job, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return nil, err
	}
	repo, err := s.dataExportRepo()
	if err != nil {
		return nil, err
	}
	scope, _ := json.Marshal(map[string]any{
		"tripId": tripID.String(), "includeReceiptFiles": input.IncludeReceiptFiles,
		"includePrivateNotes": false,
	})
	job, err := repo.CreateDataExportJob(ctx, dataexport.Job{
		ID: uuid.New(), UserID: user.ID, ExportType: dataexport.TypeTripArchive,
		Status: dataexport.StatusQueued, Scope: scope,
	})
	if err != nil {
		return nil, err
	}

	contents, fileName, buildErr := s.buildTripArchive(ctx, tripID, input.IncludeReceiptFiles)
	if buildErr != nil {
		_ = repo.FailDataExportJob(ctx, job.ID, "export_generation_failed", "We could not create this export. Please try again.")
		job.Status = dataexport.StatusFailed
		job.ErrorCode = stringPtr("export_generation_failed")
		job.ErrorMessageSafe = stringPtr("We could not create this export. Please try again.")
		s.log.Warn("trip export failed", zap.String("export_id", job.ID.String()), zap.String("trip_id", tripID.String()), zap.Error(buildErr))
		tripDataExportJobs.WithLabelValues("generation_failed").Inc()
		return job, nil
	}
	if max := s.dataExportConfig.MaxTripBytes; max > 0 && int64(len(contents)) > max {
		_ = repo.FailDataExportJob(ctx, job.ID, "export_too_large", "This export is too large to create safely.")
		job.Status = dataexport.StatusFailed
		job.ErrorCode = stringPtr("export_too_large")
		job.ErrorMessageSafe = stringPtr("This export is too large to create safely.")
		tripDataExportJobs.WithLabelValues("too_large").Inc()
		return job, nil
	}
	storageKey := "trip-exports/" + job.ID.String() + ".zip"
	_, size, checksum, saveErr := s.dataExportStorage.Save(storageKey, contents)
	if saveErr != nil {
		_ = repo.FailDataExportJob(ctx, job.ID, "export_storage_failed", "We could not store this export. Please try again.")
		return nil, saveErr
	}
	expiresAt := time.Now().UTC().Add(s.dataExportConfig.TTL)
	completed, err := repo.CompleteDataExportJob(ctx, job.ID, storageKey, fileName, "application/zip", size, checksum, expiresAt)
	if err != nil {
		_ = s.dataExportStorage.Delete(storageKey)
		return nil, err
	}
	s.log.Info("trip export completed", zap.String("export_id", job.ID.String()), zap.String("trip_id", tripID.String()), zap.Bool("include_receipts", input.IncludeReceiptFiles), zap.Int64("size_bytes", size))
	tripDataExportJobs.WithLabelValues("completed").Inc()
	tripDataExportBytes.Observe(float64(size))
	return completed, nil
}

func (s *Service) GetTripExport(ctx context.Context, tripID, exportID uuid.UUID) (*dataexport.Job, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return nil, err
	}
	repo, err := s.dataExportRepo()
	if err != nil {
		return nil, err
	}
	job, err := repo.GetDataExportJob(ctx, exportID, user.ID)
	if err != nil {
		return nil, err
	}
	if job.ExportType != dataexport.TypeTripArchive || !jobHasTrip(job, tripID) {
		return nil, apperrs.ErrForbidden
	}
	return expireJobIfNeeded(ctx, repo, job)
}

func (s *Service) OpenTripExport(ctx context.Context, tripID, exportID uuid.UUID) (*ExportFile, error) {
	job, err := s.GetTripExport(ctx, tripID, exportID)
	if err != nil {
		return nil, err
	}
	if job.Status != dataexport.StatusCompleted || job.FilePath == nil || job.FileName == nil || job.ExpiresAt == nil || !job.ExpiresAt.After(time.Now().UTC()) {
		return nil, apperrs.NewConflict("export is not available for download")
	}
	reader, err := s.dataExportStorage.Open(*job.FilePath)
	if err != nil {
		return nil, apperrs.NewConflict("export file is no longer available")
	}
	return &ExportFile{Reader: reader, Filename: *job.FileName, ContentType: "application/zip", SizeBytes: derefInt64(job.SizeBytes)}, nil
}

// CleanupExpiredTripExports removes only generated packages. User source data
// and receipts are never deleted by this maintenance loop.
func (s *Service) CleanupExpiredTripExports(ctx context.Context) (int, error) {
	repo, err := s.dataExportRepo()
	if err != nil {
		return 0, err
	}
	jobs, err := repo.ListExpiredDataExportJobs(ctx, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	count := 0
	for _, job := range jobs {
		if job.FilePath != nil {
			_ = s.dataExportStorage.Delete(*job.FilePath)
		}
		if err := repo.ExpireDataExportJob(ctx, job.ID); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func StartTripExportCleanupLoop(parent context.Context, svc *Service, interval time.Duration, log *zap.Logger) func(context.Context) error {
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
			if count, err := svc.CleanupExpiredTripExports(ctx); err != nil {
				log.Warn("data export cleanup failed", zap.Error(err))
			} else if count > 0 {
				log.Info("data exports expired", zap.Int("count", count))
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

func expireJobIfNeeded(ctx context.Context, repo tripExportRepository, job *dataexport.Job) (*dataexport.Job, error) {
	if job != nil && job.Status == dataexport.StatusCompleted && job.ExpiresAt != nil && !job.ExpiresAt.After(time.Now().UTC()) {
		if err := repo.ExpireDataExportJob(ctx, job.ID); err != nil {
			return nil, err
		}
		job.Status, job.FilePath = dataexport.StatusExpired, nil
	}
	return job, nil
}

func jobHasTrip(job *dataexport.Job, tripID uuid.UUID) bool {
	if job == nil {
		return false
	}
	var scope struct {
		TripID string `json:"tripId"`
	}
	return json.Unmarshal(job.Scope, &scope) == nil && scope.TripID == tripID.String()
}

func (s *Service) ExportTripCSV(ctx context.Context, tripID uuid.UUID, kind string) ([]byte, string, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, "", err
	}
	if _, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return nil, "", err
	}
	switch kind {
	case "expenses":
		return s.exportExpensesCSV(ctx, tripID)
	case "settlements":
		return s.exportSettlementsCSV(ctx, tripID)
	case "budget":
		return s.exportBudgetCSV(ctx, tripID)
	case "receipt-metadata":
		return s.exportReceiptMetadataCSV(ctx, tripID)
	default:
		return nil, "", apperrs.NewInvalidInput("unsupported CSV export")
	}
}

func (s *Service) buildTripArchive(ctx context.Context, tripID uuid.UUID, includeReceiptFiles bool) ([]byte, string, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, "", err
	}
	trip, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, "", err
	}
	budgetSummary, err := s.GetBudgetSummary(ctx, tripID)
	if err != nil {
		return nil, "", err
	}
	expensesCSV, _, err := s.exportExpensesCSV(ctx, tripID)
	if err != nil {
		return nil, "", err
	}
	settlementsCSV, _, err := s.exportSettlementsCSV(ctx, tripID)
	if err != nil {
		return nil, "", err
	}
	budgetCSV, _, err := s.exportBudgetCSV(ctx, tripID)
	if err != nil {
		return nil, "", err
	}
	receiptsCSV, _, err := s.exportReceiptMetadataCSV(ctx, tripID)
	if err != nil {
		return nil, "", err
	}
	checklist, err := s.GetTripChecklist(ctx, tripID)
	if err != nil {
		return nil, "", err
	}
	reminders, err := s.ListTripReminders(ctx, tripID, appdto.ReminderListFilters{})
	if err != nil {
		return nil, "", err
	}
	verification, verificationErr := s.GetTripVerification(ctx, tripID)
	var recap any
	if recapResult, recapErr := s.GetTripRecap(ctx, tripID); recapErr == nil {
		recap = recapResult
	}

	root := "trip-archive-" + archiveFilePart(trip.Destination) + "-" + time.Now().UTC().Format("2006-01-02")
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	addJSON := func(name string, value any) error {
		raw, marshalErr := json.MarshalIndent(value, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		return addZipFile(writer, root+"/"+name, raw)
	}
	manifest := map[string]any{
		"schemaVersion": "trip_archive_export_v1", "exportType": dataexport.TypeTripArchive,
		"tripId": trip.ID.String(), "tripTitle": trip.Destination, "createdAt": time.Now().UTC().Format(time.RFC3339),
		"includedReceiptFiles": includeReceiptFiles, "includedPrivateNotes": false,
		"excluded": []string{"collaborator private data", "raw receipt OCR", "tokens", "internal activity logs", "public share secrets"},
	}
	if err := addJSON("manifest.json", manifest); err != nil {
		return nil, "", err
	}
	if err := addJSON("trip.json", safeTripExport(trip)); err != nil {
		return nil, "", err
	}
	if err := addRawJSON(writer, root+"/itinerary.json", trip.Itinerary); err != nil {
		return nil, "", err
	}
	if err := addJSON("route.json", trip.Route); err != nil {
		return nil, "", err
	}
	if err := addJSON("accommodation.json", trip.Accommodation); err != nil {
		return nil, "", err
	}
	if err := addJSON("budget-summary.json", budgetSummary); err != nil {
		return nil, "", err
	}
	if err := addZipFile(writer, root+"/budget-summary.csv", budgetCSV); err != nil {
		return nil, "", err
	}
	if err := addZipFile(writer, root+"/expenses.csv", expensesCSV); err != nil {
		return nil, "", err
	}
	if err := addZipFile(writer, root+"/settlements.csv", settlementsCSV); err != nil {
		return nil, "", err
	}
	if err := addZipFile(writer, root+"/receipt-metadata.csv", receiptsCSV); err != nil {
		return nil, "", err
	}
	if err := addJSON("checklist.json", checklist); err != nil {
		return nil, "", err
	}
	if err := addJSON("reminders.json", reminders); err != nil {
		return nil, "", err
	}
	if recap != nil {
		if err := addJSON("recap.json", recap); err != nil {
			return nil, "", err
		}
	}
	if verificationErr == nil {
		if err := addJSON("verification-summary.json", verification); err != nil {
			return nil, "", err
		}
	}
	if includeReceiptFiles {
		if err := s.addReceiptFilesToArchive(ctx, writer, root, tripID, s.dataExportConfig.MaxTripBytes-int64(output.Len())); err != nil {
			return nil, "", err
		}
	}
	readme := "This archive is private and may contain sensitive travel and expense data.\nReceipt files are included only when explicitly selected.\nIt excludes passwords, tokens, provider credentials, internal logs, raw OCR text, and collaborators' private data.\n"
	if err := addZipFile(writer, root+"/README.txt", []byte(readme)); err != nil {
		return nil, "", err
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("finish archive: %w", err)
	}
	return output.Bytes(), root + ".zip", nil
}

func (s *Service) addReceiptFilesToArchive(ctx context.Context, writer *zip.Writer, root string, tripID uuid.UUID, remaining int64) error {
	if s.receiptStorage == nil {
		return nil
	}
	receiptRows, err := s.repo.ListTripExpenseReceipts(ctx, tripID, appdto.ListReceiptsInput{Limit: 100})
	if err != nil {
		return err
	}
	for _, receipt := range receiptRows {
		if remaining <= 0 {
			break
		}
		if receipt.SizeBytes > remaining {
			continue
		}
		file, openErr := s.receiptStorage.Open(ctx, receipt.StorageKey)
		if openErr != nil {
			continue
		}
		content, readErr := io.ReadAll(io.LimitReader(file.Reader, receipt.SizeBytes+1))
		_ = file.Reader.Close()
		if readErr != nil || int64(len(content)) > receipt.SizeBytes || int64(len(content)) > remaining {
			continue
		}
		name := archiveFilePart(strings.TrimSuffix(receipt.OriginalFilename, filepath.Ext(receipt.OriginalFilename))) + filepath.Ext(receipt.OriginalFilename)
		if name == "." || name == "" {
			name = receipt.ID.String()
		}
		if err := addZipFile(writer, root+"/receipts/"+receipt.ID.String()+"-"+name, content); err != nil {
			return err
		}
		remaining -= int64(len(content))
	}
	return nil
}

func (s *Service) exportExpensesCSV(ctx context.Context, tripID uuid.UUID) ([]byte, string, error) {
	rows, err := s.repo.ListTripExpensesByTrip(ctx, tripID, appdto.ListExpensesInput{Limit: 1000})
	if err != nil {
		return nil, "", err
	}
	receipts, err := s.repo.ListTripExpenseReceipts(ctx, tripID, appdto.ListReceiptsInput{Limit: 1000})
	if err != nil {
		return nil, "", err
	}
	counts := map[uuid.UUID]int{}
	for _, receipt := range receipts {
		if receipt.ExpenseID != nil {
			counts[*receipt.ExpenseID]++
		}
	}
	data := newCSV([]string{"expense_id", "date", "merchant", "title", "category", "amount", "currency", "paid_by", "split_type", "participant_count", "receipt_attached", "source", "created_at"})
	participants, err := s.repo.ListExpenseParticipantsByTrip(ctx, tripID)
	if err != nil {
		return nil, "", err
	}
	participantCounts := map[uuid.UUID]int{}
	for _, participant := range participants {
		participantCounts[participant.ExpenseID]++
	}
	for _, row := range rows {
		data.write([]string{row.ID.String(), isoDate(row.ExpenseDate), "", row.Title, string(row.Category), decimal(row.Amount), row.Currency, row.PaidByUserID.String(), string(row.SplitType), fmt.Sprint(participantCounts[row.ID]), boolString(counts[row.ID] > 0), "manual", row.CreatedAt.UTC().Format(time.RFC3339)})
	}
	return data.bytes(), "expenses.csv", nil
}

func (s *Service) exportSettlementsCSV(ctx context.Context, tripID uuid.UUID) ([]byte, string, error) {
	rows, err := s.repo.ListTripSettlementsByTrip(ctx, tripID)
	if err != nil {
		return nil, "", err
	}
	data := newCSV([]string{"settlement_id", "from_traveler", "to_traveler", "amount", "currency", "status", "marked_paid_at", "created_at"})
	for _, row := range rows {
		paidAt := ""
		if row.PaidAt != nil {
			paidAt = row.PaidAt.UTC().Format(time.RFC3339)
		}
		data.write([]string{row.ID.String(), row.FromUserID.String(), row.ToUserID.String(), decimal(row.Amount), row.Currency, string(row.Status), paidAt, row.CreatedAt.UTC().Format(time.RFC3339)})
	}
	return data.bytes(), "settlements.csv", nil
}

func (s *Service) exportBudgetCSV(ctx context.Context, tripID uuid.UUID) ([]byte, string, error) {
	trip, err := s.repo.GetByID(ctx, tripID)
	if err != nil {
		return nil, "", err
	}
	summary := budgetSummaryForTrip(trip)
	actual := map[string]float64{}
	expenses, err := s.repo.ListTripExpensesByTrip(ctx, tripID, appdto.ListExpensesInput{Limit: 1000})
	if err != nil {
		return nil, "", err
	}
	for _, expense := range expenses {
		if strings.EqualFold(expense.Currency, summary.Currency) {
			actual[string(expense.Category)] += expense.Amount
		}
	}
	data := newCSV([]string{"category", "planned_amount", "estimated_amount", "actual_amount", "currency", "variance_amount", "variance_percent", "confidence", "source_quality"})
	for _, category := range summary.ByCategory {
		actualAmount := actual[category.Category]
		variance := actualAmount - category.EstimatedTotal
		percent := ""
		if category.EstimatedTotal != 0 {
			percent = decimal((variance / category.EstimatedTotal) * 100)
		}
		data.write([]string{category.Category, "", decimal(category.EstimatedTotal), decimal(actualAmount), summary.Currency, decimal(variance), percent, "not_scored", "itinerary_estimate"})
	}
	return data.bytes(), "budget-summary.csv", nil
}

func (s *Service) exportReceiptMetadataCSV(ctx context.Context, tripID uuid.UUID) ([]byte, string, error) {
	receipts, err := s.repo.ListTripExpenseReceipts(ctx, tripID, appdto.ListReceiptsInput{Limit: 1000})
	if err != nil {
		return nil, "", err
	}
	results, err := s.repo.ListLatestReceiptOCRResults(ctx, tripID, receiptIDs(receipts))
	if err != nil {
		return nil, "", err
	}
	byReceipt := map[uuid.UUID]entity.ReceiptOCRResult{}
	for _, result := range results {
		byReceipt[result.ReceiptID] = result
	}
	data := newCSV([]string{"receipt_id", "filename", "mime_type", "size_bytes", "linked_expense_id", "upload_date", "extraction_status", "merchant_reviewed", "amount_reviewed", "currency_reviewed"})
	for _, receipt := range receipts {
		linked := ""
		if receipt.ExpenseID != nil {
			linked = receipt.ExpenseID.String()
		}
		result, found := byReceipt[receipt.ID]
		merchant, amount, currency := "", "", ""
		if found {
			if result.Merchant != nil {
				merchant = *result.Merchant
			}
			if result.Amount != nil {
				amount = decimal(*result.Amount)
			}
			if result.Currency != nil {
				currency = *result.Currency
			}
		}
		data.write([]string{receipt.ID.String(), receipt.OriginalFilename, receipt.ContentType, fmt.Sprint(receipt.SizeBytes), linked, receipt.CreatedAt.UTC().Format(time.RFC3339), string(receipt.Status), merchant, amount, currency})
	}
	return data.bytes(), "receipt-metadata.csv", nil
}

type csvExport struct {
	buffer bytes.Buffer
	writer *csv.Writer
}

func newCSV(header []string) *csvExport {
	value := &csvExport{}
	value.writer = csv.NewWriter(&value.buffer)
	value.write(header)
	return value
}
func (c *csvExport) write(row []string) {
	for index := range row {
		row[index] = csvSafe(row[index])
	}
	_ = c.writer.Write(row)
}
func (c *csvExport) bytes() []byte { c.writer.Flush(); return c.buffer.Bytes() }
func csvSafe(value string) string {
	value = strings.TrimSpace(value)
	if value != "" && strings.ContainsRune("=+-@", rune(value[0])) {
		return "'" + value
	}
	return value
}
func decimal(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value), "0"), ".")
}
func isoDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format("2006-01-02")
}
func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
func receiptIDs(receipts []entity.TripExpenseReceipt) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(receipts))
	for _, receipt := range receipts {
		ids = append(ids, receipt.ID)
	}
	return ids
}
func derefInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func addZipFile(writer *zip.Writer, name string, content []byte) error {
	file, err := writer.Create(safeArchivePath(name))
	if err != nil {
		return err
	}
	_, err = file.Write(content)
	return err
}
func addRawJSON(writer *zip.Writer, name string, content []byte) error {
	if len(bytes.TrimSpace(content)) == 0 {
		content = []byte("{}")
	}
	if !json.Valid(content) {
		content = []byte("{}")
	}
	return addZipFile(writer, name, content)
}
func safeArchivePath(name string) string {
	parts := strings.Split(filepath.ToSlash(name), "/")
	safe := make([]string, 0, len(parts))
	for _, part := range parts {
		part = archiveFilePart(part)
		if part != "" {
			safe = append(safe, part)
		}
	}
	return strings.Join(safe, "/")
}
func archiveFilePart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '.' || char == '-' || char == '_' {
			b.WriteRune(char)
		} else {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), ".-")
	if out == "" {
		return "export"
	}
	if len(out) > 100 {
		return out[:100]
	}
	return out
}

func safeTripExport(trip *entity.Trip) map[string]any {
	if trip == nil {
		return map[string]any{}
	}
	value := map[string]any{"id": trip.ID.String(), "tripType": trip.TripType, "destination": trip.Destination, "days": trip.Days, "travelers": trip.Travelers, "interests": trip.Interests, "pace": trip.Pace, "status": trip.Status, "itineraryRevision": trip.ItineraryRevision, "archivedAt": trip.ArchivedAt, "createdAt": trip.CreatedAt.UTC(), "updatedAt": trip.UpdatedAt.UTC()}
	if trip.StartDate != nil {
		value["startDate"] = trip.StartDate.UTC().Format("2006-01-02")
	}
	if trip.BudgetAmount != nil {
		value["budget"] = map[string]any{"amount": *trip.BudgetAmount, "currency": trip.BudgetCurrency}
	}
	return value
}

// Kept deterministic for archive contents and tests even if repository order
// differs across PostgreSQL query plans.
func sortedReceiptIDs(receipts []entity.TripExpenseReceipt) []entity.TripExpenseReceipt {
	sort.Slice(receipts, func(i, j int) bool { return receipts[i].ID.String() < receipts[j].ID.String() })
	return receipts
}
