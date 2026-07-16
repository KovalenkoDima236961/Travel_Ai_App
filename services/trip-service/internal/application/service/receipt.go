package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/receipts"
	tripsecurity "github.com/KovalenkoDima236961/Travel_Ai_App/internal/security"
	"go.uber.org/zap"
)

type ReceiptFile struct {
	Reader      io.ReadCloser
	ContentType string
	Filename    string
	SizeBytes   int64
}

func (s *Service) ReceiptMaxUploadBytes() int64 {
	if s.receiptConfig.MaxFileSizeBytes > 0 {
		return s.receiptConfig.MaxFileSizeBytes
	}
	maxMB := s.receiptConfig.MaxFileSizeMB
	if maxMB <= 0 {
		maxMB = receipts.DefaultConfig().MaxFileSizeMB
	}
	return int64(maxMB) * 1024 * 1024
}

func (s *Service) UploadReceipt(ctx context.Context, tripID uuid.UUID, in appdto.UploadReceiptInput) (appdto.ExpenseReceipt, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.ExpenseReceipt{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.ExpenseReceipt{}, err
	}
	if !access.Allows(tripsecurity.PermissionReceiptsUpload) {
		return appdto.ExpenseReceipt{}, apperrs.ErrForbidden
	}
	if in.File == nil {
		return appdto.ExpenseReceipt{}, apperrs.NewInvalidInput("file is required")
	}
	if s.receiptStorage == nil {
		return appdto.ExpenseReceipt{}, apperrs.NewConflict("receipt storage is not configured")
	}
	var expense *entity.TripExpense
	if in.ExpenseID != nil {
		expense, err = s.requireAttachableExpense(ctx, tripID, *in.ExpenseID, user.ID, access)
		if err != nil {
			return appdto.ExpenseReceipt{}, err
		}
	}
	contentType, reader, err := s.validateReceiptFile(in)
	if err != nil {
		return appdto.ExpenseReceipt{}, err
	}
	saveResult, err := s.receiptStorage.Save(ctx, receipts.StorageSaveInput{
		Reader:           io.LimitReader(reader, s.ReceiptMaxUploadBytes()+1),
		OriginalFilename: cleanReceiptFilename(in.OriginalFilename),
		ContentType:      contentType,
	})
	if err != nil {
		return appdto.ExpenseReceipt{}, err
	}
	if saveResult.SizeBytes > s.ReceiptMaxUploadBytes() {
		_ = s.receiptStorage.Delete(ctx, saveResult.StorageKey)
		return appdto.ExpenseReceipt{}, apperrs.NewInvalidInput("file too large")
	}
	if err := s.scanReceiptFile(ctx, saveResult.StorageKey); err != nil {
		_ = s.receiptStorage.Delete(ctx, saveResult.StorageKey)
		return appdto.ExpenseReceipt{}, err
	}
	status := entity.ReceiptStatusUploaded
	if expense != nil {
		status = entity.ReceiptStatusAttached
	}
	receipt := &entity.TripExpenseReceipt{
		ID:               uuid.New(),
		TripID:           tripID,
		ExpenseID:        in.ExpenseID,
		Status:           status,
		OriginalFilename: cleanReceiptFilename(in.OriginalFilename),
		ContentType:      contentType,
		SizeBytes:        saveResult.SizeBytes,
		StorageKey:       saveResult.StorageKey,
		FileSHA256:       &saveResult.SHA256,
		CreatedByUserID:  user.ID,
		UpdatedByUserID:  &user.ID,
	}
	created, err := s.repo.CreateTripExpenseReceipt(ctx, receipt)
	if err != nil {
		_ = s.receiptStorage.Delete(ctx, saveResult.StorageKey)
		return appdto.ExpenseReceipt{}, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventReceiptUploaded,
		EntityType:  activityEntityType(activity.EntityTripExpenseReceipt),
		EntityID:    activityEntityID(created.ID),
		Metadata:    receiptActivityMetadata(created, nil),
	})
	var latest *entity.ReceiptOCRResult
	if in.RunOCR && s.receiptConfig.OCREnabled {
		extracted, extractErr := s.extractReceipt(ctx, trip, created, user.ID)
		if extractErr != nil {
			if !s.receiptConfig.OCRFailOpen {
				return appdto.ExpenseReceipt{}, extractErr
			}
			s.log.Warn("receipt OCR failed during upload",
				zap.String("trip_id", tripID.String()),
				zap.String("receipt_id", created.ID.String()),
				zap.Error(extractErr),
			)
		}
		if extracted != nil {
			latest = extracted
			created, _ = s.repo.GetTripExpenseReceiptByID(ctx, tripID, created.ID, false)
		}
	}
	return receiptDTO(created, latest, true), nil
}

