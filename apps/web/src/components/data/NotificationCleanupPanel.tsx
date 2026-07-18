"use client";

import { useState } from "react";
import { DataCleanupConfirmDialog } from "@/components/data/DataCleanupConfirmDialog";
import { PrimaryButton, SaveNotice, SectionHeading, Switch } from "@/components/settings/controls";
import { getErrorMessage } from "@/lib/utils";
import { useNotificationCleanup } from "@/hooks/useDataExport";

export function NotificationCleanupPanel() {
  const [olderThanDays, setOlderThanDays] = useState(90);
  const [onlyRead, setOnlyRead] = useState(true);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const cleanup = useNotificationCleanup();
  return (
    <div className="border-t border-sand-300 pt-6">
      <SectionHeading title="Notification cleanup" subtitle="Remove old notifications. Read notifications are protected by default." />
      <div className="mt-4 flex flex-wrap items-center gap-4 text-sm text-cocoa-700">
        <label className="flex items-center gap-2">Older than <input className="h-10 w-20 rounded-lg border border-sand-400 px-2" max={3650} min={0} onChange={(event) => setOlderThanDays(Math.max(0, Number(event.target.value)))} type="number" value={olderThanDays} /> days</label>
        <div className="flex items-center gap-2"><span>Keep unread notifications</span><Switch checked={onlyRead} label="Keep unread notifications" onChange={setOnlyRead} /></div>
      </div>
      <PrimaryButton className="mt-4" onClick={() => setConfirmOpen(true)} type="button">Review notification cleanup</PrimaryButton>
      {cleanup.isSuccess ? <div className="mt-3"><SaveNotice successMessage={`${cleanup.data.deletedOrArchivedCount} notification${cleanup.data.deletedOrArchivedCount === 1 ? "" : "s"} removed.`} /></div> : null}
      {cleanup.isError ? <div className="mt-3"><SaveNotice errorMessage={getErrorMessage(cleanup.error, "Could not clean up notifications.")} /></div> : null}
      <DataCleanupConfirmDialog confirmLabel="Delete selected notifications" description={onlyRead ? `This permanently removes read notifications older than ${olderThanDays} days. Unread notifications will be kept.` : `This permanently removes all selected notifications older than ${olderThanDays} days, including unread ones.`} onCancel={() => setConfirmOpen(false)} onConfirm={() => cleanup.mutate({ olderThanDays, onlyRead }, { onSuccess: () => setConfirmOpen(false) })} open={confirmOpen} pending={cleanup.isPending} title="Delete notifications?" />
    </div>
  );
}
