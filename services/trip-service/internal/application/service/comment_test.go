package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
)

func ctxWithUserID(id uuid.UUID) context.Context {
	return auth.WithUser(context.Background(), auth.AuthenticatedUser{
		ID:    id,
		Email: "user@example.com",
	})
}

// commentRepo builds a repo whose GetByID returns a COMPLETED trip owned by
// testUserID with a known two-day itinerary, so item-existence checks pass.
func commentRepo(t *testing.T) (*mockRepo, uuid.UUID) {
	t.Helper()
	owner := testUserID()
	tripID := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{
			ID:        tripID,
			UserID:    &owner,
			Status:    entity.StatusCompleted,
			Days:      2,
			Itinerary: validExistingItineraryRaw(t),
		},
	}
	return repo, tripID
}

func acceptedCollaborator(role entity.CollaboratorRole) *entity.TripCollaborator {
	return &entity.TripCollaborator{Role: role, Status: entity.CollaboratorStatusAccepted}
}

func validCreateComment() appdto.CreateCommentInput {
	return appdto.CreateCommentInput{DayNumber: 1, ItemIndex: 0, Body: "Can we move this earlier?"}
}

func assertForbidden(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, apperrs.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func assertNotFound(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, domainerrs.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateComment_OwnerSuccess(t *testing.T) {
	repo, tripID := commentRepo(t)
	svc := newTestService(repo, &mockGenerator{})

	info, err := svc.CreateComment(ctxWithUserID(testUserID()), tripID, validCreateComment())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !info.IsAuthor || !info.CanEdit || !info.CanDelete {
		t.Fatalf("owner author should have full permissions, got %+v", info)
	}
	if info.Comment.Body != "Can we move this earlier?" {
		t.Fatalf("unexpected body: %q", info.Comment.Body)
	}
	if len(repo.comments) != 1 {
		t.Fatalf("expected 1 stored comment, got %d", len(repo.comments))
	}
}

func TestCreateComment_ViewerCanComment(t *testing.T) {
	repo, tripID := commentRepo(t)
	repo.collaboratorByUser = acceptedCollaborator(entity.CollaboratorRoleViewer)
	svc := newTestService(repo, &mockGenerator{})

	viewer := uuid.New()
	info, err := svc.CreateComment(ctxWithUserID(viewer), tripID, validCreateComment())
	if err != nil {
		t.Fatalf("viewer should be able to comment, got %v", err)
	}
	if !info.IsAuthor || !info.CanEdit {
		t.Fatalf("viewer author should be able to edit own comment, got %+v", info)
	}
	if info.CanDelete != true {
		t.Fatalf("author should be able to delete own comment, got %+v", info)
	}
}

func TestCreateComment_EditorCanComment(t *testing.T) {
	repo, tripID := commentRepo(t)
	repo.collaboratorByUser = acceptedCollaborator(entity.CollaboratorRoleEditor)
	svc := newTestService(repo, &mockGenerator{})

	if _, err := svc.CreateComment(ctxWithUserID(uuid.New()), tripID, validCreateComment()); err != nil {
		t.Fatalf("editor should be able to comment, got %v", err)
	}
}

func TestCreateComment_NonCollaboratorForbidden(t *testing.T) {
	repo, tripID := commentRepo(t)
	svc := newTestService(repo, &mockGenerator{})

	_, err := svc.CreateComment(ctxWithUserID(uuid.New()), tripID, validCreateComment())
	assertNotFound(t, err)
}

func TestCreateComment_PendingCollaboratorCannotComment(t *testing.T) {
	repo, tripID := commentRepo(t)
	repo.collaboratorByUser = &entity.TripCollaborator{Role: entity.CollaboratorRoleEditor, Status: entity.CollaboratorStatusPending}
	svc := newTestService(repo, &mockGenerator{})

	_, err := svc.CreateComment(ctxWithUserID(uuid.New()), tripID, validCreateComment())
	assertNotFound(t, err)
}

func TestCreateComment_RemovedCollaboratorCannotComment(t *testing.T) {
	repo, tripID := commentRepo(t)
	repo.collaboratorByUser = &entity.TripCollaborator{Role: entity.CollaboratorRoleEditor, Status: entity.CollaboratorStatusRemoved}
	svc := newTestService(repo, &mockGenerator{})

	_, err := svc.CreateComment(ctxWithUserID(uuid.New()), tripID, validCreateComment())
	assertNotFound(t, err)
}

func TestCreateComment_EmptyBodyRejected(t *testing.T) {
	repo, tripID := commentRepo(t)
	svc := newTestService(repo, &mockGenerator{})

	in := validCreateComment()
	in.Body = ""
	_, err := svc.CreateComment(ctxWithUserID(testUserID()), tripID, in)
	assertInvalidInput(t, err)
}

func TestCreateComment_WhitespaceBodyRejected(t *testing.T) {
	repo, tripID := commentRepo(t)
	svc := newTestService(repo, &mockGenerator{})

	in := validCreateComment()
	in.Body = "   \n\t  "
	_, err := svc.CreateComment(ctxWithUserID(testUserID()), tripID, in)
	assertInvalidInput(t, err)
}

func TestCreateComment_BodyTooLongRejected(t *testing.T) {
	repo, tripID := commentRepo(t)
	svc := newTestService(repo, &mockGenerator{})

	in := validCreateComment()
	in.Body = strings.Repeat("a", maxCommentBodyLength+1)
	_, err := svc.CreateComment(ctxWithUserID(testUserID()), tripID, in)
	assertInvalidInput(t, err)
}

func TestCreateComment_InvalidDayNumberRejected(t *testing.T) {
	repo, tripID := commentRepo(t)
	svc := newTestService(repo, &mockGenerator{})

	in := validCreateComment()
	in.DayNumber = 0
	_, err := svc.CreateComment(ctxWithUserID(testUserID()), tripID, in)
	assertInvalidInput(t, err)
}

func TestCreateComment_InvalidItemIndexRejected(t *testing.T) {
	repo, tripID := commentRepo(t)
	svc := newTestService(repo, &mockGenerator{})

	in := validCreateComment()
	in.ItemIndex = -1
	_, err := svc.CreateComment(ctxWithUserID(testUserID()), tripID, in)
	assertInvalidInput(t, err)
}

func TestCreateComment_NonExistentItemRejected(t *testing.T) {
	repo, tripID := commentRepo(t)
	svc := newTestService(repo, &mockGenerator{})

	in := validCreateComment()
	in.DayNumber = 99
	if _, err := svc.CreateComment(ctxWithUserID(testUserID()), tripID, in); !errors.As(err, new(*apperrs.InvalidInputError)) {
		t.Fatalf("expected invalid input for non-existent day, got %v", err)
	}

	in = validCreateComment()
	in.ItemIndex = 50
	if _, err := svc.CreateComment(ctxWithUserID(testUserID()), tripID, in); !errors.As(err, new(*apperrs.InvalidInputError)) {
		t.Fatalf("expected invalid input for non-existent item index, got %v", err)
	}
}

func TestListComments_ExcludesDeleted(t *testing.T) {
	repo, tripID := commentRepo(t)
	svc := newTestService(repo, &mockGenerator{})

	repo.comments = []entity.ItineraryComment{
		{ID: uuid.New(), TripID: tripID, DayNumber: 1, ItemIndex: 0, AuthorUserID: testUserID(), Body: "active", Status: entity.CommentStatusActive},
		{ID: uuid.New(), TripID: tripID, DayNumber: 1, ItemIndex: 0, AuthorUserID: testUserID(), Body: "gone", Status: entity.CommentStatusDeleted},
	}

	infos, err := svc.ListComments(ctxWithUserID(testUserID()), tripID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 1 || infos[0].Comment.Body != "active" {
		t.Fatalf("expected only active comment, got %+v", infos)
	}
}

func TestListItemComments_FiltersByItem(t *testing.T) {
	repo, tripID := commentRepo(t)
	svc := newTestService(repo, &mockGenerator{})

	repo.comments = []entity.ItineraryComment{
		{ID: uuid.New(), TripID: tripID, DayNumber: 1, ItemIndex: 0, AuthorUserID: testUserID(), Body: "1-0", Status: entity.CommentStatusActive},
		{ID: uuid.New(), TripID: tripID, DayNumber: 2, ItemIndex: 1, AuthorUserID: testUserID(), Body: "2-1", Status: entity.CommentStatusActive},
	}

	infos, err := svc.ListItemComments(ctxWithUserID(testUserID()), tripID, 2, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 1 || infos[0].Comment.Body != "2-1" {
		t.Fatalf("expected only the day 2 item 1 comment, got %+v", infos)
	}
}

func TestListCommentCounts_ExcludesDeleted(t *testing.T) {
	repo, tripID := commentRepo(t)
	svc := newTestService(repo, &mockGenerator{})

	repo.comments = []entity.ItineraryComment{
		{ID: uuid.New(), TripID: tripID, DayNumber: 1, ItemIndex: 0, AuthorUserID: testUserID(), Body: "a", Status: entity.CommentStatusActive},
		{ID: uuid.New(), TripID: tripID, DayNumber: 1, ItemIndex: 0, AuthorUserID: testUserID(), Body: "b", Status: entity.CommentStatusActive},
		{ID: uuid.New(), TripID: tripID, DayNumber: 1, ItemIndex: 0, AuthorUserID: testUserID(), Body: "deleted", Status: entity.CommentStatusDeleted},
	}

	counts, err := svc.ListCommentCounts(ctxWithUserID(testUserID()), tripID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(counts) != 1 || counts[0].Count != 2 {
		t.Fatalf("expected one item with 2 active comments, got %+v", counts)
	}
}

func TestListComments_NonCollaboratorForbidden(t *testing.T) {
	repo, tripID := commentRepo(t)
	svc := newTestService(repo, &mockGenerator{})

	_, err := svc.ListComments(ctxWithUserID(uuid.New()), tripID)
	assertNotFound(t, err)
}

func TestUpdateComment_AuthorSuccess(t *testing.T) {
	repo, tripID := commentRepo(t)
	repo.collaboratorByUser = acceptedCollaborator(entity.CollaboratorRoleViewer)
	svc := newTestService(repo, &mockGenerator{})

	author := uuid.New()
	commentID := uuid.New()
	repo.comments = []entity.ItineraryComment{
		{ID: commentID, TripID: tripID, DayNumber: 1, ItemIndex: 0, AuthorUserID: author, Body: "old", Status: entity.CommentStatusActive},
	}

	info, err := svc.UpdateComment(ctxWithUserID(author), tripID, commentID, appdto.UpdateCommentInput{Body: "new body"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Comment.Body != "new body" {
		t.Fatalf("expected updated body, got %q", info.Comment.Body)
	}
}

func TestUpdateComment_NonAuthorForbidden(t *testing.T) {
	repo, tripID := commentRepo(t)
	svc := newTestService(repo, &mockGenerator{})

	commentID := uuid.New()
	// Owner (testUserID) attempts to edit a collaborator's comment.
	repo.comments = []entity.ItineraryComment{
		{ID: commentID, TripID: tripID, DayNumber: 1, ItemIndex: 0, AuthorUserID: uuid.New(), Body: "theirs", Status: entity.CommentStatusActive},
	}

	_, err := svc.UpdateComment(ctxWithUserID(testUserID()), tripID, commentID, appdto.UpdateCommentInput{Body: "hijack"})
	assertForbidden(t, err)
}

func TestUpdateComment_DeletedCommentNotFound(t *testing.T) {
	repo, tripID := commentRepo(t)
	svc := newTestService(repo, &mockGenerator{})

	commentID := uuid.New()
	repo.comments = []entity.ItineraryComment{
		{ID: commentID, TripID: tripID, DayNumber: 1, ItemIndex: 0, AuthorUserID: testUserID(), Body: "gone", Status: entity.CommentStatusDeleted},
	}

	_, err := svc.UpdateComment(ctxWithUserID(testUserID()), tripID, commentID, appdto.UpdateCommentInput{Body: "x"})
	assertNotFound(t, err)
}

func TestUpdateComment_CrossTripNotFound(t *testing.T) {
	repo, tripID := commentRepo(t)
	svc := newTestService(repo, &mockGenerator{})

	commentID := uuid.New()
	// Comment belongs to a different trip; updating via this trip path must 404.
	repo.comments = []entity.ItineraryComment{
		{ID: commentID, TripID: uuid.New(), DayNumber: 1, ItemIndex: 0, AuthorUserID: testUserID(), Body: "elsewhere", Status: entity.CommentStatusActive},
	}

	_, err := svc.UpdateComment(ctxWithUserID(testUserID()), tripID, commentID, appdto.UpdateCommentInput{Body: "x"})
	assertNotFound(t, err)
}

func TestDeleteComment_AuthorSuccess(t *testing.T) {
	repo, tripID := commentRepo(t)
	repo.collaboratorByUser = acceptedCollaborator(entity.CollaboratorRoleEditor)
	svc := newTestService(repo, &mockGenerator{})

	author := uuid.New()
	commentID := uuid.New()
	repo.comments = []entity.ItineraryComment{
		{ID: commentID, TripID: tripID, DayNumber: 1, ItemIndex: 0, AuthorUserID: author, Body: "mine", Status: entity.CommentStatusActive},
	}

	if err := svc.DeleteComment(ctxWithUserID(author), tripID, commentID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.comments[0].Status != entity.CommentStatusDeleted {
		t.Fatalf("expected comment to be soft-deleted, got %s", repo.comments[0].Status)
	}
}

func TestDeleteComment_OwnerCanDeleteCollaboratorComment(t *testing.T) {
	repo, tripID := commentRepo(t)
	svc := newTestService(repo, &mockGenerator{})

	commentID := uuid.New()
	repo.comments = []entity.ItineraryComment{
		{ID: commentID, TripID: tripID, DayNumber: 1, ItemIndex: 0, AuthorUserID: uuid.New(), Body: "collab", Status: entity.CommentStatusActive},
	}

	if err := svc.DeleteComment(ctxWithUserID(testUserID()), tripID, commentID); err != nil {
		t.Fatalf("owner should delete any comment, got %v", err)
	}
	if repo.comments[0].Status != entity.CommentStatusDeleted {
		t.Fatalf("expected comment to be soft-deleted, got %s", repo.comments[0].Status)
	}
}

func TestDeleteComment_EditorCannotDeleteOthers(t *testing.T) {
	repo, tripID := commentRepo(t)
	repo.collaboratorByUser = acceptedCollaborator(entity.CollaboratorRoleEditor)
	svc := newTestService(repo, &mockGenerator{})

	commentID := uuid.New()
	repo.comments = []entity.ItineraryComment{
		{ID: commentID, TripID: tripID, DayNumber: 1, ItemIndex: 0, AuthorUserID: uuid.New(), Body: "owner-or-other", Status: entity.CommentStatusActive},
	}

	err := svc.DeleteComment(ctxWithUserID(uuid.New()), tripID, commentID)
	assertForbidden(t, err)
}

func TestDeleteComment_ViewerCannotDeleteOthers(t *testing.T) {
	repo, tripID := commentRepo(t)
	repo.collaboratorByUser = acceptedCollaborator(entity.CollaboratorRoleViewer)
	svc := newTestService(repo, &mockGenerator{})

	commentID := uuid.New()
	repo.comments = []entity.ItineraryComment{
		{ID: commentID, TripID: tripID, DayNumber: 1, ItemIndex: 0, AuthorUserID: uuid.New(), Body: "other", Status: entity.CommentStatusActive},
	}

	err := svc.DeleteComment(ctxWithUserID(uuid.New()), tripID, commentID)
	assertForbidden(t, err)
}

func TestDeleteComment_CrossTripNotFound(t *testing.T) {
	repo, tripID := commentRepo(t)
	svc := newTestService(repo, &mockGenerator{})

	commentID := uuid.New()
	repo.comments = []entity.ItineraryComment{
		{ID: commentID, TripID: uuid.New(), DayNumber: 1, ItemIndex: 0, AuthorUserID: testUserID(), Body: "elsewhere", Status: entity.CommentStatusActive},
	}

	err := svc.DeleteComment(ctxWithUserID(testUserID()), tripID, commentID)
	assertNotFound(t, err)
}
