"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useOfflineSync } from "@/hooks/useOfflineSync";
import { useNetworkStatus } from "@/hooks/useNetworkStatus";
import {
  clearOfflineDataForUser,
  deleteCachedTrip,
  getOfflineStorageEstimate,
  listCachedTrips
} from "@/lib/offline/trip-cache";
import {
  OFFLINE_QUEUE_CHANGED_EVENT,
  discardMutation
} from "@/lib/offline/sync-queue";
import type { CachedTripRecord, PendingItineraryMutation } from "@/lib/offline/types";
import { formatDate } from "@/lib/utils";
import { Button, buttonStyles } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";

type OfflineTripsListProps = {
  userId: string;
};

type StorageEstimate = {
  usage?: number;
  quota?: number;
};

export function OfflineTripsList({ userId }: OfflineTripsListProps) {
  const { online } = useNetworkStatus();
  const offlineSync = useOfflineSync({ userId, enabled: Boolean(userId) });
  const [cachedTrips, setCachedTrips] = useState<CachedTripRecord[]>([]);
  const [storageEstimate, setStorageEstimate] = useState<StorageEstimate>({});
  const [loading, setLoading] = useState(true);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [records, estimate] = await Promise.all([
        listCachedTrips(userId),
        getOfflineStorageEstimate()
      ]);
      setCachedTrips(records);
      setStorageEstimate(estimate);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not load offline trips.");
    } finally {
      setLoading(false);
    }
  }, [userId]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useEffect(() => {
    function handleOfflineQueueChanged() {
      void refresh();
    }

    window.addEventListener(OFFLINE_QUEUE_CHANGED_EVENT, handleOfflineQueueChanged);
    return () => {
      window.removeEventListener(OFFLINE_QUEUE_CHANGED_EVENT, handleOfflineQueueChanged);
    };
  }, [refresh]);

  const pendingByTripId = useMemo(() => {
    const map = new Map<string, PendingItineraryMutation>();
    for (const mutation of offlineSync.mutations) {
      map.set(mutation.tripId, mutation);
    }
    return map;
  }, [offlineSync.mutations]);

  async function handleDeleteCachedTrip(record: CachedTripRecord) {
    const pendingMutation = pendingByTripId.get(record.tripId);
    if (pendingMutation) {
      window.alert("Discard pending changes before removing this offline copy.");
      return;
    }

    if (!window.confirm(`Remove offline copy for ${record.trip.destination}?`)) {
      return;
    }

    await deleteCachedTrip(record.tripId, userId);
    setMessage("Offline copy removed.");
    await refresh();
  }

  async function handleDiscardMutation(mutation: PendingItineraryMutation) {
    if (!window.confirm("Discard offline itinerary changes?")) {
      return;
    }

    await discardMutation(mutation.mutationId);
    setMessage("Offline changes discarded.");
    await Promise.all([offlineSync.refresh(), refresh()]);
  }

  async function handleClearOfflineData() {
    const hasPendingChanges = offlineSync.pendingCount > 0;
    const confirmed = window.confirm(
      hasPendingChanges
        ? "You have unsynced changes. Clearing offline data will delete them."
        : "This removes cached trips and pending offline changes stored on this device."
    );
    if (!confirmed) {
      return;
    }

    await clearOfflineDataForUser(userId);
    setMessage("Offline data cleared.");
    await Promise.all([offlineSync.refresh(), refresh()]);
  }

  return (
    <div className="space-y-6">
      <Card>
        <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
          <div>
            <h2 className="text-lg font-semibold text-slate-950">Offline storage</h2>
            <p className="mt-2 text-sm leading-6 text-slate-600">
              Offline storage used: {formatStorageEstimate(storageEstimate)}.
            </p>
            <p className="mt-1 text-sm text-slate-500">
              Cached trips: {cachedTrips.length}. Pending changes: {offlineSync.pendingCount}.
            </p>
          </div>
          <Button
            disabled={cachedTrips.length === 0 && offlineSync.pendingCount === 0}
            onClick={() => void handleClearOfflineData()}
            variant="danger"
          >
            Clear offline data
          </Button>
        </div>
      </Card>

      {message ? (
        <div className="rounded-md border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-800" role="status">
          {message}
        </div>
      ) : null}

      {error ? (
        <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800" role="alert">
          {error}
        </div>
      ) : null}

      {loading ? (
        <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
          Loading offline trips...
        </div>
      ) : null}

      {!loading && cachedTrips.length === 0 ? (
        <div className="rounded-lg border border-slate-200 bg-white p-8 text-center">
          <h2 className="text-lg font-semibold text-slate-950">No offline trips yet</h2>
          <p className="mt-2 text-sm text-slate-600">
            No offline trips yet. Open a trip while online to save it for offline access.
          </p>
          <Link className={buttonStyles({ className: "mt-5" })} href="/trips">
            Go to trips
          </Link>
        </div>
      ) : null}

      {!loading && cachedTrips.length > 0 ? (
        <div className="grid gap-4 lg:grid-cols-2">
          {cachedTrips.map((record) => (
            <OfflineTripCard
              key={record.tripId}
              online={online}
              pendingMutation={pendingByTripId.get(record.tripId) ?? null}
              record={record}
              syncing={offlineSync.syncing}
              onDeleteCachedTrip={() => void handleDeleteCachedTrip(record)}
              onDiscardMutation={handleDiscardMutation}
              onSyncNow={offlineSync.syncNow}
            />
          ))}
        </div>
      ) : null}
    </div>
  );
}

