package service

import (
	"context"
	"strings"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
)

// CopilotPreferredLanguage returns only the profile's language preference.
// It intentionally does not expose the user profile or preferences to Copilot.
// A failed profile lookup is fail-soft because the browser's language remains a
// safe fallback for an advisory response.
func (s *Service) CopilotPreferredLanguage(ctx context.Context, fallback string) string {
	fallback = normalizeCopilotLanguage(fallback)
	if !s.userContextEnabled || s.userContextProvider == nil {
		return fallback
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil || strings.TrimSpace(user.AccessToken) == "" {
		return fallback
	}
	userContext, err := s.userContextProvider.GetUserContext(ctx, user.AccessToken)
	if err != nil || userContext == nil || userContext.Profile == nil {
		return fallback
	}
	return normalizeCopilotLanguage(userContext.Profile.PreferredLanguage)
}

func normalizeCopilotLanguage(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) >= 2 {
		value = value[:2]
	}
	switch value {
	case "es", "fr", "uk":
		return value
	default:
		return "en"
	}
}
