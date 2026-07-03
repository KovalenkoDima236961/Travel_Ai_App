"use client";

import { useCallback, useEffect, useState } from "react";
import {
  type InstallPlatform,
  getInstallPlatform,
  isInstallPromptDismissedRecently,
  isStandaloneMode,
  recordPwaVisit,
  storeInstallPromptDismissal
} from "@/lib/pwa/pwa-detection";

type BeforeInstallPromptChoice = {
  outcome: "accepted" | "dismissed";
  platform?: string;
};

type BeforeInstallPromptEvent = Event & {
  platforms?: string[];
  prompt: () => Promise<void>;
  userChoice: Promise<BeforeInstallPromptChoice>;
};

let sharedDeferredPrompt: BeforeInstallPromptEvent | null = null;
const sharedPromptListeners = new Set<() => void>();

export type PwaInstallResult = "accepted" | "dismissed" | "unavailable";

export type PwaInstallState = {
  canInstall: boolean;
  isInstalled: boolean;
  platform: InstallPlatform;
  installPromptAvailable: boolean;
  dismissedRecently: boolean;
  promptInstall: () => Promise<PwaInstallResult>;
  dismissInstallPrompt: () => void;
};

export function usePwaInstall(): PwaInstallState {
  const [installPromptAvailable, setInstallPromptAvailable] = useState(
    () => Boolean(sharedDeferredPrompt)
  );
  const [isInstalled, setIsInstalled] = useState(false);
  const [platform, setPlatform] = useState<InstallPlatform>("unsupported");
  const [dismissedRecently, setDismissedRecently] = useState(false);

  const refreshState = useCallback(() => {
    const installed = isStandaloneMode();
    setIsInstalled(installed);
    setPlatform(installed ? "installed" : getInstallPlatform());
    setDismissedRecently(isInstallPromptDismissedRecently());
  }, []);

  useEffect(() => {
    recordPwaVisit();
    refreshState();
    setInstallPromptAvailable(Boolean(sharedDeferredPrompt));

    function handleSharedPromptChanged() {
      setInstallPromptAvailable(Boolean(sharedDeferredPrompt));
    }

    function handleBeforeInstallPrompt(event: Event) {
      event.preventDefault();
      sharedDeferredPrompt = event as BeforeInstallPromptEvent;
      notifySharedPromptListeners();
      setPlatform(isStandaloneMode() ? "installed" : "chromium");
    }

    function handleAppInstalled() {
      sharedDeferredPrompt = null;
      notifySharedPromptListeners();
      setIsInstalled(true);
      setPlatform("installed");
    }

    const mediaQueries = [
      window.matchMedia("(display-mode: standalone)"),
      window.matchMedia("(display-mode: fullscreen)"),
      window.matchMedia("(display-mode: minimal-ui)")
    ];

    sharedPromptListeners.add(handleSharedPromptChanged);
    window.addEventListener("beforeinstallprompt", handleBeforeInstallPrompt);
    window.addEventListener("appinstalled", handleAppInstalled);
    mediaQueries.forEach((mediaQuery) => {
      mediaQuery.addEventListener("change", refreshState);
    });

    return () => {
      sharedPromptListeners.delete(handleSharedPromptChanged);
      window.removeEventListener("beforeinstallprompt", handleBeforeInstallPrompt);
      window.removeEventListener("appinstalled", handleAppInstalled);
      mediaQueries.forEach((mediaQuery) => {
        mediaQuery.removeEventListener("change", refreshState);
      });
    };
  }, [refreshState]);

  const dismissInstallPrompt = useCallback(() => {
    storeInstallPromptDismissal();
    setDismissedRecently(true);
  }, []);

  const promptInstall = useCallback(async (): Promise<PwaInstallResult> => {
    const promptEvent = sharedDeferredPrompt;
    if (!promptEvent || isStandaloneMode()) {
      return "unavailable";
    }

    await promptEvent.prompt();
    const choice = await promptEvent.userChoice;
    sharedDeferredPrompt = null;
    notifySharedPromptListeners();

    if (choice.outcome === "accepted") {
      setIsInstalled(true);
      setPlatform("installed");
      return "accepted";
    }

    dismissInstallPrompt();
    return "dismissed";
  }, [dismissInstallPrompt]);

  return {
    canInstall:
      !isInstalled &&
      !dismissedRecently &&
      (installPromptAvailable || platform === "ios_safari"),
    isInstalled,
    platform,
    installPromptAvailable,
    dismissedRecently,
    promptInstall,
    dismissInstallPrompt
  };
}

function notifySharedPromptListeners() {
  sharedPromptListeners.forEach((listener) => listener());
}
