package preferences

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
)

type fakePreferenceRepo struct {
	rows     []entity.NotificationPreference
	upserts  []PreferenceInput
	upsertID uuid.UUID
}

func (f *fakePreferenceRepo) ListNotificationPreferencesByUsers(_ context.Context, userIDs []uuid.UUID) ([]entity.NotificationPreference, error) {
	wanted := make(map[uuid.UUID]struct{}, len(userIDs))
	for _, id := range userIDs {
		wanted[id] = struct{}{}
	}
	out := make([]entity.NotificationPreference, 0)
	for _, row := range f.rows {
		if _, ok := wanted[row.UserID]; ok {
			out = append(out, row)
		}
	}
	return out, nil
}

func (f *fakePreferenceRepo) UpsertNotificationPreferencesBatch(_ context.Context, userID uuid.UUID, items []PreferenceInput) error {
	f.upsertID = userID
	f.upserts = append([]PreferenceInput(nil), items...)
	for _, item := range items {
		replaced := false
		for i := range f.rows {
			if f.rows[i].UserID == userID && f.rows[i].Channel == item.Channel && f.rows[i].Category == item.Category {
				f.rows[i].Enabled = item.Enabled
				replaced = true
				break
			}
		}
		if !replaced {
			f.rows = append(f.rows, entity.NotificationPreference{
				ID:       uuid.New(),
				UserID:   userID,
				Channel:  item.Channel,
				Category: item.Category,
				Enabled:  item.Enabled,
			})
		}
	}
	return nil
}

