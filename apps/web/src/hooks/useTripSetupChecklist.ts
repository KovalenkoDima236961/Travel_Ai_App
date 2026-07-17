"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useAuth } from "@/components/auth/AuthProvider";
import { useOnboardingState } from "@/hooks/useOnboardingState";
import {
  buildTripSetupChecklist,
  completedTripSetupCount,
  tripSetupDismissalKey,
  type TripSetupInput
} from "@/lib/onboarding/trip-setup";

export function useTripSetupChecklist(input: TripSetupInput) {
  const { user } = useAuth();
  const onboarding = useOnboardingState(user?.id);
  const [dismissed, setDismissed] = useState(false);
  const items = useMemo(() => buildTripSetupChecklist(input), [input]);
  const completedCount = completedTripSetupCount(items);

  useEffect(() => {
    if (!user?.id || typeof window === "undefined") {
      setDismissed(false);
      return;
    }
    try {
      setDismissed(window.localStorage.getItem(tripSetupDismissalKey(user.id, input.trip.id)) === "true");
    } catch {
      setDismissed(false);
    }
  }, [input.trip.id, user?.id]);

  useEffect(() => {
    const createdAt = new Date(input.trip.createdAt).getTime();
    const wasJustCreated = Number.isFinite(createdAt) && Date.now() - createdAt < 15 * 60 * 1000;
    if (
      onboarding.hydrated &&
      !onboarding.state.onboardingCompleted &&
      !onboarding.state.onboardingSkipped &&
      !onboarding.state.firstTripCreatedAt &&
      wasJustCreated
    ) {
      onboarding.markFirstTripCreated(input.trip.createdAt);
    }
  }, [input.trip.createdAt, onboarding]);

  const dismiss = useCallback(() => {
    setDismissed(true);
    if (!user?.id || typeof window === "undefined") {
      return;
    }
    try {
      window.localStorage.setItem(tripSetupDismissalKey(user.id, input.trip.id), "true");
    } catch {
      // UI-state persistence is best-effort.
    }
  }, [input.trip.id, user?.id]);

  const firstTripCreatedAt = onboarding.state.firstTripCreatedAt;
  const isFirstTrip = firstTripCreatedAt
    ? Math.abs(new Date(input.trip.createdAt).getTime() - new Date(firstTripCreatedAt).getTime()) < 5 * 60 * 1000
    : false;
  const show =
    onboarding.hydrated &&
    !onboarding.state.onboardingCompleted &&
    !onboarding.state.onboardingSkipped &&
    isFirstTrip &&
    !dismissed &&
    completedCount < 5;

  return { items, completedCount, dismissed, show, dismiss };
}
