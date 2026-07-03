import { describe, expect, it } from "vitest";
import {
  getInstallPlatform,
  hasPwaEngagement,
  isIOS,
  isIOSSafari,
  isInstallPromptDismissedRecently,
  isStandaloneMode,
  recordPwaEngagement,
  recordPwaVisit,
  storeInstallPromptDismissal
} from "@/lib/pwa/pwa-detection";

describe("pwa detection", () => {
  it("detects standalone mode from display media query", () => {
    expect(
      isStandaloneMode({
        matchMedia: (query) => ({ matches: query === "(display-mode: standalone)" })
      })
    ).toBe(true);
  });

  it("detects iOS navigator standalone mode", () => {
    expect(
      isStandaloneMode({
        navigator: { standalone: true }
      })
    ).toBe(true);
  });

  it("detects iOS Safari and excludes iOS Chrome", () => {
    const safari =
      "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Version/17.0 Mobile/15E148 Safari/604.1";
    const chrome =
      "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 CriOS/120.0 Mobile/15E148 Safari/604.1";

    expect(isIOS({ userAgent: safari })).toBe(true);
    expect(isIOSSafari({ userAgent: safari })).toBe(true);
    expect(isIOSSafari({ userAgent: chrome })).toBe(false);
  });

  it("classifies installed, chromium, iOS Safari, and unsupported platforms", () => {
    expect(
      getInstallPlatform({
        matchMedia: (query) => ({ matches: query === "(display-mode: standalone)" })
      })
    ).toBe("installed");
    expect(getInstallPlatform({ beforeInstallPromptSupported: true })).toBe("chromium");
    expect(
      getInstallPlatform({
        userAgent:
          "Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Version/17.0 Mobile/15E148 Safari/604.1"
      })
    ).toBe("ios_safari");
    expect(getInstallPlatform({ userAgent: "Firefox/122.0" })).toBe("unsupported");
  });

  it("tracks install dismissal and engagement state in local storage", () => {
    const storage = memoryStorage();
    const sessionStorage = memoryStorage();

    storeInstallPromptDismissal({ localStorage: storage, now: 1_000 });
    expect(
      isInstallPromptDismissedRecently({
        localStorage: storage,
        now: 1_000 + 60_000
      })
    ).toBe(true);
    expect(
      isInstallPromptDismissedRecently({
        localStorage: storage,
        now: 1_000 + 8 * 24 * 60 * 60 * 1000
      })
    ).toBe(false);

    expect(hasPwaEngagement({ localStorage: storage })).toBe(false);
    recordPwaVisit({ localStorage: storage, sessionStorage });
    recordPwaVisit({ localStorage: storage, sessionStorage });
    expect(hasPwaEngagement({ localStorage: storage })).toBe(false);

    recordPwaEngagement({ localStorage: storage, now: 2_000 });
    expect(hasPwaEngagement({ localStorage: storage })).toBe(true);
  });
});

function memoryStorage() {
  const values = new Map<string, string>();
  return {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => {
      values.set(key, value);
    }
  };
}
