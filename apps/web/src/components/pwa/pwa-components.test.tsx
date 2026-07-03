import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { AppUpdateBannerView } from "@/components/pwa/AppUpdateBanner";
import { IosInstallInstructionsDialog } from "@/components/pwa/IosInstallInstructionsDialog";
import { PwaInstallPromptView } from "@/components/pwa/PwaInstallPrompt";
import type { PwaInstallState } from "@/hooks/usePwaInstall";

describe("PwaInstallPrompt", () => {
  it("shows install action for chromium", () => {
    const html = renderToStaticMarkup(
      <PwaInstallPromptView installState={installState({ platform: "chromium" })} readyToShow />
    );

    expect(html).toContain("Install Travel AI");
    expect(html).toContain("Use your trips offline");
    expect(html).toContain("Install");
    expect(html).toContain("Not now");
  });

  it("shows iOS instructions action for iOS Safari", () => {
    const html = renderToStaticMarkup(
      <PwaInstallPromptView installState={installState({ platform: "ios_safari" })} readyToShow />
    );

    expect(html).toContain("How to install");
  });

  it("hides when installed or dismissed", () => {
    expect(
      renderToStaticMarkup(
        <PwaInstallPromptView
          installState={installState({ isInstalled: true, platform: "installed" })}
          readyToShow
        />
      )
    ).toBe("");

    expect(
      renderToStaticMarkup(
        <PwaInstallPromptView
          installState={installState({ dismissedRecently: true })}
          readyToShow
        />
      )
    ).toBe("");
  });
});

describe("IosInstallInstructionsDialog", () => {
  it("renders manual Add to Home Screen steps", () => {
    const html = renderToStaticMarkup(
      <IosInstallInstructionsDialog open onClose={() => undefined} />
    );

    expect(html).toContain("Open this site in Safari");
    expect(html).toContain("Tap the Share button");
    expect(html).toContain("Add to Home Screen");
  });
});

describe("AppUpdateBanner", () => {
  it("shows refresh action when there are no pending offline changes", () => {
    const html = renderToStaticMarkup(
      <AppUpdateBannerView
        onApplyUpdate={() => undefined}
        pendingCount={0}
        refreshing={false}
        updateAvailable
      />
    );

    expect(html).toContain("A new version is available");
    expect(html).toContain("Refresh to update");
  });

  it("shows offline-change warning instead of refresh action", () => {
    const html = renderToStaticMarkup(
      <AppUpdateBannerView
        onApplyUpdate={() => undefined}
        pendingCount={2}
        refreshing={false}
        updateAvailable
      />
    );

    expect(html).toContain("Sync or save your offline changes before refreshing");
    expect(html).toContain("Review offline changes");
    expect(html).not.toContain("Refresh to update");
  });
});

function installState(
  overrides: Partial<PwaInstallState> = {}
): Pick<
  PwaInstallState,
  | "canInstall"
  | "dismissInstallPrompt"
  | "dismissedRecently"
  | "isInstalled"
  | "platform"
  | "promptInstall"
> {
  return {
    canInstall: true,
    dismissedRecently: false,
    dismissInstallPrompt: () => undefined,
    isInstalled: false,
    platform: "chromium",
    promptInstall: async () => "accepted",
    ...overrides
  };
}
