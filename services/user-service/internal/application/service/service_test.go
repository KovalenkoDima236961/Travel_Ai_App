package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/domain/errs"
)

type mockRepo struct {
	getProfileResult *entity.Profile
	getProfileErr    error
	getProfileUserID uuid.UUID

	createDefaultProfileResult *entity.Profile
	createDefaultProfileUserID uuid.UUID

	upsertProfileInput *entity.Profile

	getPreferencesResult *entity.Preferences
	getPreferencesErr    error
	getPreferencesUserID uuid.UUID

	createDefaultPreferencesResult *entity.Preferences
	createDefaultPreferencesUserID uuid.UUID

	upsertPreferencesInput *entity.Preferences
}

func (m *mockRepo) GetProfileByUserID(_ context.Context, userID uuid.UUID) (*entity.Profile, error) {
	m.getProfileUserID = userID
	if m.getProfileErr != nil {
		return nil, m.getProfileErr
	}
	return m.getProfileResult, nil
}

func (m *mockRepo) CreateDefaultProfile(_ context.Context, userID uuid.UUID) (*entity.Profile, error) {
	m.createDefaultProfileUserID = userID
	if m.createDefaultProfileResult != nil {
		return m.createDefaultProfileResult, nil
	}
	return defaultProfile(userID), nil
}

func (m *mockRepo) UpsertProfile(_ context.Context, profile *entity.Profile) (*entity.Profile, error) {
	m.upsertProfileInput = profile
	out := *profile
	out.CreatedAt = time.Now().UTC()
	out.UpdatedAt = out.CreatedAt
	return &out, nil
}

func (m *mockRepo) GetPreferencesByUserID(_ context.Context, userID uuid.UUID) (*entity.Preferences, error) {
	m.getPreferencesUserID = userID
	if m.getPreferencesErr != nil {
		return nil, m.getPreferencesErr
	}
	return m.getPreferencesResult, nil
}

func (m *mockRepo) CreateDefaultPreferences(_ context.Context, userID uuid.UUID) (*entity.Preferences, error) {
	m.createDefaultPreferencesUserID = userID
	if m.createDefaultPreferencesResult != nil {
		return m.createDefaultPreferencesResult, nil
	}
	return defaultPreferences(userID), nil
}

func (m *mockRepo) UpsertPreferences(_ context.Context, preferences *entity.Preferences) (*entity.Preferences, error) {
	m.upsertPreferencesInput = preferences
	out := *preferences
	out.CreatedAt = time.Now().UTC()
	out.UpdatedAt = out.CreatedAt
	return &out, nil
}