func (s *Service) ListReceipts(ctx context.Context, tripID uuid.UUID, filters appdto.ListReceiptsInput) (appdto.TripReceiptsResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripReceiptsResponse{}, err
	}
	if _, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return appdto.TripReceiptsResponse{}, err
	} else if !access.Allows(tripsecurity.PermissionReceiptsView) {
		return appdto.TripReceiptsResponse{}, apperrs.ErrForbidden
	}
	receiptsList, err := s.repo.ListTripExpenseReceipts(ctx, tripID, filters)
	if err != nil {
		return appdto.TripReceiptsResponse{}, err
	}
	out := make([]appdto.ExpenseReceipt, 0, len(receiptsList))
	for i := range receiptsList {
		latest, err := s.latestReceiptOCR(ctx, tripID, receiptsList[i].ID)
		if err != nil {
			return appdto.TripReceiptsResponse{}, err
		}
		out = append(out, receiptDTO(&receiptsList[i], latest, false))
	}
	return appdto.TripReceiptsResponse{Receipts: out}, nil
}

func (s *Service) GetReceipt(ctx context.Context, tripID, receiptID uuid.UUID) (appdto.ExpenseReceipt, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.ExpenseReceipt{}, err
	}
	if _, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return appdto.ExpenseReceipt{}, err
	} else if !access.Allows(tripsecurity.PermissionReceiptsView) {
		return appdto.ExpenseReceipt{}, apperrs.ErrForbidden
	}
	receipt, err := s.repo.GetTripExpenseReceiptByID(ctx, tripID, receiptID, false)
	if err != nil {
		return appdto.ExpenseReceipt{}, err
	}
	latest, err := s.latestReceiptOCR(ctx, tripID, receiptID)
	if err != nil {
		return appdto.ExpenseReceipt{}, err
	}
	return receiptDTO(receipt, latest, true), nil
}

func (s *Service) ExtractReceipt(ctx context.Context, tripID, receiptID uuid.UUID, _ appdto.ExtractReceiptInput) (appdto.ExpenseReceipt, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.ExpenseReceipt{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.ExpenseReceipt{}, err
	}
	receipt, err := s.repo.GetTripExpenseReceiptByID(ctx, tripID, receiptID, false)
	if err != nil {
		return appdto.ExpenseReceipt{}, err
	}
	if !canMutateReceipt(access, user.ID, receipt.CreatedByUserID) {
		return appdto.ExpenseReceipt{}, apperrs.ErrForbidden
	}
	result, err := s.extractReceipt(ctx, trip, receipt, user.ID)
	if err != nil {
		return appdto.ExpenseReceipt{}, err
	}
	updated, err := s.repo.GetTripExpenseReceiptByID(ctx, tripID, receiptID, false)
	if err != nil {
		return appdto.ExpenseReceipt{}, err
	}
	return receiptDTO(updated, result, true), nil
}

