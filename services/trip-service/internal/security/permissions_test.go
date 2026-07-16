package security

import "testing"

func TestAuthorizeDenyByDefaultAndRoleBoundaries(t *testing.T) {
	tests := []struct {
		name       string
		ctx        TripAccessContext
		permission TripPermission
		allowed    bool
	}{
		{"owner edits", authenticated("owner"), PermissionTripEdit, true},
		{"owner is not ops admin", authenticated("owner"), PermissionOpsView, false},
		{"editor edits itinerary", authenticated("editor"), PermissionItineraryEdit, true},
		{"editor cannot share", authenticated("editor"), PermissionShareManage, false},
		{"viewer reads receipts", authenticated("viewer"), PermissionReceiptsView, true},
		{"viewer cannot edit route", authenticated("viewer"), PermissionRouteEdit, false},
		{"pending collaborator denied", TripAccessContext{Principal: Principal{Type: PrincipalAuthenticatedUser}, Role: "editor"}, PermissionTripView, false},
		{"removed collaborator denied", TripAccessContext{Principal: Principal{Type: PrincipalAuthenticatedUser}, Role: "owner", Accepted: true, Removed: true}, PermissionTripView, false},
		{"public reads sanitized itinerary", TripAccessContext{Principal: Principal{Type: PrincipalPublicShare}, PublicShareLive: true}, PermissionItineraryView, true},
		{"public cannot read expenses", TripAccessContext{Principal: Principal{Type: PrincipalPublicShare}, PublicShareLive: true}, PermissionExpensesView, false},
		{"ops admin reads ops", TripAccessContext{Principal: Principal{Type: PrincipalOpsAdmin}}, PermissionOpsView, true},
		{"unknown permission denied", authenticated("viewer"), TripPermission("unknown"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Authorize(tt.ctx, tt.permission); got.Allowed != tt.allowed {
				t.Fatalf("allowed=%v reason=%s, want %v", got.Allowed, got.Reason, tt.allowed)
			}
		})
	}
}

func authenticated(role string) TripAccessContext {
	return TripAccessContext{Principal: Principal{Type: PrincipalAuthenticatedUser}, Role: role, Accepted: true}
}
