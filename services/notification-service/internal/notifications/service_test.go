package notifications

import (
	"context"
	"errors"
	"sort"
	"strings"
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
	if created != 1 {
		t.Fatalf("expected 1 created (self-notification skipped), got %d", created)
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
	if created != 0 {
		t.Fatalf("expected 0 created, got %d", created)
	}
}

func TestCreateBatch_ValidatesFields(t *testing.T) {
	svc := New(newFakeRepo(), nil)
	user := uuid.New()

	cases := map[string]func(*CreateInput){
		"missing user":    func(in *CreateInput) { in.UserID = uuid.Nil },
		"missing type":    func(in *CreateInput) { in.Type = "" },
		"unknown type":    func(in *CreateInput) { in.Type = "definitely_not_a_type" },
		"missing title":   func(in *CreateInput) { in.Title = "  " },
		"long title":      func(in *CreateInput) { in.Title = strings.Repeat("x", MaxTitleLength+1) },
		"missing message": func(in *CreateInput) { in.Message = "" },
		"long message":    func(in *CreateInput) { in.Message = strings.Repeat("y", MaxMessageLength+1) },
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
	if err != nil || created != 1 {
		t.Fatalf("expected 1 created with nil metadata, got %d err=%v", created, err)
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
