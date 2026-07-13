"use client";

import { Button } from "@/shared/ui/button";
import type { PendingOfflineMutation } from "@/lib/offline/types";

type SyncConflictDialogProps = {
  mutation: PendingOfflineMutation | null;
  onClose: () => void;
};

export function SyncConflictDialog({ mutation, onClose }: SyncConflictDialogProps) {
  if (!mutation) {
    return null;
  }

  return (
    <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="font-semibold">Offline change needs review</h3>
          <p className="mt-1 leading-6">
            {mutation.errorMessage ?? "Refresh from the server, then retry or discard the local change."}
          </p>
        </div>
        <Button onClick={onClose} size="sm" type="button" variant="ghost">
          Close
        </Button>
      </div>
    </div>
  );
}
