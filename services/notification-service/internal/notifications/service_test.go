package notifications

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/errs"
)

// fakeRepo is an in-memory Repository used to test service behavior (validation,
// self-notification skipping, pagination, read state) without a database.
type fakeRepo struct {
	rows []entity.Notification
	now  time.Time
}

type atomicFakeRepo struct {
	*fakeRepo
	mu     sync.Mutex
	claims map[string]atomicFakeClaim
}

type atomicFakeClaim struct {
	latest         time.Time
	notificationID *uuid.UUID
}

func newAtomicFakeRepo() *atomicFakeRepo {
	return &atomicFakeRepo{fakeRepo: newFakeRepo(), claims: make(map[string]atomicFakeClaim)}
}

func (f *atomicFakeRepo) ClaimNotificationDedupe(_ context.Context, userID uuid.UUID, dedupeKey string, now, since time.Time) (bool, *uuid.UUID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := userID.String() + "|" + dedupeKey
	claim, ok := f.claims[key]
	if ok && !claim.latest.Before(since) {
		claim.latest = now
		f.claims[key] = claim
		return true, claim.notificationID, nil
	}
	f.claims[key] = atomicFakeClaim{latest: now}
	return false, nil, nil
}

func (f *atomicFakeRepo) BindNotificationDedupe(_ context.Context, userID uuid.UUID, dedupeKey string, notificationID uuid.UUID, since time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := userID.String() + "|" + dedupeKey
	claim, ok := f.claims[key]
	if ok && !claim.latest.Before(since) {
		id := notificationID
		claim.notificationID = &id
		f.claims[key] = claim
	}
	return nil
}

type groupingFakeRepo struct{ *fakeRepo }

func (f *groupingFakeRepo) FindRecentNotificationByDigestKey(_ context.Context, userID uuid.UUID, digestKey string, since time.Time) (*entity.Notification, error) {
	for i := len(f.rows) - 1; i >= 0; i-- {
		row := f.rows[i]
		if row.UserID == userID && row.DigestKey != nil && *row.DigestKey == digestKey &&
			canGroupNotification(row) && !row.LatestEventAt.Before(since) {
			return &row, nil
		}
	}
	return nil, nil
}

func (f *groupingFakeRepo) GroupRelatedNotification(_ context.Context, id, userID uuid.UUID, latest entity.Notification) (*entity.Notification, error) {
	for i := range f.rows {
		if f.rows[i].ID != id || f.rows[i].UserID != userID {
			continue
		}
		mergeNotificationOccurrence(&f.rows[i], latest)
		f.rows[i].ReadAt = nil
		row := f.rows[i]
		return &row, nil
	}
	return nil, domainerrs.ErrNotFound
}

type fakeInAppGate struct {
	allowed map[string]bool
}

func (g fakeInAppGate) AllowInApp(_ uuid.UUID, notificationType string) bool {
	allowed, ok := g.allowed[notificationType]
	if !ok {
		return true
	}
	return allowed
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{now: time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)}
}

func (f *fakeRepo) CreateNotifications(_ context.Context, notifications []entity.Notification) (int, error) {
	for i := range notifications {
		n := notifications[i]
		f.now = f.now.Add(time.Second)
		n.CreatedAt = f.now
		f.rows = append(f.rows, n)
	}
	return len(notifications), nil
}

func (f *fakeRepo) ListNotificationsByUser(_ context.Context, in ListInput) ([]entity.Notification, error) {
	owned := make([]entity.Notification, 0)
	for _, n := range f.rows {
		if n.UserID != in.UserID {
			continue
		}
		owned = append(owned, n)
	}
	// Newest first: created_at DESC, id DESC.
	sort.Slice(owned, func(i, j int) bool {
		if owned[i].CreatedAt.Equal(owned[j].CreatedAt) {
			return owned[i].ID.String() > owned[j].ID.String()
		}
		return owned[i].CreatedAt.After(owned[j].CreatedAt)
	})
	if in.CursorCreatedAt != nil && in.CursorID != nil {
		filtered := owned[:0:0]
		for _, n := range owned {
			if n.CreatedAt.Before(*in.CursorCreatedAt) ||
				(n.CreatedAt.Equal(*in.CursorCreatedAt) && n.ID.String() < in.CursorID.String()) {
				filtered = append(filtered, n)
			}
		}
		owned = filtered
	}
	if in.Limit > 0 && len(owned) > in.Limit {
		owned = owned[:in.Limit]
	}
	return owned, nil
}

