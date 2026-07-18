import { apiFetch } from "@/shared/api/client";
import type { PublicFeatureFlagsResponse } from "./types";

export const featureFlagKeys = {
  public: ["feature-flags", "public"] as const
};

export function getPublicFeatureFlags(): Promise<PublicFeatureFlagsResponse> {
  return apiFetch<PublicFeatureFlagsResponse>("/feature-flags/public");
}
