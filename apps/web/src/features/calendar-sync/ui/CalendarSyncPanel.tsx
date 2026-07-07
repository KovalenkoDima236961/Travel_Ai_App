"use client";

import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import {
  calendarKeys,
  disconnectGoogleCalendar,
  getGoogleCalendarStatus,
  getTripGoogleCalendarSyncStatus,
  removeTripGoogleCalendarSync,
  startGoogleCalendarConnect,
  syncTripToGoogleCalendar
} from "@/lib/api/calendar";
import { ApiError, isItineraryConflictError } from "@/shared/api/client";
import { formatDate } from "@/lib/utils";
import type { Trip } from "@/entities/trip/model";

type CalendarSyncPanelProps = {
  trip: Trip;
  canSync: boolean;
};

export function CalendarSyncPanel({ trip, canSync }: CalendarSyncPanelProps) {
  const queryClient = useQueryClient();
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const connectionQuery = useQuery({
    queryKey: calendarKeys.googleConnection(),
    queryFn: getGoogleCalendarStatus,
    enabled: canSync
  });
  const syncStatusQuery = useQuery({
    queryKey: calendarKeys.tripGoogleStatus(trip.id),
    queryFn: () => getTripGoogleCalendarSyncStatus(trip.id),
    enabled: canSync
  });

  const connectMutation = useMutation({
    mutationFn: startGoogleCalendarConnect
  });
  const syncMutation = useMutation({
    mutationFn: () => syncTripToGoogleCalendar(trip.id, trip.itineraryRevision),
    onSuccess: async (result) => {
      setError(null);
      setMessage(syncSummary(result.created, result.updated, result.deleted, result.failed));
      await invalidateCalendarQueries(queryClient, trip.id);
    },
    onError: (err) => {
      setMessage(null);
      setError(messageForSyncError(err));
    }
  });
  const removeMutation = useMutation({
    mutationFn: () => removeTripGoogleCalendarSync(trip.id),
    onSuccess: async (result) => {
      setError(null);
      setMessage(`Removed ${result.deleted} Google Calendar event${result.deleted === 1 ? "" : "s"}.`);
      await invalidateCalendarQueries(queryClient, trip.id);
    },
    onError: (err) => {
      setMessage(null);
      setError(err instanceof Error ? err.message : "Could not remove synced events.");
    }
  });
  const disconnectMutation = useMutation({
    mutationFn: disconnectGoogleCalendar,
    onSuccess: async () => {
      setError(null);
      setMessage("Google Calendar disconnected.");
      await invalidateCalendarQueries(queryClient, trip.id);
    },
    onError: (err) => {
      setMessage(null);
      setError(err instanceof Error ? err.message : "Could not disconnect Google Calendar.");
    }
  });

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (params.get("calendar_connected") === "1") {
      setMessage("Google Calendar connected.");
      setError(null);
      params.delete("calendar_connected");
      replaceSearch(params);
      void invalidateCalendarQueries(queryClient, trip.id);
      return;
    }
    const calendarError = params.get("calendar_error");
    if (calendarError) {
      setError("Google Calendar connection was not completed.");
      setMessage(null);
      params.delete("calendar_error");
      replaceSearch(params);
    }
  }, [queryClient, trip.id]);

  const connected = connectionQuery.data?.connected || syncStatusQuery.data?.connected || false;
  const accountEmail =
    connectionQuery.data?.providerAccountEmail ?? syncStatusQuery.data?.providerAccountEmail;
  const syncStatus = syncStatusQuery.data;
  const isBusy =
    connectMutation.isPending ||
    syncMutation.isPending ||
    removeMutation.isPending ||
    disconnectMutation.isPending;
  const statusLabel = useMemo(() => {
    if (!syncStatus?.synced) {
      return "Not synced";
    }
    if (syncStatus.outOfDate) {
      return `Out of date at revision ${syncStatus.syncedItineraryRevision ?? 0}`;
    }
    return `Synced at revision ${syncStatus.syncedItineraryRevision ?? trip.itineraryRevision}`;
  }, [syncStatus, trip.itineraryRevision]);

  async function connect() {
    try {
      setError(null);
      setMessage(null);
      const returnUrl = `${window.location.origin}/trips/${trip.id}`;
      const { authUrl } = await connectMutation.mutateAsync(returnUrl);
      window.location.assign(authUrl);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not start Google Calendar connection.");
    }
  }

  function removeSync() {
    if (!window.confirm("Remove Google Calendar events created for this trip?")) {
      return;
    }
    removeMutation.mutate();
  }

  if (!canSync) {
    return (
      <Card>
        <h2 className="text-lg font-semibold text-slate-950">Calendar sync</h2>
        <p className="mt-2 text-sm leading-6 text-slate-600">
          Calendar sync is available to owners and editors.
        </p>
      </Card>
    );
  }

  return (
    <Card>
      <div className="flex items-start justify-between gap-3">
        <div>
          <h2 className="text-lg font-semibold text-slate-950">Calendar sync</h2>
          <p className="mt-1 text-sm text-slate-600">Google Calendar</p>
        </div>
        {connected ? (
          <span className="rounded-full border border-emerald-200 bg-emerald-50 px-2.5 py-1 text-xs font-medium text-emerald-800">
            Connected
          </span>
        ) : (
          <span className="rounded-full border border-slate-200 bg-slate-50 px-2.5 py-1 text-xs font-medium text-slate-700">
            Not connected
          </span>
        )}
      </div>

      {message ? (
        <div className="mt-4 rounded-md border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-800">
          {message}
        </div>
      ) : null}
      {error ? (
        <div className="mt-4 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
          {error}
        </div>
      ) : null}

      {connected ? (
        <div className="mt-5 space-y-3 text-sm">
          {accountEmail ? (
            <div>
              <p className="text-slate-500">Account</p>
              <p className="mt-1 break-all font-medium text-slate-800">{accountEmail}</p>
            </div>
          ) : null}
          <div>
            <p className="text-slate-500">Trip events</p>
            <p className="mt-1 font-medium text-slate-800">{statusLabel}</p>
            {syncStatus?.lastSyncedAt ? (
              <p className="mt-1 text-xs text-slate-500">
                Last synced {formatDate(syncStatus.lastSyncedAt, { dateStyle: "medium", timeStyle: "short" })}
              </p>
            ) : null}
          </div>
          <div className="grid gap-2">
            <Button
              disabled={isBusy}
              onClick={() => syncMutation.mutate()}
              type="button"
            >
              {syncMutation.isPending
                ? "Syncing..."
                : syncStatus?.synced
                  ? "Update synced events"
                  : "Sync itinerary"}
            </Button>
            {syncStatus?.synced ? (
              <Button
                disabled={isBusy}
                onClick={removeSync}
                type="button"
                variant="secondary"
              >
                {removeMutation.isPending ? "Removing..." : "Remove synced events"}
              </Button>
            ) : null}
            <Button
              disabled={isBusy}
              onClick={() => disconnectMutation.mutate()}
              type="button"
              variant="secondary"
            >
              {disconnectMutation.isPending ? "Disconnecting..." : "Disconnect Google Calendar"}
            </Button>
          </div>
        </div>
      ) : (
        <div className="mt-5">
          <Button disabled={isBusy} onClick={connect} type="button">
            {connectMutation.isPending ? "Connecting..." : "Connect Google Calendar"}
          </Button>
        </div>
      )}
    </Card>
  );
}

