import type { SupportedLanguage } from "@/lib/i18n/languages";

export type UserProfile = {
  userId: string;
  displayName: string | null;
  homeCity: string | null;
  homeCountry: string | null;
  preferredCurrency: string;
  preferredLanguage: SupportedLanguage;
  createdAt: string;
  updatedAt: string;
};

export type UpdateUserProfileRequest = {
  displayName?: string | null;
  homeCity?: string | null;
  homeCountry?: string | null;
  preferredCurrency: string;
  preferredLanguage: SupportedLanguage;
};

export type TravelPace = "relaxed" | "balanced" | "intensive";

export type UserPreferences = {
  userId: string;
  travelStyles: string[];
  pace: TravelPace;
  maxWalkingKmPerDay: number | null;
  foodPreferences: string[];
  avoid: string[];
  preferredTransport: string[];
  accommodationStyle: string[];
  dietaryRestrictions: string[];
  createdAt: string;
  updatedAt: string;
};

export type PatchUserPreferencesRequest = Partial<{
  travelStyles: string[];
  pace: TravelPace;
  maxWalkingKmPerDay: number | null;
  foodPreferences: string[];
  avoid: string[];
  preferredTransport: string[];
  accommodationStyle: string[];
  dietaryRestrictions: string[];
}>;
