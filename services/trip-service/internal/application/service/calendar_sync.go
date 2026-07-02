package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/calendarclient"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/calendarsync"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	tripobs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/observability"
)

const googleCalendarProvider = "google"

func (s *Service) GetGoogleCalendarSyncStatus(ctx context.Context, tripID uuid.UUID) (*appdto.TripCalendarSyncStatus, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}

	eventCount, lastSyncedAt, syncedRevision, err := s.repo.GetTripCalendarSyncStatus(ctx, tripID, user.ID, googleCalendarProvider)
	if err != nil {
		return nil, err
	}

	connected := false
	var email *string
	if s.calendarSyncProvider != nil {
		if accessToken, ok := auth.AccessTokenFromContext(ctx); ok {
			if status, err := s.calendarSyncProvider.GetGoogleCalendarStatus(ctx, accessToken); err == nil && status != nil {
				connected = status.Connected
				email = status.ProviderAccountEmail
			} else if err != nil {
				s.log.Warn("calendar connection status lookup failed", zap.Error(err))
			}
		}
	}

	return &appdto.TripCalendarSyncStatus{
		Provider:                 googleCalendarProvider,
		Connected:                connected,
		ProviderAccountEmail:     email,
		Synced:                   eventCount > 0,
		LastSyncedAt:             lastSyncedAt,
		SyncedItineraryRevision:  syncedRevision,
		CurrentItineraryRevision: trip.ItineraryRevision,
		OutOfDate:                eventCount > 0 && syncedRevision < trip.ItineraryRevision,
		EventCount:               eventCount,
	}, nil
}

func (s *Service) SyncTripToGoogleCalendar(ctx context.Context, tripID uuid.UUID, expectedRevision *int) (*appdto.TripCalendarSyncResult, error) {
	if !s.calendarSyncEnabled || s.calendarSyncProvider == nil {
		return nil, apperrs.NewDependencyError("calendar sync is not configured")
	}
	if expectedRevision == nil {
		return nil, apperrs.ErrExpectedItineraryRevisionRequired
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	trip, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	if *expectedRevision != trip.ItineraryRevision {
		return nil, apperrs.NewItineraryConflict(trip.ItineraryRevision)
	}
	if trip.StartDate == nil {
		return nil, apperrs.NewInvalidInput("trip startDate is required for calendar sync")
	}
	if len(trip.Itinerary) == 0 {
		return nil, apperrs.NewInvalidInput("trip itinerary is required for calendar sync")
	}

	built, err := calendarsync.BuildEvents(calendarsync.BuildInput{
		Trip:     trip,
		TripURL:  s.tripURL(tripID),
		TimeZone: s.calendarSyncDefaultTimeZone,
	})
	if err != nil {
		return nil, apperrs.NewInvalidInput("%s", err.Error())
	}

	existing, err := s.repo.ListTripCalendarSyncsByTripUserProvider(ctx, tripID, user.ID, googleCalendarProvider)
	if err != nil {
		return nil, err
	}
	existingByKey := make(map[string]entity.TripCalendarSync, len(existing))
	for _, row := range existing {
		existingByKey[row.SyncKey] = row
	}

	currentKeys := make(map[string]struct{}, len(built.Items))
	for i := range built.Items {
		item := &built.Items[i]
		currentKeys[item.SyncKey] = struct{}{}
		if row, ok := existingByKey[item.SyncKey]; ok {
			item.ExistingCalendarID = row.ExternalCalendarID
			item.ExistingEventID = row.ExternalEventID
		}
	}

	deleteItems := make([]calendarclient.DeleteItem, 0)
	deleteKeys := make([]string, 0)
	for _, row := range existing {
		if _, ok := currentKeys[row.SyncKey]; ok {
			continue
		}
		deleteItems = append(deleteItems, calendarclient.DeleteItem{
			CalendarID: row.ExternalCalendarID,
			EventID:    row.ExternalEventID,
		})
		deleteKeys = append(deleteKeys, row.SyncKey)
	}

	deleted, deleteFailed, err := s.deleteGoogleEvents(ctx, user.ID, tripID, deleteItems, deleteKeys)
	if err != nil {
		tripobs.RecordCalendarSync(googleCalendarProvider, "delete_failed")
		return nil, err
	}

	result := &appdto.TripCalendarSyncResult{
		Provider:          googleCalendarProvider,
		Status:            "synced",
		Deleted:           deleted,
		Failed:            deleteFailed,
		Skipped:           built.Skipped,
		ItineraryRevision: trip.ItineraryRevision,
	}
	if len(built.Items) == 0 {
		result.Status = "no_timed_items"
		result.LastSyncedAt = nowPtr()
		tripobs.RecordCalendarSync(googleCalendarProvider, result.Status)
		return result, nil
	}

	syncResult, err := s.calendarSyncProvider.SyncGoogleCalendarEvents(ctx, calendarclient.SyncRequest{
		UserID:    user.ID,
		TripID:    tripID,
		TripTitle: tripDestination(trip),
		TripURL:   s.tripURL(tripID),
		TimeZone:  s.calendarSyncDefaultTimeZone,
		Items:     built.Items,
	})
	if err != nil {
		tripobs.RecordCalendarSync(googleCalendarProvider, "sync_failed")
		return nil, calendarClientError(err)
	}

	lastSynced := time.Now().UTC()
	result.LastSyncedAt = &lastSynced
	for _, item := range syncResult.Items {
		switch item.Status {
		case "created":
			result.Created++
		case "updated":
			result.Updated++
		default:
			result.Failed++
			continue
		}
		if strings.TrimSpace(item.EventID) == "" {
			result.Failed++
			continue
		}
		link := strings.TrimSpace(item.HtmlLink)
		var linkPtr *string
		if link != "" {
			linkPtr = &link
		}
		_, err := s.repo.UpsertTripCalendarSync(ctx, &entity.TripCalendarSync{
			ID:                 uuid.New(),
			TripID:             tripID,
			UserID:             user.ID,
			Provider:           googleCalendarProvider,
			ExternalCalendarID: defaultString(item.CalendarID, "primary"),
			ExternalEventID:    item.EventID,
			ExternalEventLink:  linkPtr,
			DayNumber:          item.DayNumber,
			ItemIndex:          item.ItemIndex,
			ItineraryRevision:  trip.ItineraryRevision,
			SyncKey:            item.SyncKey,
		})
		if err != nil {
			tripobs.RecordCalendarSync(googleCalendarProvider, "store_failed")
			return nil, err
		}
	}

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventCalendarSynced,
		EntityType:  activityEntityType(activity.EntityCalendarSync),
		Metadata: map[string]any{
			"provider":          googleCalendarProvider,
			"created":           result.Created,
			"updated":           result.Updated,
			"deleted":           result.Deleted,
			"itineraryRevision": trip.ItineraryRevision,
		},
	})

	tripobs.RecordCalendarSync(googleCalendarProvider, result.Status)
	return result, nil
}

