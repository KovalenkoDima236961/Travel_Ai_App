export const ONBOARDING_STEPS = [
  "welcome",
  "preferences",
  "choose_start",
  "create_first_trip",
  "first_trip_setup",
  "completed",
  "skipped"
] as const;

export type OnboardingStep = (typeof ONBOARDING_STEPS)[number];

export type OnboardingState = {
  onboardingCompleted: boolean;
  onboardingSkipped: boolean;
  onboardingStep: OnboardingStep;
  completedSteps: string[];
  firstTripCreatedAt: string | null;
  updatedAt: string;
};

export type StorageLike = Pick<Storage, "getItem" | "setItem" | "removeItem">;

export function onboardingStorageKey(userId: string) {
  return `onboarding:${userId}`;
}

export function createDefaultOnboardingState(now = new Date()): OnboardingState {
  return {
    onboardingCompleted: false,
    onboardingSkipped: false,
    onboardingStep: "welcome",
    completedSteps: [],
    firstTripCreatedAt: null,
    updatedAt: now.toISOString()
  };
}

export function readOnboardingState(
  userId: string,
  storage: StorageLike | null | undefined,
  now = new Date()
): OnboardingState {
  if (!userId || !storage) {
    return createDefaultOnboardingState(now);
  }

  try {
    const raw = storage.getItem(onboardingStorageKey(userId));
    if (!raw) {
      return createDefaultOnboardingState(now);
    }
    return normalizeOnboardingState(JSON.parse(raw), now);
  } catch {
    return createDefaultOnboardingState(now);
  }
}

export function writeOnboardingState(
  userId: string,
  state: OnboardingState,
  storage: StorageLike | null | undefined
) {
  if (!userId || !storage) {
    return false;
  }
  try {
    storage.setItem(onboardingStorageKey(userId), JSON.stringify(state));
    return true;
  } catch {
    return false;
  }
}

export function removeOnboardingState(
  userId: string,
  storage: StorageLike | null | undefined
) {
  if (!userId || !storage) {
    return false;
  }
  try {
    storage.removeItem(onboardingStorageKey(userId));
    return true;
  } catch {
    return false;
  }
}

export function nextOnboardingState(
  current: OnboardingState,
  patch: Partial<OnboardingState>,
  now = new Date()
): OnboardingState {
  return normalizeOnboardingState(
    {
      ...current,
      ...patch,
      completedSteps: patch.completedSteps ?? current.completedSteps,
      updatedAt: now.toISOString()
    },
    now
  );
}

export function addCompletedStep(state: OnboardingState, step: string, now = new Date()) {
  return nextOnboardingState(
    state,
    { completedSteps: Array.from(new Set([...state.completedSteps, step])) },
    now
  );
}

function normalizeOnboardingState(value: unknown, now: Date): OnboardingState {
  const fallback = createDefaultOnboardingState(now);
  if (!value || typeof value !== "object") {
    return fallback;
  }
  const candidate = value as Partial<OnboardingState>;
  const step = ONBOARDING_STEPS.includes(candidate.onboardingStep as OnboardingStep)
    ? (candidate.onboardingStep as OnboardingStep)
    : fallback.onboardingStep;
  return {
    onboardingCompleted: candidate.onboardingCompleted === true,
    onboardingSkipped: candidate.onboardingSkipped === true,
    onboardingStep: step,
    completedSteps: Array.isArray(candidate.completedSteps)
      ? Array.from(
          new Set(candidate.completedSteps.filter((item): item is string => typeof item === "string"))
        )
      : [],
    firstTripCreatedAt:
      typeof candidate.firstTripCreatedAt === "string" ? candidate.firstTripCreatedAt : null,
    updatedAt: typeof candidate.updatedAt === "string" ? candidate.updatedAt : fallback.updatedAt
  };
}
