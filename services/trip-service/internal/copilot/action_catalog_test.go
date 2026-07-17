package copilot

import (
	"testing"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
)

func TestAvailableActionsFiltersViewerMutations(t *testing.T) {
	actions := AvailableActions(uuid.New(), service.TripAccess{Level: service.AccessLevelViewer}, ClientContext{})
	for _, action := range actions {
		if action.Type == "find_transport" || action.Type == "add_expense" || action.Type == "open_share_settings" {
			t.Fatalf("viewer received edit action %q", action.Type)
		}
	}
	if _, ok := actionByType(actions, "open_trip_health"); !ok {
		t.Fatal("viewer should retain health navigation")
	}
}

func TestAvailableActionsAllowsOwnerShareSettings(t *testing.T) {
	actions := AvailableActions(uuid.New(), service.TripAccess{Level: service.AccessLevelOwner}, ClientContext{})
	if _, ok := actionByType(actions, "open_share_settings"); !ok {
		t.Fatal("owner should receive share settings navigation")
	}
}