function OfflineTripCard({
  record,
  pendingMutation,
  online,
  syncing,
  onSyncNow,
  onDiscardMutation,
  onDeleteCachedTrip
}: {
  record: CachedTripRecord;
  pendingMutation: PendingItineraryMutation | null;
  online: boolean;
  syncing: boolean;
  onSyncNow: () => Promise<void> | void;
  onDiscardMutation: (mutation: PendingItineraryMutation) => Promise<void> | void;
  onDeleteCachedTrip: () => void;
}) {
  return (
    <Card className="flex h-full flex-col gap-5">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h3 className="truncate text-lg font-semibold text-slate-950">
            {record.trip.destination}
          </h3>
          <p className="mt-1 text-sm text-slate-500">{tripDateSummary(record)}</p>
        </div>
        {pendingMutation ? (
          <span className="rounded-full bg-amber-100 px-2.5 py-1 text-xs font-semibold text-amber-900">
            Pending changes
          </span>
        ) : null}
      </div>

      <dl className="grid grid-cols-2 gap-3 text-sm">
        <TripFact label="Cached" value={formatDateTime(record.cachedAt)} />
        <TripFact label="Revision" value={String(record.itineraryRevision)} />
        <TripFact label="Days" value={`${record.trip.days}`} />
        <TripFact label="Status" value={record.trip.status.toLowerCase()} />
      </dl>

      {pendingMutation ? (
        <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
          <p className="font-medium">Status: {pendingMutation.status}</p>
          {pendingMutation.errorMessage ? (
            <p className="mt-1 leading-6">{pendingMutation.errorMessage}</p>
          ) : null}
        </div>
      ) : null}

      <div className="mt-auto flex flex-wrap gap-2">
        <Link className={buttonStyles({ size: "sm" })} href={`/trips/${record.tripId}`}>
          Open
        </Link>
        {pendingMutation ? (
          <>
            <Button
              disabled={!online || syncing || pendingMutation.status === "conflict"}
              onClick={() => void onSyncNow()}
              size="sm"
              title={!online ? "This action requires an internet connection." : undefined}
              variant="secondary"
            >
              {syncing ? "Syncing..." : "Sync now"}
            </Button>
            <Button
              onClick={() => void onDiscardMutation(pendingMutation)}
              size="sm"
              variant="ghost"
            >
              Discard pending changes
            </Button>
          </>
        ) : null}
        <Button
          disabled={Boolean(pendingMutation)}
          onClick={onDeleteCachedTrip}
          size="sm"
          title={pendingMutation ? "Discard pending changes before removing this copy." : undefined}
          variant="secondary"
        >
          Remove offline copy
        </Button>
      </div>
    </Card>
  );
}

function TripFact({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs font-medium text-slate-500">{label}</dt>
      <dd className="mt-1 truncate font-semibold text-slate-800">{value}</dd>
    </div>
  );
}

function tripDateSummary(record: CachedTripRecord) {
  const startDate = record.trip.startDate;
  if (!startDate) {
    return "Dates not set";
  }
  return `${formatDate(startDate)} · ${record.trip.days} ${record.trip.days === 1 ? "day" : "days"}`;
}

function formatDateTime(value: string) {
  return formatDate(value, { dateStyle: "medium", timeStyle: "short" });
}

function formatStorageEstimate(estimate: StorageEstimate) {
  if (typeof estimate.usage !== "number") {
    return "not available";
  }

  const usage = formatBytes(estimate.usage);
  if (typeof estimate.quota !== "number") {
    return usage;
  }
  return `${usage} of ${formatBytes(estimate.quota)}`;
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
