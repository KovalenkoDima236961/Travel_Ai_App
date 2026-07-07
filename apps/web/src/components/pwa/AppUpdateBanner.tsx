"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useAuth } from "@/components/auth/AuthProvider";
import { Button, buttonStyles } from "@/shared/ui/button";
import { useServiceWorkerUpdate } from "@/hooks/useServiceWorkerUpdate";
import {
  OFFLINE_QUEUE_CHANGED_EVENT,
  getPendingMutations
} from "@/lib/offline/sync-queue";

type AppUpdateBannerViewProps = {
  updateAvailable: boolean;
  pendingCount: number;
  refreshing: boolean;
  onApplyUpdate: () => void;
};

export function AppUpdateBanner() {
  const { user } = useAuth();
  const serviceWorkerUpdate = useServiceWorkerUpdate();
  const [pendingCount, setPendingCount] = useState(0);

  useEffect(() => {
    const userId = user?.id;
    if (!userId) {
      setPendingCount(0);
      return;
    }
    const activeUserId = userId;

    let cancelled = false;
    async function refreshPendingCount() {
      try {
        const mutations = await getPendingMutations(activeUserId);
        if (!cancelled) {
          setPendingCount(mutations.length);
        }
      } catch {
        if (!cancelled) {
          setPendingCount(0);
        }
      }
    }

    void refreshPendingCount();
    window.addEventListener(OFFLINE_QUEUE_CHANGED_EVENT, refreshPendingCount);
    return () => {
      cancelled = true;
      window.removeEventListener(OFFLINE_QUEUE_CHANGED_EVENT, refreshPendingCount);
    };
  }, [user?.id]);

  return (
    <AppUpdateBannerView
      onApplyUpdate={serviceWorkerUpdate.applyUpdate}
      pendingCount={pendingCount}
      refreshing={serviceWorkerUpdate.refreshing}
      updateAvailable={serviceWorkerUpdate.updateAvailable}
    />
  );
}

export function AppUpdateBannerView({
  updateAvailable,
  pendingCount,
  refreshing,
  onApplyUpdate
}: AppUpdateBannerViewProps) {
  if (!updateAvailable) {
    return null;
  }

  const hasPendingOfflineChanges = pendingCount > 0;

  return (
    <aside className="fixed inset-x-4 top-20 z-50 mx-auto max-w-2xl rounded-lg border border-sky-200 bg-white p-4 shadow-xl">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-base font-semibold text-slate-950">A new version is available.</h2>
          <p className="mt-1 text-sm leading-6 text-slate-600">
            {hasPendingOfflineChanges
              ? "Sync or save your offline changes before refreshing."
              : "Refresh when you are ready to update the app."}
          </p>
        </div>

        <div className="flex shrink-0 gap-2">
          {hasPendingOfflineChanges ? (
            <Link className={buttonStyles({ size: "sm", variant: "secondary" })} href="/offline-trips">
              Review offline changes
            </Link>
          ) : (
            <Button disabled={refreshing} onClick={onApplyUpdate} size="sm">
              {refreshing ? "Refreshing..." : "Refresh to update"}
            </Button>
          )}
        </div>
      </div>
    </aside>
  );
}
