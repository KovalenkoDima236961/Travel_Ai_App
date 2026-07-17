export const ONBOARDING_TIPS = {
  trip_health: "tripHealth",
  budget_confidence: "budgetConfidence",
  route_transport: "routeTransport",
  public_share: "publicShare",
  offline_mode: "offlineMode",
  receipts: "receipts",
  ai_generation: "aiGeneration"
} as const;

export type OnboardingTipId = keyof typeof ONBOARDING_TIPS;

export function dismissedTipsStorageKey(userId: string) {
  return `onboardingTipsDismissed:${userId}`;
}

export function readDismissedTips(userId: string, storage?: Pick<Storage, "getItem"> | null) {
  if (!userId || !storage) {
    return [] as OnboardingTipId[];
  }
  try {
    const value = JSON.parse(storage.getItem(dismissedTipsStorageKey(userId)) ?? "[]");
    if (!Array.isArray(value)) {
      return [];
    }
    return value.filter((item): item is OnboardingTipId => item in ONBOARDING_TIPS);
  } catch {
    return [];
  }
}
