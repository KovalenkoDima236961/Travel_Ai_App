package deliverypolicy

import "time"

const (
	DecisionSendInstant     = "send_instant"
	DecisionCreateInAppOnly = "create_in_app_only"
	DecisionDigest          = "digest"
	DecisionMute            = "mute"
	DecisionDelayQuietHours = "delay_until_quiet_hours_end"
	DecisionDropDuplicate   = "drop_duplicate"
)

type Decision struct {
	Decision     string
	Mode         string
	Reason       string
	ScheduledFor *time.Time
}
