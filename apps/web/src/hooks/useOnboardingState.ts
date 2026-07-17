"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  addCompletedStep,
  createDefaultOnboardingState,
  nextOnboardingState,
  readOnboardingState,
  removeOnboardingState,
  writeOnboardingState,
  type OnboardingState,
  type OnboardingStep
} from "@/lib/onboarding/state";

const CHANGE_EVENT = "travel-ai:onboarding-change";

export function useOnboardingState(userId?: string | null) {
  const [state, setState] = useState<OnboardingState>(() => createDefaultOnboardingState());
  const stateRef = useRef(state);
  const [hydrated, setHydrated] = useState(false);

  useEffect(() => {
    if (!userId) {
      const fresh = createDefaultOnboardingState();
      stateRef.current = fresh;
      setState(fresh);
      setHydrated(true);
      return;
    }

    const sync = () => {
      const stored = readOnboardingState(userId, window.localStorage);
      stateRef.current = stored;
      setState(stored);
    };
    sync();
    setHydrated(true);
    window.addEventListener(CHANGE_EVENT, sync);
    window.addEventListener("storage", sync);
    return () => {
      window.removeEventListener(CHANGE_EVENT, sync);
      window.removeEventListener("storage", sync);
    };
  }, [userId]);

  const commit = useCallback(
    (updater: (current: OnboardingState) => OnboardingState) => {
      const updated = updater(stateRef.current);
      stateRef.current = updated;
      setState(updated);
      if (userId && typeof window !== "undefined") {
        writeOnboardingState(userId, updated, window.localStorage);
        window.dispatchEvent(new Event(CHANGE_EVENT));
      }
    },
    [userId]
  );

  const goToStep = useCallback(
    (step: OnboardingStep) => commit((current) => nextOnboardingState(current, { onboardingStep: step })),
    [commit]
  );

  const markStepComplete = useCallback(
    (step: string, nextStep?: OnboardingStep) =>
      commit((current) =>
        nextOnboardingState(addCompletedStep(current, step), {
          ...(nextStep ? { onboardingStep: nextStep } : {})
        })
      ),
    [commit]
  );

  const markFirstTripCreated = useCallback(
    (createdAt = new Date().toISOString()) =>
      commit((current) =>
        nextOnboardingState(addCompletedStep(current, "first_trip_created"), {
          firstTripCreatedAt: current.firstTripCreatedAt ?? createdAt,
          onboardingStep: "first_trip_setup"
        })
      ),
    [commit]
  );

  const complete = useCallback(
    () =>
      commit((current) =>
        nextOnboardingState(addCompletedStep(current, "completed"), {
          onboardingCompleted: true,
          onboardingSkipped: false,
          onboardingStep: "completed"
        })
      ),
    [commit]
  );

  const skip = useCallback(
    () =>
      commit((current) =>
        nextOnboardingState(addCompletedStep(current, "skipped"), {
          onboardingCompleted: false,
          onboardingSkipped: true,
          onboardingStep: "skipped"
        })
      ),
    [commit]
  );

  const restart = useCallback(() => {
    const fresh = createDefaultOnboardingState();
    stateRef.current = fresh;
    setState(fresh);
    if (userId && typeof window !== "undefined") {
      removeOnboardingState(userId, window.localStorage);
      writeOnboardingState(userId, fresh, window.localStorage);
      window.dispatchEvent(new Event(CHANGE_EVENT));
    }
  }, [userId]);

  return {
    state,
    hydrated,
    goToStep,
    markStepComplete,
    markFirstTripCreated,
    complete,
    skip,
    restart
  };
}
