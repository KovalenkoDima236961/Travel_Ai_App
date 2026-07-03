"use client";

import { useEffect, useMemo, useState } from "react";
import { usePathname } from "next/navigation";
import { useAuth } from "@/components/auth/AuthProvider";
import { IosInstallInstructionsDialog } from "@/components/pwa/IosInstallInstructionsDialog";
import { Button } from "@/components/ui/Button";
import { usePwaInstall, type PwaInstallState } from "@/hooks/usePwaInstall";
import { hasPwaEngagement } from "@/lib/pwa/pwa-detection";

type PwaInstallPromptViewProps = {
  installState: Pick<
    PwaInstallState,
    | "canInstall"
    | "dismissInstallPrompt"
    | "dismissedRecently"
    | "isInstalled"
    | "platform"
    | "promptInstall"
  >;
  readyToShow: boolean;
};

export function PwaInstallPrompt() {
  const installState = usePwaInstall();
  const pathname = usePathname();
  const { isAuthenticated, isLoading } = useAuth();
  const [delayElapsed, setDelayElapsed] = useState(false);
  const [engaged, setEngaged] = useState(false);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setDelayElapsed(true);
    }, 10_000);
    return () => window.clearTimeout(timer);
  }, []);

  useEffect(() => {
    function refreshEngagement() {
      setEngaged(hasPwaEngagement());
    }

    refreshEngagement();
    window.addEventListener("storage", refreshEngagement);
    window.addEventListener("travel-ai:pwa-engagement", refreshEngagement);
    return () => {
      window.removeEventListener("storage", refreshEngagement);
      window.removeEventListener("travel-ai:pwa-engagement", refreshEngagement);
    };
  }, [pathname]);

  const allowedRoute = useMemo(() => {
    if (!pathname) {
      return false;
    }
    return (
      !pathname.startsWith("/login") &&
      !pathname.startsWith("/register") &&
      !pathname.startsWith("/share") &&
      !pathname.startsWith("/ops") &&
      pathname !== "/offline"
    );
  }, [pathname]);

  return (
    <PwaInstallPromptView
      installState={installState}
      readyToShow={
        !isLoading &&
        isAuthenticated &&
        allowedRoute &&
        delayElapsed &&
        (engaged || pathname?.startsWith("/trips") === true)
      }
    />
  );
}

export function PwaInstallPromptView({
  installState,
  readyToShow
}: PwaInstallPromptViewProps) {
  const [iosInstructionsOpen, setIosInstructionsOpen] = useState(false);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);

  if (
    !readyToShow ||
    installState.isInstalled ||
    installState.dismissedRecently ||
    !installState.canInstall
  ) {
    return null;
  }

  const isIOS = installState.platform === "ios_safari";

  async function handleInstall() {
    if (isIOS) {
      setIosInstructionsOpen(true);
      return;
    }

    const result = await installState.promptInstall();
    if (result === "accepted") {
      setStatusMessage("App installed successfully.");
    } else if (result === "unavailable") {
      setStatusMessage("Install is not available in this browser right now.");
    }
  }

  return (
    <>
      <aside className="fixed inset-x-4 bottom-4 z-40 mx-auto max-w-xl rounded-lg border border-primary-100 bg-white p-4 shadow-xl">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 className="text-base font-semibold text-slate-950">Install Travel AI</h2>
            <p className="mt-1 text-sm leading-6 text-slate-600">
              Use your trips offline and get a faster app-like experience.
            </p>
            {statusMessage ? (
              <p className="mt-2 text-sm text-emerald-700" role="status">
                {statusMessage}
              </p>
            ) : null}
          </div>
          <div className="flex shrink-0 gap-2">
            <Button onClick={handleInstall} size="sm">
              {isIOS ? "How to install" : "Install"}
            </Button>
            <Button onClick={installState.dismissInstallPrompt} size="sm" variant="ghost">
              Not now
            </Button>
          </div>
        </div>
      </aside>

      <IosInstallInstructionsDialog
        open={iosInstructionsOpen}
        onClose={() => setIosInstructionsOpen(false)}
      />
    </>
  );
}
