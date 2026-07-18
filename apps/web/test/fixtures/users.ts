import type { UserPreferences, UserProfile } from "@/entities/user/model";
import type { AuthUser } from "@/shared/api/auth";
import type { UserPreferencesContract, UserProfileContract } from "@/lib/api/contracts";

export const TEST_USER_IDS = {
  owner: "10000000-0000-4000-8000-000000000001",
  editor: "10000000-0000-4000-8000-000000000002",
  viewer: "10000000-0000-4000-8000-000000000003",
  outsider: "10000000-0000-4000-8000-000000000004"
} as const;

export const ownerAuthUser: AuthUser = {
  id: TEST_USER_IDS.owner,
  email: "owner@example.test",
  createdAt: "2026-01-15T09:00:00Z"
};

export const editorAuthUser: AuthUser = {
  id: TEST_USER_IDS.editor,
  email: "editor@example.test",
  createdAt: "2026-01-15T09:01:00Z"
};

export const viewerAuthUser: AuthUser = {
  id: TEST_USER_IDS.viewer,
  email: "viewer@example.test",
  createdAt: "2026-01-15T09:02:00Z"
};

export const ownerProfile: UserProfile = {
  userId: TEST_USER_IDS.owner,
  displayName: "Test Owner",
  homeCity: "Bratislava",
  homeCountry: "Slovakia",
  preferredCurrency: "EUR",
  preferredLanguage: "en",
  createdAt: "2026-01-15T09:00:00Z",
  updatedAt: "2026-01-15T09:00:00Z"
};

ownerProfile satisfies UserProfileContract;

export const ownerPreferences: UserPreferences = {
  userId: TEST_USER_IDS.owner,
  travelStyles: ["food", "culture"],
  pace: "balanced",
  maxWalkingKmPerDay: 8,
  foodPreferences: ["local"],
  avoid: [],
  preferredTransport: ["train", "public_transport"],
  accommodationStyle: ["apartment"],
  dietaryRestrictions: [],
  createdAt: "2026-01-15T09:00:00Z",
  updatedAt: "2026-01-15T09:00:00Z"
};

ownerPreferences satisfies UserPreferencesContract;

export const workspaceRoles = {
  owner: { userId: TEST_USER_IDS.owner, role: "owner" },
  admin: { userId: TEST_USER_IDS.editor, role: "admin" },
  member: { userId: TEST_USER_IDS.viewer, role: "member" },
  viewer: { userId: TEST_USER_IDS.outsider, role: "viewer" }
} as const;