func (f *fakeRepo) GetNotificationByIDAndUser(_ context.Context, id, userID uuid.UUID) (*entity.Notification, error) {
	for i := range f.rows {
		if f.rows[i].ID == id && f.rows[i].UserID == userID {
			n := f.rows[i]
			return &n, nil
		}
	}
	return nil, domainerrs.ErrNotFound
}

func (f *fakeRepo) CountUnreadNotifications(_ context.Context, userID uuid.UUID) (int, error) {
	count := 0
	for _, n := range f.rows {
		if n.UserID == userID && n.ReadAt == nil {
			count++
		}
	}
	return count, nil
}

func (f *fakeRepo) MarkNotificationRead(_ context.Context, id, userID uuid.UUID) (*entity.Notification, error) {
	for i := range f.rows {
		if f.rows[i].ID == id && f.rows[i].UserID == userID {
			if f.rows[i].ReadAt == nil {
				t := f.now
				f.rows[i].ReadAt = &t
			}
			n := f.rows[i]
			return &n, nil
		}
	}
	return nil, domainerrs.ErrNotFound
}

func (f *fakeRepo) MarkAllNotificationsRead(_ context.Context, userID uuid.UUID) (int, error) {
	changed := 0
	for i := range f.rows {
		if f.rows[i].UserID == userID && f.rows[i].ReadAt == nil {
			t := f.now
			f.rows[i].ReadAt = &t
			changed++
		}
	}
	return changed, nil
}

func (f *fakeRepo) FindRecentNotificationByDedupeKey(_ context.Context, userID uuid.UUID, dedupeKey string, since time.Time) (*entity.Notification, error) {
	for i := len(f.rows) - 1; i >= 0; i-- {
		row := f.rows[i]
		if row.UserID == userID && row.DedupeKey != nil && *row.DedupeKey == dedupeKey && !row.LatestEventAt.Before(since) {
			return &row, nil
		}
	}
	return nil, nil
}

func (f *fakeRepo) GroupNotification(_ context.Context, id, userID uuid.UUID, latest entity.Notification) (*entity.Notification, error) {
	for i := range f.rows {
		if f.rows[i].ID != id || f.rows[i].UserID != userID {
			continue
		}
		f.rows[i].GroupedCount++
		f.rows[i].LatestEventAt = latest.LatestEventAt
		f.rows[i].Title = latest.Title
		f.rows[i].Message = latest.Message
		f.rows[i].Metadata = latest.Metadata
		row := f.rows[i]
		return &row, nil
	}
	return nil, domainerrs.ErrNotFound
}

func validInput(userID uuid.UUID) CreateInput {
	return CreateInput{
		UserID:  userID,
		Type:    TypeCommentCreated,
		Title:   "New comment",
		Message: "A collaborator commented on Day 2 · Louvre Museum.",
	}
}

func TestCreateBatch_SkipsSelfNotification(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo, nil)

	actor := uuid.New()
	other := uuid.New()

	selfNotify := validInput(actor)
	selfNotify.ActorUserID = &actor

	otherNotify := validInput(other)
	otherNotify.ActorUserID = &actor

	created, err := svc.CreateBatch(context.Background(), []CreateInput{selfNotify, otherNotify})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("expected 1 created (self-notification skipped), got %d", len(created))
	}
	if created[0].UserID != other {
		t.Fatalf("expected the created notification to target the non-actor recipient, got %s", created[0].UserID)
	}
	if len(repo.rows) != 1 || repo.rows[0].UserID != other {
		t.Fatalf("expected only the non-actor recipient stored, got %+v", repo.rows)
	}
}

func TestCreateBatch_AllSelfNotificationsCreatesNothing(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo, nil)

	actor := uuid.New()
	in := validInput(actor)
	in.ActorUserID = &actor

	created, err := svc.CreateBatch(context.Background(), []CreateInput{in})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(created) != 0 {
		t.Fatalf("expected 0 created, got %d", len(created))
	}
}

func TestCreateBatchWithPreferencesSkipsDisabledInApp(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo, nil)
	user := uuid.New()
	actor := uuid.New()
	in := validInput(user)
	in.ActorUserID = &actor

	result, err := svc.CreateBatchWithPreferences(context.Background(), []CreateInput{in}, fakeInAppGate{
		allowed: map[string]bool{TypeCommentCreated: false},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Created) != 0 || len(repo.rows) != 0 {
		t.Fatalf("expected no stored in-app notifications, got result=%+v rows=%+v", result, repo.rows)
	}
	if result.Skipped != 1 || result.SkippedByPreference != 1 {
		t.Fatalf("expected preference skip counts, got %+v", result)
	}
	if len(result.EmailCandidates) != 1 || result.EmailCandidates[0].UserID != user {
		t.Fatalf("expected email candidate preserved despite in-app skip, got %+v", result.EmailCandidates)
	}
}

