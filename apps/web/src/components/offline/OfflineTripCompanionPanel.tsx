"use client";

import { useEffect, useState } from "react";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { listOfflineReceiptDrafts } from "@/lib/offline/trip-cache";
import { OfflineReceiptDraftsList } from "@/components/offline/OfflineReceiptDraftsList";
import { PendingChangesList } from "@/components/offline/PendingChangesList";
import { SyncConflictDialog } from "@/components/offline/SyncConflictDialog";
import { SyncQueueStatus } from "@/components/offline/SyncQueueStatus";
import type { OfflineReceiptDraftRecord, PendingOfflineMutation } from "@/lib/offline/types";

type OfflineTripCompanionPanelProps = {
  tripId: string;
  userId?: string | null;
  cachedAt?: string | null;
  mutations: PendingOfflineMutation[];
  online: boolean;
  syncing: boolean;
  onDiscard: (mutation: PendingOfflineMutation) => Promise<void> | void;
  onRefreshOfflineCopy: () => Promise<void> | void;
  onRemoveOfflineCopy: () => Promise<void> | void;
  onSyncNow: () => Promise<void> | void;
};

export function OfflineTripCompanionPanel({
  tripId,
  userId,
  cachedAt,
  mutations,
  online,
  syncing,
  onDiscard,
  onRefreshOfflineCopy,
  onRemoveOfflineCopy,
  onSyncNow
}: OfflineTripCompanionPanelProps) {
  const [drafts, setDrafts] = useState<OfflineReceiptDraftRecord[]>([]);
  const [dismissedConflictId, setDismissedConflictId] = useState<string | null>(null);
  const reviewMutation =
    mutations.find(
      (mutation) =>
        mutation.mutationId !== dismissedConflictId &&
        (mutation.status === "conflict" || mutation.status === "failed")
    ) ?? null;

  useEffect(() => {
    if (!userId) {
      setDrafts([]);
      return;
    }
    let cancelled = false;
    listOfflineReceiptDrafts(userId, tripId).then((items) => {
      if (!cancelled) {
        setDrafts(items);
      }
    });
    return () => {
      cancelled = true;
    };
  }, [mutations.length, tripId, userId]);

  return (
    <Card className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 className="text-lg font-semibold text-slate-950">Offline Trip Companion</h2>
          <p className="mt-1 text-sm leading-6 text-slate-600">
            Available offline{cachedAt ? ` · last saved ${formatDateTime(cachedAt)}` : ""}.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button disabled={!online} onClick={() => void onRefreshOfflineCopy()} size="sm" title={!online ? "This action requires internet." : undefined} variant="secondary">
            Refresh offline copy
          </Button>
          <Button disabled={mutations.length > 0} onClick={() => void onRemoveOfflineCopy()} size="sm" title={mutations.length > 0 ? "Discard or sync pending changes before removing this copy." : undefined} variant="ghost">
            Remove offline copy
          </Button>
        </div>
      </div>

      <SyncQueueStatus
        mutations={mutations}
        onSyncNow={onSyncNow}
        online={online}
        receiptDraftCount={drafts.length}
        syncing={syncing}
      />
      <SyncConflictDialog
        mutation={reviewMutation}
        onClose={() => setDismissedConflictId(reviewMutation?.mutationId ?? null)}
      />
      <OfflineReceiptDraftsList drafts={drafts} />
      <PendingChangesList mutations={mutations} onDiscard={onDiscard} />
    </Card>
  );
}

function formatDateTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}
