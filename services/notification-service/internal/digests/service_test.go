package digests

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/users"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/pkg/email"
)

type fakeDigestRepo struct {
	mu      sync.Mutex
	batch   *entity.NotificationDigestBatch
	grouped bool
	sent    bool
	failed  bool
	retry   bool
	claimed bool
}

func (f *fakeDigestRepo) QueueDigestItem(context.Context, QueueInput) (*entity.NotificationDigestBatch, bool, bool, error) {
	return f.batch, f.grouped, true, nil
}
func (f *fakeDigestRepo) ClaimDueDigestBatch(context.Context, time.Time) (*entity.NotificationDigestBatch, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.claimed {
		return nil, nil
	}
	f.claimed = true
	return f.batch, nil
}
func (f *fakeDigestRepo) GetDigestBatchByID(context.Context, uuid.UUID) (*entity.NotificationDigestBatch, error) {
	return f.batch, nil
}
func (f *fakeDigestRepo) GetDigestBatchByIDAndUser(context.Context, uuid.UUID, uuid.UUID) (*entity.NotificationDigestBatch, error) {
	return f.batch, nil
}
func (f *fakeDigestRepo) ListDigestBatchesByUser(context.Context, ListInput) ([]entity.NotificationDigestBatch, error) {
	return nil, nil
}
func (f *fakeDigestRepo) MarkDigestBatchSent(context.Context, uuid.UUID, time.Time) error {
	f.sent = true
	return nil
}
func (f *fakeDigestRepo) MarkDigestBatchFailed(_ context.Context, _ uuid.UUID, retry bool, _ *time.Time, _, _ string) error {
	f.failed = true
	f.retry = retry
	return nil
}

type fakeLookup struct{ userID uuid.UUID }

func (f fakeLookup) LookupByIDs(context.Context, []uuid.UUID) (map[uuid.UUID]users.UserProfile, error) {
	return map[uuid.UUID]users.UserProfile{f.userID: {UserID: f.userID, Email: "traveler@example.com", DisplayName: "Traveler"}}, nil
}

type fakeSender struct {
	mu      sync.Mutex
	sent    int
	err     error
	message email.EmailMessage
}

func (f *fakeSender) Send(_ context.Context, message email.EmailMessage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent++
	f.message = message
	return f.err
}

func (f *fakeSender) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sent
}

func TestProcessDueSendsOneGroupedEmailAndMarksSent(t *testing.T) {
	batch := testDigestBatch()
	repo := &fakeDigestRepo{batch: batch}
	sender := &fakeSender{}
	svc := New(repo, fakeLookup{userID: batch.UserID}, sender, nil, Config{PublicWebBaseURL: "https://travel.example", MaxAttempts: 3}, nil)
	result, err := svc.ProcessDue(context.Background(), ProcessInput{Now: time.Now(), Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if result.Sent != 1 || sender.count() != 1 || !repo.sent {
		t.Fatalf("unexpected result=%+v sender=%d sent=%v", result, sender.count(), repo.sent)
	}
}

func TestProcessDueConcurrentWorkersSendBatchOnce(t *testing.T) {
	batch := testDigestBatch()
	repo := &fakeDigestRepo{batch: batch}
	sender := &fakeSender{}
	svc := New(repo, fakeLookup{userID: batch.UserID}, sender, nil, Config{MaxAttempts: 3}, nil)

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.ProcessDue(context.Background(), ProcessInput{Now: time.Now(), Limit: 1})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if sender.count() != 1 {
		t.Fatalf("expected one delivery after concurrent claims, got %d", sender.count())
	}
}

func TestDigestEmailDoesNotReuseRawEventMessage(t *testing.T) {
	batch := testDigestBatch()
	batch.Items[0].Message = "PRIVATE COMMENT BODY SHOULD NOT LEAK"
	sender := &fakeSender{}
	svc := New(&fakeDigestRepo{batch: batch}, fakeLookup{userID: batch.UserID}, sender, nil, Config{PublicWebBaseURL: "https://travel.example"}, nil)
	if _, err := svc.ProcessDue(context.Background(), ProcessInput{Now: time.Now(), Limit: 1}); err != nil {
		t.Fatal(err)
	}
	sender.mu.Lock()
	message := sender.message
	sender.mu.Unlock()
	if strings.Contains(message.TextBody, "PRIVATE COMMENT") || strings.Contains(message.HTMLBody, "PRIVATE COMMENT") {
		t.Fatalf("digest leaked raw event message: %+v", message)
	}
	if !strings.Contains(message.TextBody, "3 comment updates") {
		t.Fatalf("expected deterministic grouped label, got %q", message.TextBody)
	}
}
func TestProcessDueRetriesTransientFailure(t *testing.T) {
	batch := testDigestBatch()
	batch.Attempts = 1
	repo := &fakeDigestRepo{batch: batch}
	sender := &fakeSender{err: errors.New("temporary")}
	svc := New(repo, fakeLookup{userID: batch.UserID}, sender, nil, Config{MaxAttempts: 3, RetryDelay: time.Minute}, nil)
	result, err := svc.ProcessDue(context.Background(), ProcessInput{Now: time.Now(), Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.Retrying != 1 || !repo.failed || !repo.retry {
		t.Fatalf("expected retry, got %+v repo=%+v", result, repo)
	}
}
func TestQueueReportsGroupedDigestKey(t *testing.T) {
	batch := testDigestBatch()
	repo := &fakeDigestRepo{batch: batch, grouped: true}
	svc := New(repo, nil, nil, nil, Config{}, nil)
	grouped, err := svc.Queue(context.Background(), QueueInput{Notification: entity.Notification{UserID: batch.UserID}, Channel: "email", Mode: "daily_digest", ScheduledFor: time.Now()})
	if err != nil || !grouped {
		t.Fatalf("expected grouped=true, got %v err=%v", grouped, err)
	}
}
func testDigestBatch() *entity.NotificationDigestBatch {
	userID := uuid.New()
	return &entity.NotificationDigestBatch{ID: uuid.New(), UserID: userID, Channel: "email", Mode: "daily_digest", Status: "processing", ScheduledFor: time.Now(), Attempts: 1, Items: []entity.NotificationDigestItem{{ID: uuid.New(), UserID: userID, Category: "comments", Priority: "normal", DigestKey: "trip:x:comments", Title: "Comment", Message: "A collaborator commented.", Metadata: map[string]any{"tripName": "Austria"}, EventCount: 3, LatestEventAt: time.Now()}}}
}
