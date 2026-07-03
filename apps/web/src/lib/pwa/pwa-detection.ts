export type InstallPlatform = "chromium" | "ios_safari" | "unsupported" | "installed";

export const PWA_INSTALL_DISMISSED_AT_KEY = "pwa_install_prompt_dismissed_at";
export const PWA_INSTALL_DISMISS_MS = 7 * 24 * 60 * 60 * 1000;
export const PWA_ENGAGEMENT_AT_KEY = "pwa_meaningful_engagement_at";
export const PWA_VISIT_COUNT_KEY = "pwa_visit_count";
const PWA_VISIT_SESSION_KEY = "pwa_visit_recorded";

type NavigatorLike = Partial<Navigator> & {
  standalone?: boolean;
  maxTouchPoints?: number;
};

type PwaRuntime = {
  matchMedia?: (query: string) => { matches: boolean };
  navigator?: NavigatorLike;
  userAgent?: string;
  platform?: string;
  localStorage?: Pick<Storage, "getItem" | "setItem">;
  sessionStorage?: Pick<Storage, "getItem" | "setItem">;
  beforeInstallPromptSupported?: boolean;
  now?: number;
};

export function isStandaloneMode(runtime: PwaRuntime = browserRuntime()): boolean {
  const matchMedia = runtime.matchMedia;
  const displayModeStandalone = Boolean(matchMedia?.("(display-mode: standalone)").matches);
  const displayModeFullscreen = Boolean(matchMedia?.("(display-mode: fullscreen)").matches);
  const displayModeMinimalUi = Boolean(matchMedia?.("(display-mode: minimal-ui)").matches);
  const iosStandalone = runtime.navigator?.standalone === true;

  return displayModeStandalone || displayModeFullscreen || displayModeMinimalUi || iosStandalone;
}

export function isIOS(runtime: PwaRuntime = browserRuntime()): boolean {
  const userAgent = runtime.userAgent ?? runtime.navigator?.userAgent ?? "";
  const platform = runtime.platform ?? runtime.navigator?.platform ?? "";

  return (
    /iPad|iPhone|iPod/.test(userAgent) ||
    (platform === "MacIntel" && Number(runtime.navigator?.maxTouchPoints ?? 0) > 1)
  );
}

export function isIOSSafari(runtime: PwaRuntime = browserRuntime()): boolean {
  const userAgent = runtime.userAgent ?? runtime.navigator?.userAgent ?? "";

  return (
    isIOS(runtime) &&
    /Safari/i.test(userAgent) &&
    !/(CriOS|FxiOS|EdgiOS|OPiOS|Chrome|Chromium|Android)/i.test(userAgent)
  );
}

export function isAndroid(runtime: PwaRuntime = browserRuntime()): boolean {
  const userAgent = runtime.userAgent ?? runtime.navigator?.userAgent ?? "";
  return /Android/i.test(userAgent);
}

export function supportsBeforeInstallPrompt(runtime: PwaRuntime = browserRuntime()): boolean {
  if (typeof runtime.beforeInstallPromptSupported === "boolean") {
    return runtime.beforeInstallPromptSupported;
  }

  return typeof window !== "undefined" && "onbeforeinstallprompt" in window;
}

export function supportsServiceWorker(runtime: PwaRuntime = browserRuntime()): boolean {
  return Boolean(runtime.navigator && "serviceWorker" in runtime.navigator);
}

export function getInstallPlatform(runtime: PwaRuntime = browserRuntime()): InstallPlatform {
  if (isStandaloneMode(runtime)) {
    return "installed";
  }
  if (supportsBeforeInstallPrompt(runtime)) {
    return "chromium";
  }
  if (isIOSSafari(runtime)) {
    return "ios_safari";
  }
  return "unsupported";
}

export function isInstallPromptDismissedRecently(
  runtime: PwaRuntime = browserRuntime()
): boolean {
  const storage = runtime.localStorage;
  if (!storage) {
    return false;
  }

  const value = storage.getItem(PWA_INSTALL_DISMISSED_AT_KEY);
  const dismissedAt = value ? Number(value) : Number.NaN;
  if (!Number.isFinite(dismissedAt)) {
    return false;
  }

  return (runtime.now ?? Date.now()) - dismissedAt < PWA_INSTALL_DISMISS_MS;
}

export function storeInstallPromptDismissal(runtime: PwaRuntime = browserRuntime()): void {
  try {
    runtime.localStorage?.setItem(
      PWA_INSTALL_DISMISSED_AT_KEY,
      String(runtime.now ?? Date.now())
    );
  } catch {
    // Dismissal tracking is best-effort.
  }
}

export function recordPwaVisit(runtime: PwaRuntime = browserRuntime()): void {
  const { localStorage, sessionStorage } = runtime;
  if (!localStorage || !sessionStorage) {
    return;
  }

  try {
    if (sessionStorage.getItem(PWA_VISIT_SESSION_KEY)) {
      return;
    }
    const current = Number(localStorage.getItem(PWA_VISIT_COUNT_KEY) ?? "0");
    localStorage.setItem(PWA_VISIT_COUNT_KEY, String((Number.isFinite(current) ? current : 0) + 1));
    sessionStorage.setItem(PWA_VISIT_SESSION_KEY, "1");
  } catch {
    // Visit count is only an install prompt signal.
  }
}

export function recordPwaEngagement(runtime: PwaRuntime = browserRuntime()): void {
  try {
    runtime.localStorage?.setItem(PWA_ENGAGEMENT_AT_KEY, String(runtime.now ?? Date.now()));
    if (typeof window !== "undefined") {
      window.dispatchEvent(new Event("travel-ai:pwa-engagement"));
    }
  } catch {
    // Engagement tracking is best-effort.
  }
}

export function hasPwaEngagement(runtime: PwaRuntime = browserRuntime()): boolean {
  const storage = runtime.localStorage;
  if (!storage) {
    return false;
  }

  try {
    if (storage.getItem(PWA_ENGAGEMENT_AT_KEY)) {
      return true;
    }
    const visits = Number(storage.getItem(PWA_VISIT_COUNT_KEY) ?? "0");
    return Number.isFinite(visits) && visits >= 2;
  } catch {
    return false;
  }
}

function browserRuntime(): PwaRuntime {
  if (typeof window === "undefined") {
    return {};
  }

  return {
    matchMedia: window.matchMedia.bind(window),
    navigator,
    userAgent: navigator.userAgent,
    platform: navigator.platform,
    localStorage: window.localStorage,
    sessionStorage: window.sessionStorage
  };
}
