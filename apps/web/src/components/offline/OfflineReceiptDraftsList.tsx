"use client";

import type { OfflineReceiptDraftRecord } from "@/lib/offline/types";

type OfflineReceiptDraftsListProps = {
  drafts: OfflineReceiptDraftRecord[];
};

export function OfflineReceiptDraftsList({ drafts }: OfflineReceiptDraftsListProps) {
  if (drafts.length === 0) {
    return null;
  }

  return (
    <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
      <h3 className="font-semibold">Receipt drafts waiting to upload</h3>
      <ul className="mt-2 space-y-1">
        {drafts.map((draft) => (
          <li key={draft.id}>
            {draft.filename} · {formatBytes(draft.sizeBytes)} · {draft.status}
          </li>
        ))}
      </ul>
    </div>
  );
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
