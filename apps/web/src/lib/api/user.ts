import { apiFetch } from "@/lib/api/client";
import { getUserApiBaseUrl } from "@/lib/config";
import type {
  PatchUserPreferencesRequest,
  UpdateUserProfileRequest,
  UserPreferences,
  UserProfile
} from "@/types/user";

export const userKeys = {
  all: ["user"] as const,
  profile: () => [...userKeys.all, "profile"] as const,
  preferences: () => [...userKeys.all, "preferences"] as const
};

export function getMyProfile() {
  return userFetch<UserProfile>("/users/me/profile");
}

export function updateMyProfile(data: UpdateUserProfileRequest) {
  return userFetch<UserProfile>("/users/me/profile", {
    method: "PUT",
    body: JSON.stringify(cleanProfilePayload(data))
  });
}

export function getMyPreferences() {
  return userFetch<UserPreferences>("/users/me/preferences");
}

export function patchMyPreferences(data: PatchUserPreferencesRequest) {
  return userFetch<UserPreferences>("/users/me/preferences", {
    method: "PATCH",
    body: JSON.stringify(data)
  });
}

function userFetch<T>(path: string, init: RequestInit = {}) {
  return apiFetch<T>(path, init, {
    baseUrl: getUserApiBaseUrl(),
    serviceName: "User Service"
  });
}

function cleanProfilePayload(data: UpdateUserProfileRequest): UpdateUserProfileRequest {
  return {
    displayName: cleanOptionalText(data.displayName),
    homeCity: cleanOptionalText(data.homeCity),
    homeCountry: cleanOptionalText(data.homeCountry),
    preferredCurrency: data.preferredCurrency.trim().toUpperCase(),
    preferredLanguage: data.preferredLanguage.trim()
  };
}

function cleanOptionalText(value: string | null | undefined) {
  const trimmed = value?.trim() ?? "";
  return trimmed.length > 0 ? trimmed : null;
}
