package response

import (
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
)

type DigestItem struct {
	ID            string         `json:"id"`
	TripID        *string        `json:"tripId"`
	Category      string         `json:"category"`
	Priority      string         `json:"priority"`
	DigestKey     string         `json:"digestKey"`
	Title         string         `json:"title"`
	Message       string         `json:"message"`
	Metadata      map[string]any `json:"metadata"`
	EventCount    int            `json:"eventCount"`
	LatestEventAt string         `json:"latestEventAt"`
}
type DigestBatch struct {
	ID               string       `json:"id"`
	Channel          string       `json:"channel"`
	Mode             string       `json:"mode"`
	Status           string       `json:"status"`
	ScheduledFor     string       `json:"scheduledFor"`
	SentAt           *string      `json:"sentAt"`
	Attempts         int          `json:"attempts"`
	NextAttemptAt    *string      `json:"nextAttemptAt"`
	ErrorCode        *string      `json:"errorCode"`
	ErrorMessageSafe *string      `json:"errorMessageSafe"`
	EventCount       int          `json:"eventCount"`
	Items            []DigestItem `json:"items"`
}
type DigestList struct {
	Items []DigestBatch `json:"items"`
}

func NewDigestBatch(batch entity.NotificationDigestBatch) DigestBatch {
	items := make([]DigestItem, 0, len(batch.Items))
	count := 0
	for _, item := range batch.Items {
		metadata := item.Metadata
		if metadata == nil {
			metadata = map[string]any{}
		}
		items = append(items, DigestItem{ID: item.ID.String(), TripID: uuidPtrString(item.TripID), Category: item.Category, Priority: item.Priority, DigestKey: item.DigestKey, Title: item.Title, Message: item.Message, Metadata: metadata, EventCount: item.EventCount, LatestEventAt: item.LatestEventAt.UTC().Format(time.RFC3339Nano)})
		count += item.EventCount
	}
	return DigestBatch{
		ID: batch.ID.String(), Channel: batch.Channel, Mode: batch.Mode, Status: batch.Status,
		ScheduledFor: batch.ScheduledFor.UTC().Format(time.RFC3339Nano), SentAt: timePtrString(batch.SentAt),
		Attempts: batch.Attempts, NextAttemptAt: timePtrString(batch.NextAttemptAt),
		ErrorCode: batch.ErrorCode, ErrorMessageSafe: batch.ErrorMessageSafe,
		EventCount: count, Items: items,
	}
}
func NewDigestList(batches []entity.NotificationDigestBatch) DigestList {
	items := make([]DigestBatch, 0, len(batches))
	for _, batch := range batches {
		items = append(items, NewDigestBatch(batch))
	}
	return DigestList{Items: items}
}
