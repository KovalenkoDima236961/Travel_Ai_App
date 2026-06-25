package users

import "github.com/google/uuid"

// UserProfile is a resolved recipient. DisplayName may be empty (Auth Service
// owns email, not profile display names in v1); callers fall back to a neutral
// greeting when it is blank.
type UserProfile struct {
	UserID      uuid.UUID
	Email       string
	DisplayName string
}

// --- wire shapes (JSON sent to / received from Auth Service) ---

type batchRequest struct {
	UserIDs []string `json:"userIds"`
}

type batchResponse struct {
	Items []userItem `json:"items"`
}

type userItem struct {
	UserID      string `json:"userId"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}