func (s *Service) CreateExpenseFromReceipt(ctx context.Context, tripID, receiptID uuid.UUID, in appdto.CreateExpenseInput) (appdto.TripExpense, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	receipt, err := s.repo.GetTripExpenseReceiptByID(ctx, tripID, receiptID, false)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	if !canMutateReceipt(access, user.ID, receipt.CreatedByUserID) {
		return appdto.TripExpense{}, apperrs.ErrForbidden
	}
	users, travelers, err := s.expenseUsers(ctx, trip, user)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	if in.Metadata == nil {
		in.Metadata = map[string]any{}
	}
	in.Metadata["sourceReceiptId"] = receiptID.String()
	normalized, participants, err := s.prepareExpenseForCreate(trip, users, travelers, user.ID, access, in)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	created, createdParticipants, err := s.repo.CreateTripExpenseWithParticipants(ctx, normalized, participants)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	linked, err := s.repo.AttachTripExpenseReceipt(ctx, tripID, receiptID, created.ID, user.ID)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventExpenseCreated,
		EntityType:  activityEntityType(activity.EntityTripExpense),
		EntityID:    activityEntityID(created.ID),
		Metadata:    expenseActivityMetadata(created, len(createdParticipants)),
	})
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventExpenseCreatedFromReceipt,
		EntityType:  activityEntityType(activity.EntityTripExpense),
		EntityID:    activityEntityID(created.ID),
		Metadata:    receiptExpenseActivityMetadata(linked, created),
	})
	s.notifyExpenseParticipants(ctx, trip, user.ID, created, createdParticipants, users)
	out := expenseDTO(created, createdParticipants, users)
	summaries, err := s.receiptSummariesForExpense(ctx, tripID, created.ID)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	return withExpenseReceipts(out, summaries), nil
}

func (s *Service) AttachReceiptToExpense(ctx context.Context, tripID, expenseID uuid.UUID, in appdto.AttachReceiptInput) (appdto.ExpenseReceipt, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.ExpenseReceipt{}, err
	}
	_, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.ExpenseReceipt{}, err
	}
	receipt, err := s.repo.GetTripExpenseReceiptByID(ctx, tripID, in.ReceiptID, false)
	if err != nil {
		return appdto.ExpenseReceipt{}, err
	}
	expense, err := s.requireAttachableExpense(ctx, tripID, expenseID, user.ID, access)
	if err != nil {
		return appdto.ExpenseReceipt{}, err
	}
	if !access.CanEdit() && receipt.CreatedByUserID != user.ID && expense.CreatedByUserID != user.ID {
		return appdto.ExpenseReceipt{}, apperrs.ErrForbidden
	}
	attached, err := s.repo.AttachTripExpenseReceipt(ctx, tripID, in.ReceiptID, expenseID, user.ID)
	if err != nil {
		return appdto.ExpenseReceipt{}, err
	}
	latest, err := s.latestReceiptOCR(ctx, tripID, attached.ID)
	if err != nil {
		return appdto.ExpenseReceipt{}, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventReceiptAttached,
		EntityType:  activityEntityType(activity.EntityTripExpenseReceipt),
		EntityID:    activityEntityID(attached.ID),
		Metadata:    receiptActivityMetadata(attached, latest),
	})
	return receiptDTO(attached, latest, true), nil
}

func (s *Service) DeleteReceipt(ctx context.Context, tripID, receiptID uuid.UUID) error {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return err
	}
	_, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return err
	}
	receipt, err := s.repo.GetTripExpenseReceiptByID(ctx, tripID, receiptID, false)
	if err != nil {
		return err
	}
	if !canMutateReceipt(access, user.ID, receipt.CreatedByUserID) {
		return apperrs.ErrForbidden
	}
	deleted, err := s.repo.SoftDeleteTripExpenseReceipt(ctx, tripID, receiptID, user.ID)
	if err != nil {
		return err
	}
	if s.receiptStorage != nil {
		if err := s.receiptStorage.Delete(ctx, deleted.StorageKey); err != nil {
			s.log.Warn("delete receipt file failed",
				zap.String("trip_id", tripID.String()),
				zap.String("receipt_id", receiptID.String()),
				zap.Error(err),
			)
		}
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventReceiptDeleted,
		EntityType:  activityEntityType(activity.EntityTripExpenseReceipt),
		EntityID:    activityEntityID(deleted.ID),
		Metadata:    receiptActivityMetadata(deleted, nil),
	})
	return nil
}