func TestCreateBatchWithPreferencesCreatesWhenInAppEnabled(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo, nil)
	user := uuid.New()

	result, err := svc.CreateBatchWithPreferences(context.Background(), []CreateInput{validInput(user)}, fakeInAppGate{
		allowed: map[string]bool{TypeCommentCreated: true},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Created) != 1 || len(repo.rows) != 1 {
		t.Fatalf("expected one stored notification, got result=%+v rows=%+v", result, repo.rows)
	}
	if result.Skipped != 0 || result.SkippedByPreference != 0 {
		t.Fatalf("expected no skip counts, got %+v", result)
	}
}

func TestCreateBatch_ValidatesFields(t *testing.T) {
	svc := New(newFakeRepo(), nil)
	user := uuid.New()

	cases := map[string]func(*CreateInput){
		"missing user":    func(in *CreateInput) { in.UserID = uuid.Nil },
		"missing type":    func(in *CreateInput) { in.Type = "" },
		"missing title":   func(in *CreateInput) { in.Title = "  " },
		"long title":      func(in *CreateInput) { in.Title = strings.Repeat("x", MaxTitleLength+1) },
		"missing message": func(in *CreateInput) { in.Message = "" },
		"long message":    func(in *CreateInput) { in.Message = strings.Repeat("y", MaxMessageLength+1) },
		"long digest key": func(in *CreateInput) { value := strings.Repeat("d", MaxGroupingKeyLength+1); in.DigestKey = &value },
		"long dedupe key": func(in *CreateInput) { value := strings.Repeat("e", MaxGroupingKeyLength+1); in.DedupeKey = &value },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			in := validInput(user)
			mutate(&in)
			_, err := svc.CreateBatch(context.Background(), []CreateInput{in})
			var invalid *apperrs.InvalidInputError
			if !errors.As(err, &invalid) {
				t.Fatalf("expected InvalidInputError, got %v", err)
			}
		})
	}
}

func TestCreateBatch_AllowsUnknownTypeForFutureInAppCompatibility(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo, nil)
	in := validInput(uuid.New())
	in.Type = "future_notification_type"

	created, err := svc.CreateBatch(context.Background(), []CreateInput{in})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(created) != 1 || created[0].Type != "future_notification_type" {
		t.Fatalf("expected unknown type to be created in-app, got %+v", created)
	}
}

func TestCreateBatch_EmptyAndOversize(t *testing.T) {
	svc := New(newFakeRepo(), nil)

	if _, err := svc.CreateBatch(context.Background(), nil); err == nil {
		t.Fatal("expected error for empty batch")
	}

	user := uuid.New()
	big := make([]CreateInput, MaxBatchSize+1)
	for i := range big {
		big[i] = validInput(user)
	}
	if _, err := svc.CreateBatch(context.Background(), big); err == nil {
		t.Fatal("expected error for oversize batch")
	}
}

func TestCreateBatch_EmptyMetadataDoesNotBreak(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo, nil)
	in := validInput(uuid.New())
	in.Metadata = nil

	created, err := svc.CreateBatch(context.Background(), []CreateInput{in})
	if err != nil || len(created) != 1 {
		t.Fatalf("expected 1 created with nil metadata, got %d err=%v", len(created), err)
	}
}

func TestCreateBatch_GroupsDuplicateKeysWithinOneBatch(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo, nil)
	userID := uuid.New()
	dedupeKey := "comment:one:recipient:" + userID.String()
	first := validInput(userID)
	first.DedupeKey = &dedupeKey
	second := first
	second.Message = "The latest safe state."

	result, err := svc.CreateBatchWithPreferences(context.Background(), []CreateInput{first, second}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Created) != 1 || len(result.EmailCandidates) != 1 || len(repo.rows) != 1 {
		t.Fatalf("expected one grouped row/candidate, got result=%+v rows=%+v", result, repo.rows)
	}
	if result.Created[0].GroupedCount != 2 || result.Created[0].Message != second.Message {
		t.Fatalf("expected latest state and count=2, got %+v", result.Created[0])
	}
	if result.DuplicatesDropped != 1 || result.Grouped != 1 {
		t.Fatalf("expected one grouped duplicate, got %+v", result)
	}
}

