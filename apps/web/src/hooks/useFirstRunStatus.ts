import type { OnboardingState } from "@/lib/onboarding/state";

export function getFirstRunStatus(tripCount: number, onboarding: OnboardingState) {
  const hasTrips = tripCount > 0;
  const onboardingActive = !onboarding.onboardingCompleted && !onboarding.onboardingSkipped;
  return {
    hasTrips,
    onboardingActive,
    showFirstRunDashboard: !hasTrips && onboardingActive,
    showHelpfulEmptyState: !hasTrips && !onboardingActive
  };
}

export function useFirstRunStatus(tripCount: number, onboarding: OnboardingState) {
  return getFirstRunStatus(tripCount, onboarding);
}