func (s *Service) OpenReceiptFile(ctx context.Context, tripID, receiptID uuid.UUID) (*ReceiptFile, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return nil, err
	} else if !access.Allows(tripsecurity.PermissionReceiptsView) {
		return nil, apperrs.ErrForbidden
	}
	receipt, err := s.repo.GetTripExpenseReceiptByID(ctx, tripID, receiptID, false)
	if err != nil {
		return nil, err
	}
	if s.receiptStorage == nil {
		return nil, apperrs.NewConflict("receipt storage is not configured")
	}
	file, err := s.receiptStorage.Open(ctx, receipt.StorageKey)
	if err != nil {
		return nil, err
	}
	return &ReceiptFile{
		Reader:      file.Reader,
		ContentType: receipt.ContentType,
		Filename:    receipt.OriginalFilename,
		SizeBytes:   receipt.SizeBytes,
	}, nil
}

func (s *Service) extractReceipt(ctx context.Context, trip *entity.Trip, receipt *entity.TripExpenseReceipt, actorUserID uuid.UUID) (*entity.ReceiptOCRResult, error) {
	if !s.receiptConfig.OCREnabled {
		return nil, apperrs.NewConflict("receipt OCR is disabled")
	}
	if s.receiptStorage == nil || s.receiptOCRProvider == nil {
		return nil, apperrs.NewConflict("receipt OCR is not configured")
	}
	if _, err := s.repo.UpdateTripExpenseReceiptStatus(ctx, trip.ID, receipt.ID, entity.ReceiptStatusProcessing, &actorUserID); err != nil {
		return nil, err
	}
	file, err := s.receiptStorage.Open(ctx, receipt.StorageKey)
	if err != nil {
		return nil, err
	}
	defer file.Reader.Close()
	ocrCtx := ctx
	cancel := func() {}
	if s.receiptConfig.OCRTimeout > 0 {
		ocrCtx, cancel = context.WithTimeout(ctx, s.receiptConfig.OCRTimeout)
	}
	defer cancel()
	result, err := s.receiptOCRProvider.Extract(ocrCtx, file.Reader, receipts.OCRMetadata{
		OriginalFilename: receipt.OriginalFilename,
		ContentType:      receipt.ContentType,
		SizeBytes:        receipt.SizeBytes,
	}, receipts.OCRTripContext{DefaultCurrency: trip.BudgetCurrency})
	if err != nil {
		errorMessage := err.Error()
		failed := &entity.ReceiptOCRResult{
			ID:              uuid.New(),
			ReceiptID:       receipt.ID,
			TripID:          trip.ID,
			Provider:        s.receiptOCRProvider.Name(),
			Status:          entity.ReceiptStatusExtractionFailed,
			Confidence:      entity.ReceiptOCRConfidenceLow,
			FieldConfidence: map[string]entity.ReceiptOCRConfidence{},
			Warnings:        []string{"Extraction failed. Enter receipt details manually."},
			ErrorMessage:    &errorMessage,
		}
		created, createErr := s.repo.CreateReceiptOCRResult(ctx, failed)
		if createErr != nil {
			return nil, createErr
		}
		_, _ = s.repo.UpdateTripExpenseReceiptStatus(ctx, trip.ID, receipt.ID, entity.ReceiptStatusExtractionFailed, &actorUserID)
		s.recordActivity(ctx, activity.RecordActivityInput{
			TripID:      trip.ID,
			ActorUserID: &actorUserID,
			EventType:   activity.EventReceiptExtractionFailed,
			EntityType:  activityEntityType(activity.EntityTripExpenseReceipt),
			EntityID:    activityEntityID(receipt.ID),
			Metadata:    receiptActivityMetadata(receipt, created),
		})
		return created, err
	}
	result.ID = uuid.New()
	result.ReceiptID = receipt.ID
	result.TripID = trip.ID
	if result.Provider == "" {
		result.Provider = s.receiptOCRProvider.Name()
	}
	result.Status = entity.ReceiptStatusExtracted
	if result.Confidence == "" {
		result.Confidence = entity.ReceiptOCRConfidenceLow
	}
	if result.FieldConfidence == nil {
		result.FieldConfidence = map[string]entity.ReceiptOCRConfidence{}
	}
	if result.Warnings == nil {
		result.Warnings = []string{}
	}
	if !s.receiptConfig.StoreRawText {
		result.RawText = nil
	}
	created, err := s.repo.CreateReceiptOCRResult(ctx, result)
	if err != nil {
		return nil, err
	}
	_, err = s.repo.UpdateTripExpenseReceiptStatus(ctx, trip.ID, receipt.ID, entity.ReceiptStatusExtracted, &actorUserID)
	if err != nil {
		return nil, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      trip.ID,
		ActorUserID: &actorUserID,
		EventType:   activity.EventReceiptExtracted,
		EntityType:  activityEntityType(activity.EntityTripExpenseReceipt),
		EntityID:    activityEntityID(receipt.ID),
		Metadata:    receiptActivityMetadata(receipt, created),
	})
	return created, nil
}