func TestCreateBatch_GroupsPersistedDuplicateWithoutAnotherCandidate(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo, nil)
	userID := uuid.New()
	dedupeKey := "reminder:one:recipient:" + userID.String()
	input := validInput(userID)
	input.DedupeKey = &dedupeKey
	if _, err := svc.CreateBatch(context.Background(), []CreateInput{input}); err != nil {
		t.Fatal(err)
	}
	result, err := svc.CreateBatchWithPreferences(context.Background(), []CreateInput{input}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Created) != 0 || len(result.EmailCandidates) != 0 || len(repo.rows) != 1 {
		t.Fatalf("expected persisted duplicate to be grouped and suppressed, got result=%+v rows=%+v", result, repo.rows)
	}
	if repo.rows[0].GroupedCount != 2 || result.DuplicatesDropped != 1 {
		t.Fatalf("expected grouped_count=2 and one dropped duplicate, got result=%+v row=%+v", result, repo.rows[0])
	}
}

func TestCreateBatch_AtomicDedupeWorksWithoutInAppRow(t *testing.T) {
	repo := newAtomicFakeRepo()
	svc := New(repo, nil)
	userID := uuid.New()
	dedupeKey := "muted-event:one:recipient:" + userID.String()
	input := validInput(userID)
	input.DedupeKey = &dedupeKey
	gate := fakeInAppGate{allowed: map[string]bool{TypeCommentCreated: false}}

	first, err := svc.CreateBatchWithPreferences(context.Background(), []CreateInput{input}, gate)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.CreateBatchWithPreferences(context.Background(), []CreateInput{input}, gate)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.EmailCandidates) != 1 || len(second.EmailCandidates) != 0 || len(repo.rows) != 0 {
		t.Fatalf("expected one channel-independent event then an exact drop, first=%+v second=%+v", first, second)
	}
	if second.DuplicatesDropped != 1 || second.SkippedByPreference != 0 {
		t.Fatalf("expected the retry to be classified as a duplicate, got %+v", second)
	}
}

func TestCreateBatch_ValidatesWholeBatchBeforeAtomicClaim(t *testing.T) {
	repo := newAtomicFakeRepo()
	svc := New(repo, nil)
	userID := uuid.New()
	dedupeKey := "valid:event"
	valid := validInput(userID)
	valid.DedupeKey = &dedupeKey
	invalid := validInput(userID)
	invalid.Message = ""

	if _, err := svc.CreateBatch(context.Background(), []CreateInput{valid, invalid}); err == nil {
		t.Fatal("expected invalid batch error")
	}
	if len(repo.claims) != 0 {
		t.Fatalf("expected no dedupe claims for an invalid batch, got %+v", repo.claims)
	}
}

func TestCreateBatch_GroupsRelatedEventsWithoutCollapsingChannelCandidates(t *testing.T) {
	repo := &groupingFakeRepo{fakeRepo: newFakeRepo()}
	svc := New(repo, nil)
	userID := uuid.New()
	digestKey := "trip:one:comments"
	firstKey, secondKey := "comment:one", "comment:two"
	first := validInput(userID)
	first.DigestKey, first.DedupeKey = &digestKey, &firstKey
	second := validInput(userID)
	second.Message = "A second collaborator commented."
	second.DigestKey, second.DedupeKey = &digestKey, &secondKey

	result, err := svc.CreateBatchWithPreferences(context.Background(), []CreateInput{first, second}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Created) != 1 || result.Created[0].GroupedCount != 2 || len(repo.rows) != 1 {
		t.Fatalf("expected one grouped in-app card, got result=%+v rows=%+v", result, repo.rows)
	}
	if len(result.EmailCandidates) != 2 || result.EmailCandidates[0].GroupedCount != 1 || result.EmailCandidates[1].GroupedCount != 1 {
		t.Fatalf("expected one channel candidate per real event, got %+v", result.EmailCandidates)
	}
	if result.Grouped != 1 || result.DuplicatesDropped != 0 {
		t.Fatalf("expected related grouping without exact dedupe, got %+v", result)
	}
}

