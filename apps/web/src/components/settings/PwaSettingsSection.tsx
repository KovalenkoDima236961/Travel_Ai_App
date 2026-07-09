"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { useAuth } from "@/components/auth/AuthProvider";
import { IosInstallInstructionsDialog } from "@/components/pwa/IosInstallInstructionsDialog";
import {
  PrimaryButton,
  SaveNotice,
  SectionHeading,
  SettingsCard
} from "@/components/settings/controls";
import { usePwaInstall } from "@/hooks/usePwaInstall";
import { useWebPushNotifications } from "@/hooks/useWebPushNotifications";
import {
  clearOfflineDataForUser,
  getOfflineStorageEstimate,
  listCachedTrips
} from "@/lib/offline/trip-cache";
import {
  OFFLINE_QUEUE_CHANGED_EVENT,
  getPendingMutations
} from "@/lib/offline/sync-queue";

const LINK_BUTTON_CLASS =
  "inline-flex h-11 items-center justify-center rounded-full border border-sand-400 bg-white px-[22px] text-[14.5px] font-medium text-cocoa-700 transition hover:border-sand-600 hover:text-cocoa-900";

type OfflineSummary = {
  cachedTripsCount: number;
  pendingChangesCount: number;
  storageUsage?: number;
  storageQuota?: number;
};

export function PwaSettingsSection() {
  const { user } = useAuth();
  const install = usePwaInstall();
  const push = useWebPushNotifications();
  const [iosInstructionsOpen, setIosInstructionsOpen] = useState(false);
  const [offlineSummary, setOfflineSummary] = useState<OfflineSummary>({
    cachedTripsCount: 0,
    pendingChangesCount: 0
  });
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const refreshOfflineSummary = useCallback(async () => {
    if (!user?.id) {
      setOfflineSummary({
        cachedTripsCount: 0,
        pendingChangesCount: 0
      });
      return;
    }

    try {
      const [cachedTrips, pendingMutations, storageEstimate] = await Promise.all([
        listCachedTrips(user.id),
        getPendingMutations(user.id),
        getOfflineStorageEstimate()
      ]);
      setOfflineSummary({
        cachedTripsCount: cachedTrips.length,
        pendingChangesCount: pendingMutations.length,
        storageUsage: storageEstimate.usage,
        storageQuota: storageEstimate.quota
      });
    } catch {
      setOfflineSummary({
        cachedTripsCount: 0,
        pendingChangesCount: 0
      });
    }
  }, [user?.id]);

  useEffect(() => {
    void refreshOfflineSummary();
  }, [refreshOfflineSummary]);

  useEffect(() => {
    window.addEventListener(OFFLINE_QUEUE_CHANGED_EVENT, refreshOfflineSummary);
    return () => {
      window.removeEventListener(OFFLINE_QUEUE_CHANGED_EVENT, refreshOfflineSummary);
    };
  }, [refreshOfflineSummary]);

  async function handleInstall() {
    setError(null);
    setMessage(null);

    if (install.platform === "ios_safari") {
      setIosInstructionsOpen(true);
      return;
    }

    const result = await install.promptInstall();
    if (result === "accepted") {
      setMessage("App installed successfully.");
    } else if (result === "dismissed") {
      setMessage("Install dismissed. You can install from settings later.");
    } else {
      setError("App install is not available in this browser right now.");
    }
  }

  async function handleClearOfflineData() {
    if (!user?.id) {
      return;
    }

    const confirmed = window.confirm(
      offlineSummary.pendingChangesCount > 0
        ? "You have unsynced changes. Clearing offline data will delete them."
        : "This removes cached trips and pending offline changes stored on this device."
    );
    if (!confirmed) {
      return;
    }

    await clearOfflineDataForUser(user.id);
    setMessage("Offline data cleared.");
    await refreshOfflineSummary();
  }

  return (
    <>
      <SettingsCard>
        <SectionHeading
          title="App and offline access"
          subtitle="Manage install status, offline trip storage, and device capabilities."
        />

        <div className="mt-6 grid gap-4 lg:grid-cols-3">
          <StatusPanel
            label="Install status"
            primary={installStatusLabel(install.platform, install.installPromptAvailable)}
            secondary={installStatusDescription(install.platform)}
          />
          <StatusPanel
            label="Offline storage"
            primary={formatStorage(offlineSummary.storageUsage, offlineSummary.storageQuota)}
            secondary={`${offlineSummary.cachedTripsCount} cached ${
              offlineSummary.cachedTripsCount === 1 ? "trip" : "trips"
            }`}
          />
          <StatusPanel
            label="Pending changes"
            primary={String(offlineSummary.pendingChangesCount)}
            secondary="Unsynced itinerary drafts on this device."
          />
        </div>

        <div className="mt-6 grid gap-4 lg:grid-cols-3">
          <StatusPanel
            label="Push support"
            primary={push.supported ? "Supported" : "Unsupported"}
            secondary={`Permission: ${push.permission}`}
          />
          <StatusPanel
            label="Push on this device"
            primary={push.enabled ? "Enabled" : "Not enabled"}
            secondary={
              typeof push.activeSubscriptions === "number"
                ? `Active devices: ${push.activeSubscriptions}`
                : "Device status will update when available."
            }
          />
          <StatusPanel
            label="Installed mode"
            primary={install.isInstalled ? "Installed" : "Browser"}
            secondary={
              install.isInstalled
                ? "Travel AI is running as an installed app."
                : "Install support depends on browser and platform."
            }
          />
        </div>

        <div className="mt-6 flex flex-wrap gap-2.5">
          {!install.isInstalled && install.platform !== "unsupported" ? (
            <PrimaryButton
              disabled={install.platform === "chromium" && !install.installPromptAvailable}
              onClick={() => void handleInstall()}
            >
              {install.platform === "ios_safari" ? "Show install instructions" : "Install app"}
            </PrimaryButton>
          ) : null}
          {install.platform === "unsupported" ? (
            <span className="inline-flex h-11 items-center rounded-full border border-sand-400 px-4 text-[13.5px] text-cocoa-500">
              App install is not supported in this browser.
            </span>
          ) : null}
          <Link href="/offline-trips" className={LINK_BUTTON_CLASS}>
            Manage offline trips
          </Link>
          <Link href="#push-notifications" className={LINK_BUTTON_CLASS}>
            Push settings
          </Link>
          <button
            type="button"
            disabled={
              offlineSummary.cachedTripsCount === 0 && offlineSummary.pendingChangesCount === 0
            }
            onClick={() => void handleClearOfflineData()}
            className="inline-flex h-11 items-center justify-center rounded-full border border-clay/40 bg-white px-[22px] text-[14.5px] font-semibold text-clay-deep transition hover:bg-clay-tint/50 disabled:cursor-not-allowed disabled:opacity-60"
          >
            Clear offline data
          </button>
        </div>

        {message ? (
          <div className="mt-4">
            <SaveNotice successMessage={message} />
          </div>
        ) : null}

        {error ? (
          <div className="mt-4">
            <SaveNotice errorMessage={error} />
          </div>
        ) : null}
      </SettingsCard>

      <IosInstallInstructionsDialog
        open={iosInstructionsOpen}
        onClose={() => setIosInstructionsOpen(false)}
      />
    </>
  );
}

