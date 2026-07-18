"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/components/auth/AuthProvider";
import { useDocumentVisibility } from "@/hooks/useDocumentVisibility";
import {
  createAccountExport,
  createTripArchiveExport,
  getAccountExportStatus,
  getTripExportStatus
} from "@/lib/api/data-export";
import { cleanupNotifications, requestAccountCleanup } from "@/lib/api/cleanup";
import { clearOfflineDataScope, getOfflineDataSummary, type OfflineCleanupScope } from "@/lib/offline/data-cleanup";
import type { CreateAccountExportInput, CreateTripArchiveExportInput, NotificationCleanupInput } from "@/types/data-export";

const exportKeys = {
  account: (id: string) => ["data-export", "account", id] as const,
  trip: (tripId: string, id: string) => ["data-export", "trip", tripId, id] as const,
  offline: (userId: string) => ["data-export", "offline", userId] as const
};

export function useCreateAccountExport() {
  return useMutation({ mutationFn: (input: CreateAccountExportInput) => createAccountExport(input) });
}

export function useAccountExportStatus(exportId: string | null) {
  const documentVisible = useDocumentVisibility();
  return useQuery({
    queryKey: exportKeys.account(exportId ?? "none"), queryFn: () => getAccountExportStatus(exportId!),
    enabled: Boolean(exportId),
    refetchInterval: (query) => exportPollInterval(documentVisible, query.state.data?.status, query.state.dataUpdateCount),
    refetchIntervalInBackground: false
  });
}

export function useCreateTripArchiveExport(tripId: string) {
  return useMutation({ mutationFn: (input: CreateTripArchiveExportInput) => createTripArchiveExport(tripId, input) });
}

export function useTripExportStatus(tripId: string, exportId: string | null) {
  const documentVisible = useDocumentVisibility();
  return useQuery({
    queryKey: exportKeys.trip(tripId, exportId ?? "none"), queryFn: () => getTripExportStatus(tripId, exportId!),
    enabled: Boolean(tripId && exportId),
    refetchInterval: (query) => exportPollInterval(documentVisible, query.state.data?.status, query.state.dataUpdateCount),
    refetchIntervalInBackground: false
  });
}

export function useNotificationCleanup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: NotificationCleanupInput) => cleanupNotifications(input),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["notifications"] })
  });
}

export function useOfflineDataSummary() {
  const { user } = useAuth();
  return useQuery({ queryKey: exportKeys.offline(user?.id ?? "none"), queryFn: () => getOfflineDataSummary(user!.id), enabled: Boolean(user?.id) });
}

export function useClearOfflineData() {
  const { user } = useAuth();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (scope: OfflineCleanupScope) => {
      if (!user?.id) throw new Error("Sign in to manage offline data.");
      return clearOfflineDataScope(user.id, scope);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: exportKeys.offline(user?.id ?? "none") })
  });
}

export function useAccountCleanupRequest() {
  return useMutation({ mutationFn: (input: { reason: string; exportRequestedFirst: boolean }) => requestAccountCleanup(input) });
}

// Completed-trip archival remains a normal trip action. This hook is retained
// as the settings extension point without ever deleting trip data automatically.
export function useArchiveCompletedTrips() {
  return useMutation({ mutationFn: async () => ({ archivedCount: 0, message: "Archive individual trips from the travel library." }) });
}

function exportPollInterval(
  documentVisible: boolean,
  status: string | undefined,
  dataUpdateCount: number
) {
  if (!documentVisible || (status !== "queued" && status !== "processing")) {
    return false;
  }
  return dataUpdateCount <= 4 ? 1_500 : 5_000;
}