func TestGetProfileCreatesDefaultWhenMissing(t *testing.T) {
	userID := testUserID()
	repo := &mockRepo{getProfileErr: domainerrs.ErrNotFound}
	svc := New(repo, zap.NewNop())

	got, err := svc.GetProfile(authContext(userID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.UserID != userID {
		t.Fatalf("expected profile user %s, got %s", userID, got.UserID)
	}
	if repo.getProfileUserID != userID || repo.createDefaultProfileUserID != userID {
		t.Fatalf("expected repository calls scoped to %s, got get=%s create=%s", userID, repo.getProfileUserID, repo.createDefaultProfileUserID)
	}
	if got.PreferredCurrency != defaultCurrency || got.PreferredLanguage != defaultLanguage {
		t.Fatalf("unexpected defaults: %+v", got)
	}
}

func TestUpdateProfileUsesAuthenticatedUserID(t *testing.T) {
	userID := testUserID()
	repo := &mockRepo{}
	svc := New(repo, zap.NewNop())

	_, err := svc.UpdateProfile(authContext(userID), validProfileInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.upsertProfileInput == nil || repo.upsertProfileInput.UserID != userID {
		t.Fatalf("expected upsert for authenticated user %s, got %+v", userID, repo.upsertProfileInput)
	}
}

func TestUpdateProfileRejectsInvalidCurrency(t *testing.T) {
	repo := &mockRepo{}
	svc := New(repo, zap.NewNop())

	in := validProfileInput()
	in.PreferredCurrency = "eur"

	_, err := svc.UpdateProfile(authContext(testUserID()), in)
	assertInvalidInput(t, err)
	if repo.upsertProfileInput != nil {
		t.Fatal("repository must not be called for invalid currency")
	}
}

func TestUpdateProfileRejectsTooLongDisplayName(t *testing.T) {
	repo := &mockRepo{}
	svc := New(repo, zap.NewNop())

	in := validProfileInput()
	in.DisplayName = strings.Repeat("a", 101)

	_, err := svc.UpdateProfile(authContext(testUserID()), in)
	assertInvalidInput(t, err)
	if repo.upsertProfileInput != nil {
		t.Fatal("repository must not be called for invalid display name")
	}
}

func TestGetPreferencesCreatesDefaultWhenMissing(t *testing.T) {
	userID := testUserID()
	repo := &mockRepo{getPreferencesErr: domainerrs.ErrNotFound}
	svc := New(repo, zap.NewNop())

	got, err := svc.GetPreferences(authContext(userID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.UserID != userID {
		t.Fatalf("expected preferences user %s, got %s", userID, got.UserID)
	}
	if repo.getPreferencesUserID != userID || repo.createDefaultPreferencesUserID != userID {
		t.Fatalf("expected repository calls scoped to %s, got get=%s create=%s", userID, repo.getPreferencesUserID, repo.createDefaultPreferencesUserID)
	}
	if got.Pace != defaultPace || len(got.TravelStyles) != 0 {
		t.Fatalf("unexpected defaults: %+v", got)
	}
}

func TestPatchPreferencesMergesProvidedFieldsAndSanitizesArrays(t *testing.T) {
	userID := testUserID()
	maxWalking := 8.0
	styles := []string{" budget ", "", "food", "budget", "hidden_gems"}
	avoid := []string{" nightclubs ", "nightclubs", ""}
	repo := &mockRepo{
		getPreferencesResult: &entity.Preferences{
			UserID:             userID,
			TravelStyles:       []string{"existing"},
			Pace:               "relaxed",
			FoodPreferences:    []string{"local"},
			PreferredTransport: []string{"train"},
		},
	}
	svc := New(repo, zap.NewNop())

	got, err := svc.PatchPreferences(authContext(userID), appdto.PatchPreferencesInput{
		TravelStyles:       &styles,
		MaxWalkingKmPerDay: &appdto.OptionalFloat64{Value: &maxWalking},
		Avoid:              &avoid,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.getPreferencesUserID != userID || repo.upsertPreferencesInput.UserID != userID {
		t.Fatalf("expected repository calls scoped to %s", userID)
	}
	assertStrings(t, got.TravelStyles, []string{"budget", "food", "hidden_gems"})
	assertStrings(t, got.Avoid, []string{"nightclubs"})
	assertStrings(t, got.FoodPreferences, []string{"local"})
	assertStrings(t, got.PreferredTransport, []string{"train"})
	if got.Pace != "relaxed" {
		t.Fatalf("expected omitted pace to remain relaxed, got %q", got.Pace)
	}
	if got.MaxWalkingKmPerDay == nil || *got.MaxWalkingKmPerDay != 8 {
		t.Fatalf("expected max walking 8, got %v", got.MaxWalkingKmPerDay)
	}
}

func TestPatchPreferencesRejectsInvalidPace(t *testing.T) {
	userID := testUserID()
	pace := "packed"
	repo := &mockRepo{getPreferencesResult: defaultPreferences(userID)}
	svc := New(repo, zap.NewNop())

	_, err := svc.PatchPreferences(authContext(userID), appdto.PatchPreferencesInput{Pace: &pace})
	assertInvalidInput(t, err)
	if repo.upsertPreferencesInput != nil {
		t.Fatal("repository must not be called for invalid pace")
	}
}

func TestPatchPreferencesClearsMaxWalkingWhenExplicitlyNull(t *testing.T) {
	userID := testUserID()
	existingMaxWalking := 8.0
	styles := []string{"food"}
	repo := &mockRepo{
		getPreferencesResult: &entity.Preferences{
			UserID:             userID,
			TravelStyles:       []string{"budget"},
			Pace:               "balanced",
			MaxWalkingKmPerDay: &existingMaxWalking,
		},
	}
	svc := New(repo, zap.NewNop())

	got, err := svc.PatchPreferences(authContext(userID), appdto.PatchPreferencesInput{
		TravelStyles:       &styles,
		MaxWalkingKmPerDay: &appdto.OptionalFloat64{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.MaxWalkingKmPerDay != nil {
		t.Fatalf("expected max walking to be cleared, got %v", got.MaxWalkingKmPerDay)
	}
	assertStrings(t, got.TravelStyles, []string{"food"})
}

func TestPatchPreferencesRejectsMaxWalkingOver50(t *testing.T) {
	userID := testUserID()
	maxWalking := 51.0
	repo := &mockRepo{getPreferencesResult: defaultPreferences(userID)}
	svc := New(repo, zap.NewNop())

	_, err := svc.PatchPreferences(authContext(userID), appdto.PatchPreferencesInput{
		MaxWalkingKmPerDay: &appdto.OptionalFloat64{Value: &maxWalking},
	})
	assertInvalidInput(t, err)
	if repo.upsertPreferencesInput != nil {
		t.Fatal("repository must not be called for invalid max walking distance")
	}
}

func TestPatchPreferencesUsesAuthenticatedUserIDWhenCreatingMissingPreferences(t *testing.T) {
	userA := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userB := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	pace := "intensive"
	repo := &mockRepo{getPreferencesErr: domainerrs.ErrNotFound}
	svc := New(repo, zap.NewNop())

	if _, err := svc.PatchPreferences(authContext(userA), appdto.PatchPreferencesInput{Pace: &pace}); err != nil {
		t.Fatalf("unexpected user A error: %v", err)
	}
	if repo.upsertPreferencesInput.UserID != userA {
		t.Fatalf("expected user A upsert, got %s", repo.upsertPreferencesInput.UserID)
	}

	if _, err := svc.PatchPreferences(authContext(userB), appdto.PatchPreferencesInput{Pace: &pace}); err != nil {
		t.Fatalf("unexpected user B error: %v", err)
	}
	if repo.upsertPreferencesInput.UserID != userB {
		t.Fatalf("expected user B upsert, got %s", repo.upsertPreferencesInput.UserID)
	}
}

func validProfileInput() appdto.UpdateProfileInput {
	return appdto.UpdateProfileInput{
		DisplayName:       "Test Traveler",
		HomeCity:          "Bratislava",
		HomeCountry:       "Slovakia",
		PreferredCurrency: "EUR",
		PreferredLanguage: "en",
	}
}

func defaultProfile(userID uuid.UUID) *entity.Profile {
	now := time.Now().UTC()
	return &entity.Profile{
		UserID:            userID,
		PreferredCurrency: defaultCurrency,
		PreferredLanguage: defaultLanguage,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func authContext(userID uuid.UUID) context.Context {
	return auth.WithUser(context.Background(), auth.AuthenticatedUser{
		ID:    userID,
		Email: "traveler@example.com",
	})
}

func testUserID() uuid.UUID {
	return uuid.MustParse("11111111-1111-1111-1111-111111111111")
}

func assertInvalidInput(t *testing.T, err error) {
	t.Helper()
	var invalid *apperrs.InvalidInputError
	if !errors.As(err, &invalid) {
		t.Fatalf("expected *InvalidInputError, got %v", err)
	}
}

func assertStrings(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}