function StatusPanel({
  label,
  primary,
  secondary
}: {
  label: string;
  primary: string;
  secondary: string;
}) {
  return (
    <div className="rounded-2xl border border-sand-300 bg-sand-50/60 p-4">
      <p className="text-[11.5px] font-semibold uppercase tracking-wide text-cocoa-400">{label}</p>
      <p className="mt-2 text-[15px] font-semibold text-cocoa-900">{primary}</p>
      <p className="mt-1 text-[13px] leading-[1.5] text-cocoa-500">{secondary}</p>
    </div>
  );
}

function installStatusLabel(platform: string, promptAvailable: boolean) {
  if (platform === "installed") {
    return "Installed";
  }
  if (platform === "ios_safari") {
    return "iOS manual install";
  }
  if (platform === "chromium" && promptAvailable) {
    return "Available to install";
  }
  if (platform === "chromium") {
    return "Install prompt pending";
  }
  return "Unsupported";
}

function installStatusDescription(platform: string) {
  if (platform === "installed") {
    return "Install prompts are hidden while running standalone.";
  }
  if (platform === "ios_safari") {
    return "Use Safari Share, then Add to Home Screen.";
  }
  if (platform === "chromium") {
    return "The browser controls when install is available.";
  }
  return "App install is not supported in this browser.";
}

function formatStorage(usage?: number, quota?: number) {
  if (typeof usage !== "number") {
    return "Not available";
  }
  if (typeof quota !== "number") {
    return formatBytes(usage);
  }
  return `${formatBytes(usage)} of ${formatBytes(quota)}`;
}

function formatBytes(value: number) {
  if (value < 1024) {
    return `${value} B`;
  }
  const units = ["KB", "MB", "GB"];
  let size = value / 1024;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex += 1;
  }
  return `${size.toFixed(size >= 10 ? 1 : 2)} ${units[unitIndex]}`;
}
