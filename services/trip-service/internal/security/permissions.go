// Package security contains the deny-by-default authorization policy used by
// Trip Service. Resource handlers resolve identity/membership once and ask this
// package for a concrete permission instead of duplicating role checks.
package security

type PrincipalType string

const (
	PrincipalAuthenticatedUser PrincipalType = "authenticated_user"
	PrincipalPublicShare       PrincipalType = "public_share"
	PrincipalInternalService   PrincipalType = "internal_service"
	PrincipalOpsAdmin          PrincipalType = "ops_admin"
)

type TripPermission string

const (
	PermissionTripView            TripPermission = "trip:view"
	PermissionTripEdit            TripPermission = "trip:edit"
	PermissionTripDelete          TripPermission = "trip:delete"
	PermissionItineraryView       TripPermission = "itinerary:view"
	PermissionItineraryEdit       TripPermission = "itinerary:edit"
	PermissionRouteView           TripPermission = "route:view"
	PermissionRouteEdit           TripPermission = "route:edit"
	PermissionBudgetView          TripPermission = "budget:view"
	PermissionBudgetEdit          TripPermission = "budget:edit"
	PermissionExpensesView        TripPermission = "expenses:view"
	PermissionExpensesEdit        TripPermission = "expenses:edit"
	PermissionReceiptsView        TripPermission = "receipts:view"
	PermissionReceiptsUpload      TripPermission = "receipts:upload"
	PermissionReceiptsDelete      TripPermission = "receipts:delete"
	PermissionCommentsView        TripPermission = "comments:view"
	PermissionCommentsCreate      TripPermission = "comments:create"
	PermissionActivityView        TripPermission = "activity:view"
	PermissionShareManage         TripPermission = "share:manage"
	PermissionCollaboratorsManage TripPermission = "collaborators:manage"
	PermissionApprovalView        TripPermission = "approval:view"
	PermissionApprovalAct         TripPermission = "approval:act"
	PermissionPolicyView          TripPermission = "policy:view"
	PermissionPolicyEdit          TripPermission = "policy:edit"
	PermissionHealthView          TripPermission = "health:view"
	PermissionCommandCenterView   TripPermission = "command_center:view"
	PermissionGroupReadinessView  TripPermission = "group_readiness:view"
	PermissionGroupReadinessNudge TripPermission = "group_readiness:nudge"
	PermissionOpsView             TripPermission = "ops:view"
	PermissionOpsAct              TripPermission = "ops:act"
)

type Principal struct {
	Type PrincipalType
}

type TripAccessContext struct {
	Principal       Principal
	Role            string
	WorkspaceRole   string
	Accepted        bool
	Removed         bool
	PublicShareLive bool
}

type AccessDecision struct {
	Allowed bool
	Reason  string
}

var viewPermissions = map[TripPermission]struct{}{
	PermissionTripView: {}, PermissionItineraryView: {}, PermissionRouteView: {},
	PermissionBudgetView: {}, PermissionExpensesView: {}, PermissionReceiptsView: {},
	PermissionCommentsView: {}, PermissionActivityView: {}, PermissionApprovalView: {},
	PermissionPolicyView: {}, PermissionHealthView: {}, PermissionCommandCenterView: {},
	PermissionGroupReadinessView: {},
}

var editPermissions = map[TripPermission]struct{}{
	PermissionTripEdit: {}, PermissionItineraryEdit: {}, PermissionRouteEdit: {},
	PermissionBudgetEdit: {}, PermissionExpensesEdit: {}, PermissionReceiptsUpload: {},
	PermissionReceiptsDelete: {}, PermissionCommentsCreate: {}, PermissionApprovalAct: {},
	PermissionGroupReadinessNudge: {},
}

// Authorize applies a closed permission list. Unknown principals, roles,
// permissions, pending/removed collaborators, and inactive public shares are
// denied by default.
func Authorize(ctx TripAccessContext, permission TripPermission) AccessDecision {
	switch ctx.Principal.Type {
	case PrincipalOpsAdmin:
		if permission == PermissionOpsView || permission == PermissionOpsAct {
			return AccessDecision{Allowed: true, Reason: "ops_admin"}
		}
		return AccessDecision{Reason: "ops_scope_only"}
	case PrincipalInternalService:
		return AccessDecision{Reason: "internal_endpoint_only"}
	case PrincipalPublicShare:
		if !ctx.PublicShareLive {
			return AccessDecision{Reason: "share_inactive"}
		}
		if permission == PermissionTripView || permission == PermissionItineraryView || permission == PermissionRouteView {
			return AccessDecision{Allowed: true, Reason: "sanitized_public_share"}
		}
		return AccessDecision{Reason: "private_resource"}
	case PrincipalAuthenticatedUser:
		if ctx.Removed || !ctx.Accepted {
			return AccessDecision{Reason: "membership_inactive"}
		}
	default:
		return AccessDecision{Reason: "unknown_principal"}
	}

	switch ctx.Role {
	case "owner":
		if permission == PermissionOpsView || permission == PermissionOpsAct {
			return AccessDecision{Reason: "ops_admin_only"}
		}
		return AccessDecision{Allowed: true, Reason: "owner"}
	case "editor":
		if _, ok := viewPermissions[permission]; ok {
			return AccessDecision{Allowed: true, Reason: "editor_read"}
		}
		if _, ok := editPermissions[permission]; ok {
			return AccessDecision{Allowed: true, Reason: "editor_write"}
		}
		return AccessDecision{Reason: "owner_only"}
	case "viewer":
		if _, ok := viewPermissions[permission]; ok {
			return AccessDecision{Allowed: true, Reason: "viewer_read"}
		}
		// Existing product behavior allows accepted viewers to contribute a
		// comment and upload/delete only their own receipt. Ownership is checked
		// separately by the receipt/comment services.
		if permission == PermissionCommentsCreate || permission == PermissionReceiptsUpload {
			return AccessDecision{Allowed: true, Reason: "viewer_contribution"}
		}
		return AccessDecision{Reason: "read_only"}
	default:
		return AccessDecision{Reason: "unknown_role"}
	}
}
