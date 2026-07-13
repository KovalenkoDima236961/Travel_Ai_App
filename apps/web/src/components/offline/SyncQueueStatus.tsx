"use client";

import { Button } from "@/shared/ui/button";
import type { PendingOfflineMutation } from "@/lib/offline/types";

type SyncQueueStatusProps = {
  mutations: PendingOfflineMutation[];
  online: boolean;
  syncing: boolean;
  receiptDraftCount?: number;
  onSyncNow: () => Promise<void> | void;
};

export function SyncQueueStatus({
  mutations,
  online,
  syncing,
  receiptDraftCount = 0,
  onSyncNow
}: SyncQueueStatusProps) {
  const pending = mutations.filter((mutation) => mutation.status === "pending").length;
  const failed = mutations.filter((mutation) => mutation.status === "failed").length;
  const conflicts = mutations.filter((mutation) => mutation.status === "conflict").length;

  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h3 className="text-base font-semibold text-slate-950">Offline sync queue</h3>
          <p className="mt-1 text-sm leading-6 text-slate-600">
            {pending} pending, {failed} failed, {conflicts} conflicts, {receiptDraftCount} receipt drafts.
          </p>
        </div>
        <Button
          disabled={!online || syncing || mutations.length === 0}
          onClick={() => void onSyncNow()}
          size="sm"
          title={!online ? "This action requires internet." : undefined}
          variant="secondary"
        >
          {syncing ? "Syncing..." : "Sync now"}
        </Button>
      </div>
    </div>
  );
}
