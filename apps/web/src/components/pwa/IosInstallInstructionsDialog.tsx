"use client";

import { Button } from "@/shared/ui/button";

type IosInstallInstructionsDialogProps = {
  open: boolean;
  onClose: () => void;
};

export function IosInstallInstructionsDialog({
  open,
  onClose
}: IosInstallInstructionsDialogProps) {
  if (!open) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 flex items-end bg-slate-950/30 p-4 sm:items-center sm:justify-center">
      <section
        aria-modal="true"
        className="w-full max-w-md rounded-lg border border-slate-200 bg-white p-5 shadow-xl"
        role="dialog"
      >
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold text-slate-950">Install on iPhone or iPad</h2>
            <p className="mt-2 text-sm leading-6 text-slate-600">
              iOS installs web apps from Safari.
            </p>
          </div>
          <Button aria-label="Close install instructions" onClick={onClose} size="sm" variant="ghost">
            Close
          </Button>
        </div>

        <ol className="mt-5 space-y-3 text-sm leading-6 text-slate-700">
          <li>1. Open this site in Safari.</li>
          <li>2. Tap the Share button.</li>
          <li>3. Tap Add to Home Screen.</li>
          <li>4. Open Travel AI from your Home Screen.</li>
        </ol>
      </section>
    </div>
  );
}