func TestCreateBatch_QueuesOneOccurrenceWhenGroupingIntoPersistedCard(t *testing.T) {
	repo := &groupingFakeRepo{fakeRepo: newFakeRepo()}
	svc := New(repo, nil)
	userID := uuid.New()
	digestKey := "trip:one:comments"
	firstKey, secondKey := "comment:one", "comment:two"
	first := validInput(userID)
	first.DigestKey, first.DedupeKey = &digestKey, &firstKey
	second := validInput(userID)
	second.DigestKey, second.DedupeKey = &digestKey, &secondKey

	if _, err := svc.CreateBatch(context.Background(), []CreateInput{first}); err != nil {
		t.Fatal(err)
	}
	result, err := svc.CreateBatchWithPreferences(context.Background(), []CreateInput{second}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Created) != 0 || len(result.GroupedInApp) != 1 || result.GroupedInApp[0].GroupedCount != 1 {
		t.Fatalf("expected one digest occurrence for the persisted group, got %+v", result)
	}
	if len(repo.rows) != 1 || repo.rows[0].GroupedCount != 2 {
		t.Fatalf("expected persisted card count=2, got %+v", repo.rows)
	}
}

func TestCreateBatch_StripsSensitiveMetadataRecursively(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo, nil)
	input := validInput(uuid.New())
	input.Metadata = map[string]any{
		"tripId": "safe", "publicShareToken": "secret-value", "receiptOCRText": "private",
		"nested": map[string]any{"destination": "Vienna", "privateNotes": "do not persist"},
	}
	created, err := svc.CreateBatch(context.Background(), []CreateInput{input})
	if err != nil {
		t.Fatal(err)
	}
	metadata := created[0].Metadata
	if metadata["tripId"] != "safe" || metadata["publicShareToken"] != nil || metadata["receiptOCRText"] != nil {
		t.Fatalf("unexpected sanitized metadata: %#v", metadata)
	}
	nested, ok := metadata["nested"].(map[string]any)
	if !ok || nested["destination"] != "Vienna" || nested["privateNotes"] != nil {
		t.Fatalf("nested metadata was not sanitized: %#v", metadata["nested"])
	}
}

func TestList_Pagination(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo, nil)
	user := uuid.New()

	// Seed 3 notifications for the user and 1 for someone else.
	for i := 0; i < 3; i++ {
		if _, err := svc.CreateBatch(context.Background(), []CreateInput{validInput(user)}); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	if _, err := svc.CreateBatch(context.Background(), []CreateInput{validInput(uuid.New())}); err != nil {
		t.Fatalf("seed other: %v", err)
	}

	page1, err := svc.List(context.Background(), ListInput{UserID: user, Limit: 2})
	if err != nil {
		t.Fatalf("list page 1: %v", err)
	}
	if len(page1.Notifications) != 2 {
		t.Fatalf("expected 2 on page 1, got %d", len(page1.Notifications))
	}
	if page1.NextCursor == "" {
		t.Fatal("expected a next cursor when more rows exist")
	}

	createdAt, id, err := DecodeCursor(page1.NextCursor)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	page2, err := svc.List(context.Background(), ListInput{UserID: user, Limit: 2, CursorCreatedAt: createdAt, CursorID: id})
	if err != nil {
		t.Fatalf("list page 2: %v", err)
	}
	if len(page2.Notifications) != 1 {
		t.Fatalf("expected 1 on page 2, got %d", len(page2.Notifications))
	}
	if page2.NextCursor != "" {
		t.Fatal("expected no next cursor on the last page")
	}
}

func TestUnreadCountAndMarkRead(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo, nil)
	user := uuid.New()

	for i := 0; i < 3; i++ {
		if _, err := svc.CreateBatch(context.Background(), []CreateInput{validInput(user)}); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	count, err := svc.CountUnread(context.Background(), user)
	if err != nil || count != 3 {
		t.Fatalf("expected 3 unread, got %d err=%v", count, err)
	}

	// Mark one read (idempotent: marking twice keeps the count consistent).
	target := repo.rows[0].ID
	if _, err := svc.MarkRead(context.Background(), target, user); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	if _, err := svc.MarkRead(context.Background(), target, user); err != nil {
		t.Fatalf("mark read (idempotent): %v", err)
	}
	if count, _ = svc.CountUnread(context.Background(), user); count != 2 {
		t.Fatalf("expected 2 unread after one read, got %d", count)
	}

	changed, err := svc.MarkAllRead(context.Background(), user)
	if err != nil || changed != 2 {
		t.Fatalf("expected 2 changed by mark-all, got %d err=%v", changed, err)
	}
	if count, _ = svc.CountUnread(context.Background(), user); count != 0 {
		t.Fatalf("expected 0 unread after mark-all, got %d", count)
	}
}