function syncSummary(created: number, updated: number, deleted: number, failed: number) {
  if (failed > 0) {
    return `Calendar sync finished with ${failed} failed item${failed === 1 ? "" : "s"}.`;
  }
  return `Calendar synced. Created ${created}, updated ${updated}, removed ${deleted}.`;
}

function messageForSyncError(error: unknown) {
  if (isItineraryConflictError(error)) {
    return "Trip changed. Reload before syncing.";
  }
  if (error instanceof ApiError && error.code === "calendar_not_connected") {
    return "Connect Google Calendar before syncing.";
  }
  if (error instanceof ApiError && error.code === "calendar_reauth_required") {
    return "Reconnect Google Calendar before syncing.";
  }
  return error instanceof Error ? error.message : "Could not sync itinerary.";
}

async function invalidateCalendarQueries(queryClient: ReturnType<typeof useQueryClient>, tripId: string) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: calendarKeys.googleConnection() }),
    queryClient.invalidateQueries({ queryKey: calendarKeys.tripGoogleStatus(tripId) })
  ]);
}

function replaceSearch(params: URLSearchParams) {
  const query = params.toString();
  const nextUrl = `${window.location.pathname}${query ? `?${query}` : ""}${window.location.hash}`;
  window.history.replaceState(null, "", nextUrl);
}
