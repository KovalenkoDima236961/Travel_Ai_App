package dto

import "time"

type WorkspaceCostAnalyticsInput struct {
	Currency        string
	From            *time.Time
	To              *time.Time
	IncludeArchived bool
}
