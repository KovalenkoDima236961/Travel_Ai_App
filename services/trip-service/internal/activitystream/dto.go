package activitystream

import "github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"

type ActivityCreatedPayload struct {
	Event activity.EventDTO `json:"event"`
}

type heartbeatPayload struct {
	Timestamp string `json:"ts"`
}