func TestGetPreferencesReturnsFullDefaultMatrix(t *testing.T) {
	userID := uuid.New()
	svc := New(&fakePreferenceRepo{}, nil)

	result, err := svc.GetPreferences(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 18 {
		t.Fatalf("expected 18 preference items, got %d", len(result.Items))
	}
	if !findPreference(t, result.Items, ChannelInApp, CategoryTripUpdates).Enabled {
		t.Fatal("expected in-app trip updates enabled by default")
	}
	if findPreference(t, result.Items, ChannelEmail, CategoryTripUpdates).Enabled {
		t.Fatal("expected email trip updates disabled by default")
	}
	if !findPreference(t, result.Items, ChannelEmail, CategoryCollaboration).Enabled {
		t.Fatal("expected email collaboration enabled by default")
	}
	if !findPreference(t, result.Items, ChannelPush, CategoryTripUpdates).Enabled {
		t.Fatal("expected push trip updates enabled by default")
	}
	if !findPreference(t, result.Items, ChannelInApp, CategoryPreTripReminders).Enabled {
		t.Fatal("expected in-app pre-trip reminders enabled by default")
	}
	if findPreference(t, result.Items, ChannelEmail, CategoryPreTripReminders).Enabled {
		t.Fatal("expected email pre-trip reminders disabled by default")
	}
	if !findPreference(t, result.Items, ChannelPush, CategoryChecklistReminders).Enabled {
		t.Fatal("expected push checklist reminders enabled by default")
	}
}

func TestGetPreferencesAppliesStoredOverrides(t *testing.T) {
	userID := uuid.New()
	svc := New(&fakePreferenceRepo{rows: []entity.NotificationPreference{
		{UserID: userID, Channel: ChannelInApp, Category: CategoryComments, Enabled: false},
		{UserID: userID, Channel: ChannelEmail, Category: CategoryTripUpdates, Enabled: true},
	}}, nil)

	result, err := svc.GetPreferences(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if findPreference(t, result.Items, ChannelInApp, CategoryComments).Enabled {
		t.Fatal("expected stored in-app comments override to disable comments")
	}
	if !findPreference(t, result.Items, ChannelEmail, CategoryTripUpdates).Enabled {
		t.Fatal("expected stored email trip updates override to enable trip updates")
	}
}

func TestUpdatePreferencesUpsertsAndReturnsEffectiveMatrix(t *testing.T) {
	userID := uuid.New()
	repo := &fakePreferenceRepo{}
	svc := New(repo, nil)

	result, err := svc.UpdatePreferences(context.Background(), userID, []PreferenceInput{
		{Channel: ChannelEmail, Category: CategoryComments, Enabled: false},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.upsertID != userID || len(repo.upserts) != 1 {
		t.Fatalf("expected one upsert for user, got user=%s upserts=%+v", repo.upsertID, repo.upserts)
	}
	if findPreference(t, result.Items, ChannelEmail, CategoryComments).Enabled {
		t.Fatal("expected returned matrix to include saved disabled override")
	}
}

func TestUpdatePreferencesValidation(t *testing.T) {
	svc := New(&fakePreferenceRepo{}, nil)
	userID := uuid.New()

	cases := map[string][]PreferenceInput{
		"empty":            {},
		"invalid channel":  {{Channel: "sms", Category: CategoryComments, Enabled: true}},
		"invalid category": {{Channel: ChannelEmail, Category: "billing", Enabled: true}},
		"duplicate": {
			{Channel: ChannelEmail, Category: CategoryComments, Enabled: true},
			{Channel: ChannelEmail, Category: CategoryComments, Enabled: false},
		},
	}

	for name, items := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.UpdatePreferences(context.Background(), userID, items)
			var invalid *apperrs.InvalidInputError
			if !errors.As(err, &invalid) {
				t.Fatalf("expected InvalidInputError, got %v", err)
			}
		})
	}
}

func TestCategoryForNotificationType(t *testing.T) {
	cases := map[string]string{
		notifications.TypeCollaborationInvited:     CategoryCollaboration,
		notifications.TypeCollaborationAccepted:    CategoryCollaboration,
		notifications.TypeCommentCreated:           CategoryComments,
		notifications.TypeCollaboratorRoleChange:   CategoryRoleChanges,
		notifications.TypeCollaboratorRemoved:      CategoryRoleChanges,
		notifications.TypeItineraryUpdated:         CategoryTripUpdates,
		notifications.TypeItineraryGenerated:       CategoryTripUpdates,
		notifications.TypeDayRegenerated:           CategoryTripUpdates,
		notifications.TypeItemRegenerated:          CategoryTripUpdates,
		notifications.TypeVersionRestored:          CategoryTripUpdates,
		notifications.TypeGenerationJobFailed:      CategoryTripUpdates,
		notifications.TypeBudgetOptimizationReady:  CategoryTripUpdates,
		notifications.TypeBudgetOptimizationFailed: CategoryTripUpdates,
	}

	for typ, expected := range cases {
		t.Run(typ, func(t *testing.T) {
			got, ok := CategoryForNotificationType(typ)
			if !ok || got != expected {
				t.Fatalf("expected %q -> %q, got %q ok=%v", typ, expected, got, ok)
			}
		})
	}

	if _, ok := CategoryForNotificationType("future_type"); ok {
		t.Fatal("expected unknown type to return ok=false")
	}
}

func TestEffectiveSetUnknownTypeDefaults(t *testing.T) {
	userID := uuid.New()
	set := BuildEffectiveSet([]uuid.UUID{userID}, nil)

	if !set.AllowInApp(userID, "future_type") {
		t.Fatal("expected unknown types allowed in-app")
	}
	if set.AllowEmail(userID, "future_type") {
		t.Fatal("expected unknown types blocked for email")
	}
	if set.AllowPush(userID, "future_type") {
		t.Fatal("expected unknown types blocked for push")
	}
}

func findPreference(t *testing.T, items []PreferenceItem, channel, category string) PreferenceItem {
	t.Helper()
	for _, item := range items {
		if item.Channel == channel && item.Category == category {
			return item
		}
	}
	t.Fatalf("preference %s/%s not found in %+v", channel, category, items)
	return PreferenceItem{}
}
