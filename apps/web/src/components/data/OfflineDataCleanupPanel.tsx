"use client";

import { useState } from "react";
import { DataCleanupConfirmDialog } from "@/components/data/DataCleanupConfirmDialog";
import { GhostButton, SaveNotice, SectionHeading } from "@/components/settings/controls";
import { getErrorMessage } from "@/lib/utils";
import { useClearOfflineData, useOfflineDataSummary } from "@/hooks/useDataExport";
import type { OfflineCleanupScope } from "@/lib/offline/data-cleanup";

const actions: Array<{ scope: OfflineCleanupScope; label: string; description: string }> = [
  { scope: "cachedTrips", label: "Remove cached trip data", description: "Remove downloaded trips, itinerary snapshots, and cached expense details from this device." },
  { scope: "pendingMutations", label: "Discard pending changes", description: "Discard unsynced offline changes. This cannot be undone and changes will not reach the cloud." },
  { scope: "receiptDrafts", label: "Remove offline receipt drafts", description: "Remove receipt files that have not yet been uploaded." },
  { scope: "all", label: "Clear all local app data", description: "Remove cached content, pending changes, receipt drafts, and this app's offline cache from this device." }
];

export function OfflineDataCleanupPanel() {
  const summary = useOfflineDataSummary();
  const clear = useClearOfflineData();
  const [target, setTarget] = useState<(typeof actions)[number] | null>(null);
  const data = summary.data;
  const totalCached = (data?.cachedTrips ?? 0) + (data?.cachedDetails ?? 0) + (data?.cachedChecklists ?? 0) + (data?.cachedReminders ?? 0) + (data?.cachedExpenses ?? 0) + (data?.cachedTravelDays ?? 0);
  return (
    <div className="border-t border-sand-300 pt-6">
      <SectionHeading title="Offline data on this device" subtitle="These controls affect only this browser and device. Synced cloud data is not deleted." />
      <dl className="mt-4 grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
        <Summary label="Cached records" value={String(totalCached)} />
        <Summary label="Pending changes" value={String(data?.pendingMutations ?? 0)} />
        <Summary label="Receipt drafts" value={String(data?.receiptDrafts ?? 0)} />
        <Summary label="Last cleanup" value={data?.lastCleanupAt ? new Date(data.lastCleanupAt).toLocaleDateString() : "Never"} />
      </dl>
      <div className="mt-5 grid gap-2 sm:grid-cols-2">
        {actions.map((action) => <GhostButton className="justify-start text-left" key={action.scope} onClick={() => setTarget(action)} type="button">{action.label}</GhostButton>)}
      </div>
      {clear.isError ? <div className="mt-4"><SaveNotice errorMessage={getErrorMessage(clear.error, "Could not clear local data.")} /></div> : null}
      <DataCleanupConfirmDialog confirmLabel={target?.label ?? "Clear local data"} description={target?.description ?? ""} onCancel={() => setTarget(null)} onConfirm={() => { if (target) clear.mutate(target.scope, { onSuccess: () => setTarget(null) }); }} open={Boolean(target)} pending={clear.isPending} title="Confirm local data cleanup" />
    </div>
  );
}

function Summary({ label, value }: { label: string; value: string }) {
  return <div className="rounded-xl border border-sand-300 bg-sand-50/60 p-3"><dt className="text-xs text-cocoa-500">{label}</dt><dd className="mt-1 font-semibold text-cocoa-900">{value}</dd></div>;
}
