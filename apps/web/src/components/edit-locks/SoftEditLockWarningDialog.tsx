"use client";

import { Button } from "@/components/ui/Button";
import type { EditLockView } from "@/types/edit-locks";

type SoftEditLockWarningDialogProps = {
  lock: EditLockView;
  onCancel: () => void;
  onContinue: () => void;
};

export function SoftEditLockWarningDialog({
  lock,
  onCancel,
  onContinue
}: SoftEditLockWarningDialogProps) {
  const name = lock.lockedByDisplayName?.trim() || "A collaborator";

  return (
    <div
      aria-modal="true"
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 px-4"
      role="dialog"
    >
      <div className="w-full max-w-md rounded-lg border border-slate-200 bg-white p-6 shadow-xl">
        <h2 className="text-lg font-semibold text-slate-950">
          Someone is already editing
        </h2>
        <p className="mt-3 text-sm leading-6 text-slate-600">
          {name} is currently editing this itinerary. You can continue, but your
          changes may conflict if they save first.
        </p>
        <div className="mt-6 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <Button onClick={onCancel} type="button" variant="secondary">
            Cancel
          </Button>
          <Button onClick={onContinue} type="button">
            Continue anyway
          </Button>
        </div>
      </div>
    </div>
  );
}
