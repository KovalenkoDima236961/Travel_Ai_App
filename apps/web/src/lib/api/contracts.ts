/**
 * Stable aliases over generated OpenAPI schemas.
 *
 * Keep UI-only form/view models outside this file. Generated files are replaced
 * by scripts/contracts/generate-web-client.sh and must not be edited directly.
 */
import type { components as AuthComponents } from "@/lib/api/generated/auth/schema";
import type { components as NotificationComponents } from "@/lib/api/generated/notifications/schema";
import type { components as TripComponents } from "@/lib/api/generated/trips/schema";
import type { components as UserComponents } from "@/lib/api/generated/user/schema";

export type AuthUserContract = AuthComponents["schemas"]["AuthUser"];
export type AuthResponseContract = AuthComponents["schemas"]["AuthResponse"];
export type TokenResponseContract = AuthComponents["schemas"]["TokenResponse"];

export type UserProfileContract = UserComponents["schemas"]["UserProfile"];
export type UpdateUserProfileContract = UserComponents["schemas"]["UpdateUserProfileRequest"];
export type UserPreferencesContract = UserComponents["schemas"]["UserPreferences"];
export type PatchUserPreferencesContract = UserComponents["schemas"]["PatchUserPreferencesRequest"];

export type TripContract = TripComponents["schemas"]["Trip"];
export type TripsListContract = TripComponents["schemas"]["TripsListResponse"];
export type CreateTripContract = TripComponents["schemas"]["CreateTripRequest"];
export type UpdateItineraryContract = TripComponents["schemas"]["UpdateItineraryRequest"];
export type ExpectedRevisionContract = TripComponents["schemas"]["ExpectedRevisionRequest"];
export type BudgetSummaryContract = TripComponents["schemas"]["BudgetSummary"];
export type ExpenseContract = TripComponents["schemas"]["Expense"];
export type PublicTripContract = TripComponents["schemas"]["PublicTripResponse"];

export type NotificationContract = NotificationComponents["schemas"]["Notification"];
export type NotificationsResponseContract = NotificationComponents["schemas"]["NotificationsResponse"];
export type UnreadNotificationsContract = NotificationComponents["schemas"]["UnreadNotificationsResponse"];
