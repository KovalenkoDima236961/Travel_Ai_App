import { describe, expect, it } from "vitest";
import {
  addCompletedStep,
  createDefaultOnboardingState,
  nextOnboardingState,
  onboardingStorageKey,
  readOnboardingState,
  writeOnboardingState
} from "@/lib/onboarding/state";
import { getFirstRunStatus } from "@/hooks/useFirstRunStatus";
import { dismissedTipsStorageKey, readDismissedTips } from "@/lib/onboarding/tips";

describe("onboarding local state", () => {
  it("uses user-scoped keys and never crosses accounts", () => {
    const storage = memoryStorage();
    const state = nextOnboardingState(createDefaultOnboardingState(), {
      onboardingStep: "preferences"
    });
    expect(writeOnboardingState("user-a", state, storage)).toBe(true);
    expect(storage.getItem(onboardingStorageKey("user-b"))).toBeNull();
    expect(readOnboardingState("user-a", storage).onboardingStep).toBe("preferences");
    expect(readOnboardingState("user-b", storage).onboardingStep).toBe("welcome");
  });

  it("recovers from unavailable or invalid storage without crashing", () => {
    const broken = {
      getItem: () => { throw new Error("blocked"); },
      setItem: () => { throw new Error("blocked"); },
      removeItem: () => { throw new Error("blocked"); }
    };
    expect(readOnboardingState("user-a", broken).onboardingStep).toBe("welcome");
    expect(writeOnboardingState("user-a", createDefaultOnboardingState(), broken)).toBe(false);
  });

  it("de-duplicates completed steps and controls first-run visibility", () => {
    const active = addCompletedStep(addCompletedStep(createDefaultOnboardingState(), "welcome"), "welcome");
    expect(active.completedSteps).toEqual(["welcome"]);
    expect(getFirstRunStatus(0, active).showFirstRunDashboard).toBe(true);
    expect(getFirstRunStatus(1, active).showFirstRunDashboard).toBe(false);
    expect(getFirstRunStatus(0, { ...active, onboardingCompleted: true }).showHelpfulEmptyState).toBe(true);
  });

  it("stores dismissed tips per user", () => {
    const storage = memoryStorage();
    storage.setItem(dismissedTipsStorageKey("user-a"), JSON.stringify(["trip_health"]));
    expect(readDismissedTips("user-a", storage)).toEqual(["trip_health"]);
    expect(readDismissedTips("user-b", storage)).toEqual([]);
  });
});

function memoryStorage() {
  const values = new Map<string, string>();
  return {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => { values.set(key, value); },
    removeItem: (key: string) => { values.delete(key); }
  };
}
