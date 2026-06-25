package emailnotifications

import (
	"testing"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
)

func defaultAllowlist() []string {
	return []string{
		notifications.TypeCollaborationInvited,
		notifications.TypeCommentCreated,
		notifications.TypeCollaboratorRoleChange,
		notifications.TypeCollaboratorRemoved,
	}
}

func TestShouldSendEmail(t *testing.T) {
	recipient := uuid.New()
	actor := uuid.New()

	enabled := NewPolicy(true, defaultAllowlist())
	disabled := NewPolicy(false, defaultAllowlist())

	t.Run("disabled globally is false", func(t *testing.T) {
		n := entity.Notification{UserID: recipient, Type: notifications.TypeCommentCreated}
		if disabled.ShouldSendEmail(n) {
			t.Fatal("expected false when email disabled")
		}
	})

	t.Run("type not in allowlist is false", func(t *testing.T) {
		n := entity.Notification{UserID: recipient, Type: notifications.TypeItineraryUpdated}
		if enabled.ShouldSendEmail(n) {
			t.Fatal("expected false for non-allowlisted type")
		}
	})

	t.Run("self notification is false", func(t *testing.T) {
		n := entity.Notification{UserID: recipient, ActorUserID: &recipient, Type: notifications.TypeCommentCreated}
		if enabled.ShouldSendEmail(n) {
			t.Fatal("expected false when recipient is the actor")
		}
	})

	t.Run("nil recipient is false", func(t *testing.T) {
		n := entity.Notification{UserID: uuid.Nil, Type: notifications.TypeCommentCreated}
		if enabled.ShouldSendEmail(n) {
			t.Fatal("expected false for nil recipient")
		}
	})

	t.Run("allowlisted comment_created is true", func(t *testing.T) {
		n := entity.Notification{UserID: recipient, ActorUserID: &actor, Type: notifications.TypeCommentCreated}
		if !enabled.ShouldSendEmail(n) {
			t.Fatal("expected true for allowlisted comment_created to a non-actor")
		}
	})
}
