package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/notifications"
)

func (s *Service) RecordGenerationJobFailed(
	ctx context.Context,
	tripID, requesterID, jobID uuid.UUID,
	jobType entity.GenerationJobType,
	errorCode, _ string,
) {
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &requesterID,
		EventType:   activity.EventGenerationJobFailed,
		EntityType:  activityEntityType(activity.EntityItinerary),
		EntityID:    activityEntityID(tripID),
		Metadata: map[string]any{
			"jobId":     jobID.String(),
			"jobType":   string(jobType),
			"errorCode": errorCode,
		},
	})

	trip := tripID
	job := jobID
	s.sendNotifications(ctx, []notifications.NotificationCreateInput{{
		UserID:     requesterID,
		TripID:     &trip,
		Type:       notifications.TypeGenerationJobFailed,
		Title:      "Generation failed",
		Message:    "Your itinerary generation could not be completed. Open the trip to retry.",
		Priority:   notifications.PriorityUrgent,
		Category:   "ai_generation",
		DigestKey:  "trip:" + tripID.String() + ":ai_generation",
		DedupeKey:  "generation_job:" + jobID.String() + ":recipient:" + requesterID.String(),
		EntityType: activityEntityType(notifications.EntityItinerary),
		EntityID:   &job,
		Metadata: map[string]any{
			"tripId":    tripID.String(),
			"jobId":     jobID.String(),
			"jobType":   string(jobType),
			"errorCode": errorCode,
		},
	}})
}