func (s *Service) RemoveTripGoogleCalendarSync(ctx context.Context, tripID uuid.UUID) (*appdto.TripCalendarDeleteResult, error) {
	if !s.calendarSyncEnabled || s.calendarSyncProvider == nil {
		return nil, apperrs.NewDependencyError("calendar sync is not configured")
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListTripCalendarSyncsByTripUserProvider(ctx, tripID, user.ID, googleCalendarProvider)
	if err != nil {
		return nil, err
	}
	items := make([]calendarclient.DeleteItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, calendarclient.DeleteItem{CalendarID: row.ExternalCalendarID, EventID: row.ExternalEventID})
	}
	deleted, failed, err := s.deleteGoogleEvents(ctx, user.ID, tripID, items, nil)
	if err != nil {
		tripobs.RecordCalendarSync(googleCalendarProvider, "delete_failed")
		return nil, err
	}
	if err := s.repo.MarkAllTripCalendarSyncsDeleted(ctx, tripID, user.ID, googleCalendarProvider); err != nil {
		tripobs.RecordCalendarSync(googleCalendarProvider, "delete_store_failed")
		return nil, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventCalendarSyncRemoved,
		EntityType:  activityEntityType(activity.EntityCalendarSync),
		Metadata: map[string]any{
			"provider": googleCalendarProvider,
			"deleted":  deleted,
			"failed":   failed,
		},
	})
	result := "delete_success"
	if failed > 0 {
		result = "delete_partial"
	}
	tripobs.RecordCalendarSync(googleCalendarProvider, result)
	return &appdto.TripCalendarDeleteResult{Provider: googleCalendarProvider, Deleted: deleted, Failed: failed}, nil
}

func (s *Service) deleteGoogleEvents(ctx context.Context, userID, tripID uuid.UUID, items []calendarclient.DeleteItem, keys []string) (int, int, error) {
	if len(items) == 0 {
		return 0, 0, nil
	}
	result, err := s.calendarSyncProvider.DeleteGoogleCalendarEvents(ctx, calendarclient.DeleteRequest{
		UserID: userID,
		Events: items,
	})
	if err != nil {
		return 0, 0, calendarClientError(err)
	}
	if keys != nil {
		for _, key := range keys {
			if err := s.repo.MarkTripCalendarSyncDeleted(ctx, tripID, userID, googleCalendarProvider, key); err != nil {
				return result.Deleted, result.Failed, err
			}
		}
	}
	return result.Deleted, result.Failed, nil
}

func (s *Service) tripURL(tripID uuid.UUID) string {
	baseURL := strings.TrimRight(strings.TrimSpace(s.calendarSyncPublicWebBaseURL), "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(strings.TrimSpace(s.publicWebBaseURL), "/")
	}
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	return fmt.Sprintf("%s/trips/%s", baseURL, tripID.String())
}

func calendarClientError(err error) error {
	var clientErr *calendarclient.Error
	if errors.As(err, &clientErr) {
		switch clientErr.Code {
		case "calendar_not_connected":
			return apperrs.NewInvalidInput("calendar_not_connected")
		case "calendar_reauth_required":
			return apperrs.NewInvalidInput("calendar_reauth_required")
		default:
			return apperrs.NewDependencyError("%s", clientErr.Code)
		}
	}
	return apperrs.NewDependencyError("calendar sync failed")
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func nowPtr() *time.Time {
	t := time.Now().UTC()
	return &t
}
