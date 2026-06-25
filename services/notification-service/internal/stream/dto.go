package stream

const (
	EventNotificationCreated  = "notification.created"
	EventNotificationRead     = "notification.read"
	EventNotificationsReadAll = "notifications.read_all"
	EventHeartbeat            = "heartbeat"
)

// StreamEvent is one Server-Sent Event queued for a connected client.
type StreamEvent struct {
	Name string
	Data any
}