func (s *Service) requireAttachableExpense(ctx context.Context, tripID, expenseID, actorUserID uuid.UUID, access TripAccess) (*entity.TripExpense, error) {
	expense, err := s.repo.GetTripExpenseByID(ctx, tripID, expenseID)
	if err != nil {
		return nil, err
	}
	if expense.Status != entity.ExpenseStatusActive {
		return nil, domainerrs.ErrNotFound
	}
	if !access.CanEdit() && expense.CreatedByUserID != actorUserID {
		return nil, apperrs.ErrForbidden
	}
	return expense, nil
}

func (s *Service) validateReceiptFile(in appdto.UploadReceiptInput) (string, io.Reader, error) {
	filename := cleanReceiptFilename(in.OriginalFilename)
	if filename == "" {
		return "", nil, apperrs.NewInvalidInput("original filename is required")
	}
	if in.SizeBytes <= 0 {
		return "", nil, apperrs.NewInvalidInput("file is empty")
	}
	if in.SizeBytes > s.ReceiptMaxUploadBytes() {
		return "", nil, apperrs.NewInvalidInput("file too large")
	}
	head := make([]byte, 512)
	n, err := io.ReadFull(in.File, head)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", nil, fmt.Errorf("read receipt file header: %w", err)
	}
	if n == 0 {
		return "", nil, apperrs.NewInvalidInput("file is empty")
	}
	contentType := detectReceiptContentType(head[:n], in.ContentType)
	if !s.allowedReceiptMIME(contentType) {
		return "", nil, apperrs.NewInvalidInput("unsupported file type")
	}
	declaredType := strings.ToLower(strings.TrimSpace(strings.Split(in.ContentType, ";")[0]))
	if declaredType != "" && declaredType != "application/octet-stream" && declaredType != contentType {
		return "", nil, apperrs.NewInvalidInput("declared and detected file types do not match")
	}
	if !validReceiptExtension(filename, contentType, s.receiptConfig.AllowedExtensions) {
		return "", nil, apperrs.NewInvalidInput("unsupported file extension")
	}
	return contentType, io.MultiReader(bytes.NewReader(head[:n]), in.File), nil
}

func (s *Service) scanReceiptFile(ctx context.Context, storageKey string) error {
	if !s.receiptConfig.ScanningEnabled {
		return nil
	}
	pathProvider, ok := s.receiptStorage.(receipts.LocalPathProvider)
	if !ok || s.receiptFileScanner == nil {
		if s.receiptConfig.ScanningFailOpen {
			s.log.Warn("receipt scanner unavailable; upload allowed by fail-open policy")
			return nil
		}
		return apperrs.NewDependencyError("receipt file scanner is unavailable")
	}
	filePath, err := pathProvider.PathForScanning(storageKey)
	if err != nil {
		return apperrs.NewInvalidInput("invalid receipt storage key")
	}
	result, err := s.receiptFileScanner.Scan(ctx, filePath)
	if err != nil || !result.Available {
		if s.receiptConfig.ScanningFailOpen {
			s.log.Warn("receipt scan unavailable; upload allowed by fail-open policy", zap.Error(err))
			return nil
		}
		return apperrs.NewDependencyError("receipt file scan could not be completed")
	}
	if !result.Clean {
		s.log.Warn("receipt upload rejected by file scanner")
		return apperrs.NewInvalidInput("receipt file failed security scan")
	}
	return nil
}

