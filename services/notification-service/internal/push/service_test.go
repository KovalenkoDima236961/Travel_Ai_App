package push

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
)

type fakePushRepo struct {
	rows       []entity.PushSubscription
	upserted   *entity.PushSubscription
	disabled   []uuid.UUID
	lastUsed   []uuid.UUID
	countError error
}

func (f *fakePushRepo) UpsertPushSubscription(_ context.Context, subscription entity.PushSubscription) (*entity.PushSubscription, error) {
	subscription.Status = entity.PushSubscriptionStatusActive
	subscription.CreatedAt = time.Now().UTC()
	f.upserted = &subscription
	f.rows = append(f.rows, subscription)
	return &subscription, nil
}

func (f *fakePushRepo) GetPushSubscriptionByEndpoint(_ context.Context, endpoint string) (*entity.PushSubscription, error) {
	for i := range f.rows {
		if f.rows[i].Endpoint == endpoint {
			return &f.rows[i], nil
		}
	}
	return nil, nil
}

func (f *fakePushRepo) ListActivePushSubscriptionsByUserID(_ context.Context, userID uuid.UUID) ([]entity.PushSubscription, error) {
	out := make([]entity.PushSubscription, 0)
	for _, row := range f.rows {
		if row.UserID == userID && row.Status == entity.PushSubscriptionStatusActive {
			out = append(out, row)
		}
	}
	return out, nil
}

func (f *fakePushRepo) DisablePushSubscriptionByEndpoint(_ context.Context, userID uuid.UUID, endpoint, _ string) error {
	for i := range f.rows {
		if f.rows[i].UserID == userID && f.rows[i].Endpoint == endpoint {
			f.rows[i].Status = entity.PushSubscriptionStatusDisabled
		}
	}
	return nil
}

func (f *fakePushRepo) DisablePushSubscriptionByID(_ context.Context, id uuid.UUID, _ string) error {
	f.disabled = append(f.disabled, id)
	for i := range f.rows {
		if f.rows[i].ID == id {
			f.rows[i].Status = entity.PushSubscriptionStatusDisabled
		}
	}
	return nil
}

func (f *fakePushRepo) CountActivePushSubscriptionsByUserID(_ context.Context, userID uuid.UUID) (int, error) {
	if f.countError != nil {
		return 0, f.countError
	}
	count := 0
	for _, row := range f.rows {
		if row.UserID == userID && row.Status == entity.PushSubscriptionStatusActive {
			count++
		}
	}
	return count, nil
}

func (f *fakePushRepo) UpdatePushSubscriptionLastUsed(_ context.Context, id uuid.UUID) error {
	f.lastUsed = append(f.lastUsed, id)
	return nil
}

type fakePushSender struct {
	result *PushSendResult
	err    error
	sent   []PushPayload
}

func (f *fakePushSender) Send(_ context.Context, _ PushSubscription, payload PushPayload) (*PushSendResult, error) {
	f.sent = append(f.sent, payload)
	if f.err != nil {
		return f.result, f.err
	}
	if f.result != nil {
		return f.result, nil
	}
	return &PushSendResult{StatusCode: 201}, nil
}

type fakePushGate struct {
	allowed bool
}

func (f fakePushGate) AllowPush(uuid.UUID, string) bool {
	return f.allowed
}

func enabledConfig(failOpen bool) Config {
	return Config{
		Enabled:         true,
		VAPIDPublicKey:  "public-key",
		VAPIDPrivateKey: "private-key",
		Subject:         "mailto:test@example.com",
		Timeout:         time.Second,
		TTLSeconds:      30,
		Urgency:         "normal",
		FailOpen:        failOpen,
	}
}

