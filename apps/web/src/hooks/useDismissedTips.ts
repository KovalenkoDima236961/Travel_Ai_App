"use client";

import { useCallback, useEffect, useState } from "react";
import {
  dismissedTipsStorageKey,
  readDismissedTips,
  type OnboardingTipId
} from "@/lib/onboarding/tips";

const CHANGE_EVENT = "travel-ai:dismissed-tips-change";

export function useDismissedTips(userId?: string | null) {
  const [dismissed, setDismissed] = useState<OnboardingTipId[]>([]);

  useEffect(() => {
    if (!userId) {
      setDismissed([]);
      return;
    }
    const sync = () => setDismissed(readDismissedTips(userId, window.localStorage));
    sync();
    window.addEventListener(CHANGE_EVENT, sync);
    window.addEventListener("storage", sync);
    return () => {
      window.removeEventListener(CHANGE_EVENT, sync);
      window.removeEventListener("storage", sync);
    };
  }, [userId]);

  const dismiss = useCallback(
    (tipId: OnboardingTipId) => {
      if (!userId || typeof window === "undefined") {
        return;
      }
      const next = Array.from(new Set([...readDismissedTips(userId, window.localStorage), tipId]));
      try {
        window.localStorage.setItem(dismissedTipsStorageKey(userId), JSON.stringify(next));
      } catch {
        // Dismissal is convenience state; blocked storage must not block the feature.
      }
      setDismissed(next);
      window.dispatchEvent(new Event(CHANGE_EVENT));
    },
    [userId]
  );

  const clear = useCallback(() => {
    if (!userId || typeof window === "undefined") {
      return;
    }
    try {
      window.localStorage.removeItem(dismissedTipsStorageKey(userId));
    } catch {
      // Keep settings usable when storage is unavailable.
    }
    setDismissed([]);
    window.dispatchEvent(new Event(CHANGE_EVENT));
  }, [userId]);

  return {
    dismissed,
    isDismissed: (tipId: OnboardingTipId) => dismissed.includes(tipId),
    dismiss,
    clear
  };
}