func (s *Service) allowedReceiptMIME(contentType string) bool {
	allowed := s.receiptConfig.AllowedMIMEs
	if len(allowed) == 0 {
		allowed = receipts.DefaultConfig().AllowedMIMEs
	}
	for _, item := range allowed {
		if strings.EqualFold(strings.TrimSpace(item), contentType) {
			return true
		}
	}
	return false
}

func (s *Service) latestReceiptOCR(ctx context.Context, tripID, receiptID uuid.UUID) (*entity.ReceiptOCRResult, error) {
	latest, err := s.repo.GetLatestReceiptOCRResult(ctx, tripID, receiptID)
	if err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return latest, nil
}

func (s *Service) receiptSummariesForExpense(ctx context.Context, tripID, expenseID uuid.UUID) ([]appdto.ExpenseReceiptSummary, error) {
	receiptRows, err := s.repo.ListTripExpenseReceiptsByExpense(ctx, tripID, expenseID)
	if err != nil {
		return nil, err
	}
	summaries := make([]appdto.ExpenseReceiptSummary, 0, len(receiptRows))
	for i := range receiptRows {
		latest, err := s.latestReceiptOCR(ctx, tripID, receiptRows[i].ID)
		if err != nil {
			return nil, err
		}
		var confidence *entity.ReceiptOCRConfidence
		if latest != nil {
			confidence = &latest.Confidence
		}
		summaries = append(summaries, appdto.ExpenseReceiptSummary{
			ID:               receiptRows[i].ID,
			OriginalFilename: receiptRows[i].OriginalFilename,
			ContentType:      receiptRows[i].ContentType,
			Status:           receiptRows[i].Status,
			OCRConfidence:    confidence,
			CreatedAt:        receiptRows[i].CreatedAt,
		})
	}
	return summaries, nil
}

func receiptDTO(receipt *entity.TripExpenseReceipt, latest *entity.ReceiptOCRResult, includeRaw bool) appdto.ExpenseReceipt {
	return appdto.ExpenseReceipt{
		ID:               receipt.ID,
		TripID:           receipt.TripID,
		ExpenseID:        receipt.ExpenseID,
		Status:           receipt.Status,
		OriginalFilename: receipt.OriginalFilename,
		ContentType:      receipt.ContentType,
		SizeBytes:        receipt.SizeBytes,
		PreviewURL:       fmt.Sprintf("/trips/%s/expenses/receipts/%s/file", receipt.TripID, receipt.ID),
		OCRResult:        ocrResultDTO(latest, includeRaw),
		CreatedByUserID:  receipt.CreatedByUserID,
		CreatedAt:        receipt.CreatedAt,
		UpdatedAt:        receipt.UpdatedAt,
	}
}

func ocrResultDTO(result *entity.ReceiptOCRResult, includeRaw bool) *appdto.ReceiptOCRResult {
	if result == nil {
		return nil
	}
	var expenseDate *string
	if result.ExpenseDate != nil {
		value := result.ExpenseDate.Format("2006-01-02")
		expenseDate = &value
	}
	var amount *appdto.MoneyAmount
	if result.Amount != nil && result.Currency != nil {
		amount = &appdto.MoneyAmount{Amount: round2(*result.Amount), Currency: *result.Currency}
	}
	var taxAmount *appdto.MoneyAmount
	if result.TaxAmount != nil && result.Currency != nil {
		taxAmount = &appdto.MoneyAmount{Amount: round2(*result.TaxAmount), Currency: *result.Currency}
	}
	rawText := result.RawText
	if !includeRaw {
		rawText = nil
	}
	return &appdto.ReceiptOCRResult{
		Merchant:        result.Merchant,
		ExpenseDate:     expenseDate,
		Amount:          amount,
		TaxAmount:       taxAmount,
		Category:        result.Category,
		SuggestedTitle:  result.SuggestedTitle,
		Confidence:      result.Confidence,
		FieldConfidence: result.FieldConfidence,
		Warnings:        result.Warnings,
		RawText:         rawText,
	}
}

