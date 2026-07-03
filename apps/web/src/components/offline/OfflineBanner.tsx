"use client";

import { cn, formatDate } from "@/lib/utils";

type OfflineBannerProps = {
  online: boolean;
  offlineCopy?: boolean;
  cachedAt?: string | null;
  pendingCount?: number;
  syncing?: boolean;
  conflictCount?: number;
  failedCount?: number;
  className?: string;
};

export function OfflineBanner({
  online,
  offlineCopy = false,
  cachedAt,
  pendingCount = 0,
  syncing = false,
  conflictCount = 0,
  failedCount = 0,
  className
}: OfflineBannerProps) {
  if (online && !offlineCopy && pendingCount === 0 && !syncing && conflictCount === 0) {
    return null;
  }

  const tone = conflictCount > 0 || failedCount > 0 ? "amber" : online ? "blue" : "amber";

  return (
    <div
      className={cn(
        "rounded-lg border p-4 text-sm",
        tone === "blue"
          ? "border-sky-200 bg-sky-50 text-sky-900"
          : "border-amber-200 bg-amber-50 text-amber-900",
        className
      )}
    >
      <p className="font-semibold">{primaryMessage({ online, syncing, conflictCount, failedCount })}</p>
      <div className="mt-1 space-y-1 leading-6">
        {!online || offlineCopy ? (
          <p>You are viewing a saved trip. Some actions are unavailable.</p>
        ) : null}
        {cachedAt ? <p>{`Saved offline at ${formatSavedAt(cachedAt)}.`}</p> : null}
        {pendingCount > 0 ? (
          <p>
            {pendingCount} pending offline {pendingCount === 1 ? "change" : "changes"}.
          </p>
        ) : null}
      </div>
    </div>
  );
}

function primaryMessage({
  online,
  syncing,
  conflictCount,
  failedCount
}: {
  online: boolean;
  syncing: boolean;
  conflictCount: number;
  failedCount: number;
}) {
  if (syncing) {
    return "Syncing offline changes...";
  }
  if (conflictCount > 0) {
    return "Offline changes need review.";
  }
  if (failedCount > 0) {
    return "Offline changes could not be synced.";
  }
  if (!online) {
    return "You are offline. Viewing saved data.";
  }
  return "You have pending offline changes.";
}

function formatSavedAt(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return formatDate(value, { hour: "2-digit", minute: "2-digit" });
}