func TestSubscribeStoresSubscription(t *testing.T) {
	repo := &fakePushRepo{}
	svc := New(enabledConfig(true), repo, &fakePushSender{}, nil)
	userID := uuid.New()

	subscribed, err := svc.Subscribe(context.Background(), SubscribeInput{
		UserID:   userID,
		Endpoint: "https://push.example.test/subscription/1",
		P256DH:   "p256dh",
		Auth:     "auth",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !subscribed {
		t.Fatal("expected subscribed=true")
	}
	if repo.upserted == nil || repo.upserted.UserID != userID || repo.upserted.Endpoint == "" {
		t.Fatalf("expected upserted subscription, got %+v", repo.upserted)
	}
}

func TestSubscribeRejectsInvalidEndpoint(t *testing.T) {
	svc := New(enabledConfig(true), &fakePushRepo{}, &fakePushSender{}, nil)
	_, err := svc.Subscribe(context.Background(), SubscribeInput{
		UserID:   uuid.New(),
		Endpoint: "http://not-https.example.test",
		P256DH:   "p256dh",
		Auth:     "auth",
	})
	var invalid *apperrs.InvalidInputError
	if !errors.As(err, &invalid) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestSendPushSendsForEligibleNotification(t *testing.T) {
	userID := uuid.New()
	tripID := uuid.New()
	subscriptionID := uuid.New()
	repo := &fakePushRepo{rows: []entity.PushSubscription{{
		ID:       subscriptionID,
		UserID:   userID,
		Endpoint: "https://push.example.test/subscription/1",
		P256DH:   "p256dh",
		Auth:     "auth",
		Status:   entity.PushSubscriptionStatusActive,
	}}}
	sender := &fakePushSender{}
	svc := New(enabledConfig(true), repo, sender, nil)

	result, err := svc.SendPushForNotifications(context.Background(), []entity.Notification{{
		ID:        uuid.New(),
		UserID:    userID,
		TripID:    &tripID,
		Type:      notifications.TypeItineraryGenerated,
		Title:     "Itinerary ready",
		Message:   "Done",
		CreatedAt: time.Now().UTC(),
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attempted != 1 || result.Sent != 1 || result.Failed != 0 {
		t.Fatalf("unexpected result %+v", result)
	}
	if len(sender.sent) != 1 || sender.sent[0].URL != "/trips/"+tripID.String() {
		t.Fatalf("unexpected payloads %+v", sender.sent)
	}
	if len(repo.lastUsed) != 1 || repo.lastUsed[0] != subscriptionID {
		t.Fatalf("expected last_used update, got %+v", repo.lastUsed)
	}
}

func TestSendPushSkipsDisabledPreference(t *testing.T) {
	userID := uuid.New()
	repo := &fakePushRepo{rows: []entity.PushSubscription{{
		ID:       uuid.New(),
		UserID:   userID,
		Endpoint: "https://push.example.test/subscription/1",
		P256DH:   "p256dh",
		Auth:     "auth",
		Status:   entity.PushSubscriptionStatusActive,
	}}}
	sender := &fakePushSender{}
	svc := New(enabledConfig(true), repo, sender, nil)

	result, err := svc.SendPushForNotifications(
		context.Background(),
		[]entity.Notification{{ID: uuid.New(), UserID: userID, Type: notifications.TypeCommentCreated}},
		fakePushGate{allowed: false},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Skipped != 1 || result.SkippedByPreference != 1 || result.Attempted != 0 {
		t.Fatalf("expected preference skip, got %+v", result)
	}
	if len(sender.sent) != 0 {
		t.Fatalf("expected no sends, got %+v", sender.sent)
	}
}

func TestSendPushDisablesGoneSubscription(t *testing.T) {
	userID := uuid.New()
	subscriptionID := uuid.New()
	repo := &fakePushRepo{rows: []entity.PushSubscription{{
		ID:       subscriptionID,
		UserID:   userID,
		Endpoint: "https://push.example.test/subscription/1",
		P256DH:   "p256dh",
		Auth:     "auth",
		Status:   entity.PushSubscriptionStatusActive,
	}}}
	svc := New(enabledConfig(true), repo, &fakePushSender{
		result: &PushSendResult{StatusCode: 410, SubscriptionGone: true},
	}, nil)

	result, err := svc.SendPushForNotifications(context.Background(), []entity.Notification{{
		ID:     uuid.New(),
		UserID: userID,
		Type:   notifications.TypeCommentCreated,
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SubscriptionsDisabled != 1 || len(repo.disabled) != 1 || repo.disabled[0] != subscriptionID {
		t.Fatalf("expected disabled subscription, result=%+v disabled=%+v", result, repo.disabled)
	}
}

func TestSendPushFailClosedReturnsError(t *testing.T) {
	userID := uuid.New()
	repo := &fakePushRepo{rows: []entity.PushSubscription{{
		ID:       uuid.New(),
		UserID:   userID,
		Endpoint: "https://push.example.test/subscription/1",
		P256DH:   "p256dh",
		Auth:     "auth",
		Status:   entity.PushSubscriptionStatusActive,
	}}}
	svc := New(enabledConfig(false), repo, &fakePushSender{err: errors.New("send failed")}, nil)

	result, err := svc.SendPushForNotifications(context.Background(), []entity.Notification{{
		ID:     uuid.New(),
		UserID: userID,
		Type:   notifications.TypeCommentCreated,
	}})
	if err == nil {
		t.Fatal("expected fail-closed error")
	}
	if result.Failed != 1 {
		t.Fatalf("expected failed=1, got %+v", result)
	}
}