func withExpenseReceipts(expense appdto.TripExpense, receipts []appdto.ExpenseReceiptSummary) appdto.TripExpense {
	expense.Receipts = receipts
	expense.ReceiptCount = len(receipts)
	expense.HasReceipt = len(receipts) > 0
	if len(receipts) > 0 {
		status := receipts[0].Status
		expense.LatestReceiptStatus = &status
	}
	return expense
}

func receiptActivityMetadata(receipt *entity.TripExpenseReceipt, result *entity.ReceiptOCRResult) map[string]any {
	filename := receipt.OriginalFilename
	if len([]rune(filename)) > 80 {
		filename = string([]rune(filename)[:80])
	}
	metadata := map[string]any{
		"receiptId":        receipt.ID.String(),
		"originalFilename": filename,
	}
	if receipt.ExpenseID != nil {
		metadata["expenseId"] = receipt.ExpenseID.String()
	}
	if result != nil {
		metadata["ocrConfidence"] = string(result.Confidence)
		if result.Category != nil {
			metadata["category"] = string(*result.Category)
		}
		if result.Amount != nil {
			metadata["amount"] = *result.Amount
		}
		if result.Currency != nil {
			metadata["currency"] = *result.Currency
		}
	}
	return metadata
}

func receiptExpenseActivityMetadata(receipt *entity.TripExpenseReceipt, expense *entity.TripExpense) map[string]any {
	metadata := receiptActivityMetadata(receipt, nil)
	metadata["expenseId"] = expense.ID.String()
	metadata["expenseTitle"] = expense.Title
	metadata["amount"] = expense.Amount
	metadata["currency"] = expense.Currency
	metadata["category"] = string(expense.Category)
	return metadata
}

func canMutateReceipt(access TripAccess, actorID, createdBy uuid.UUID) bool {
	return access.CanEdit() || actorID == createdBy
}

func detectReceiptContentType(header []byte, declared string) string {
	if len(header) >= 12 && string(header[0:4]) == "RIFF" && string(header[8:12]) == "WEBP" {
		return "image/webp"
	}
	if len(header) >= 4 && string(header[:4]) == "%PDF" {
		return "application/pdf"
	}
	return strings.ToLower(strings.TrimSpace(strings.Split(http.DetectContentType(header), ";")[0]))
}

func validReceiptExtension(filename, contentType string, allowedExtensions []string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	if len(allowedExtensions) == 0 {
		allowedExtensions = receipts.DefaultConfig().AllowedExtensions
	}
	extensionAllowed := false
	for _, allowed := range allowedExtensions {
		if ext == strings.ToLower(strings.TrimSpace(allowed)) {
			extensionAllowed = true
			break
		}
	}
	if !extensionAllowed {
		return false
	}
	switch contentType {
	case "image/jpeg":
		return ext == ".jpg" || ext == ".jpeg"
	case "image/png":
		return ext == ".png"
	case "image/webp":
		return ext == ".webp"
	case "application/pdf":
		return ext == ".pdf"
	default:
		exts, _ := mime.ExtensionsByType(contentType)
		for _, item := range exts {
			if item == ext {
				return true
			}
		}
		return false
	}
}

func cleanReceiptFilename(filename string) string {
	filename = strings.ReplaceAll(filename, "\\", "/")
	filename = strings.TrimSpace(filepath.Base(filename))
	filename = strings.Map(func(char rune) rune {
		if unicode.IsControl(char) {
			return -1
		}
		return char
	}, filename)
	if runes := []rune(filename); len(runes) > 255 {
		filename = string(runes[len(runes)-255:])
	}
	return filename
}
