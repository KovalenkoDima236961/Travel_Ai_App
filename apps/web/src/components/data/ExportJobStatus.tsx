"use client";

import { useState } from "react";
import { GhostButton, PrimaryButton, SaveNotice } from "@/components/settings/controls";
import { getErrorMessage } from "@/lib/utils";
import type { DataExportJob } from "@/types/data-export";

export function ExportJobStatus({ job, onDownload }: { job: DataExportJob | null; onDownload: (job: DataExportJob) => Promise<void> }) {
  const [error, setError] = useState<string | null>(null);
  if (!job) return null;
  const status = job.status === "completed" ? "Ready" : job.status === "failed" ? "Failed" : job.status === "expired" ? "Expired" : "Preparing";
  return (
    <div className="mt-4 rounded-xl border border-sand-300 bg-sand-50/70 p-4" aria-live="polite">
      <p className="text-sm font-semibold text-cocoa-900">Export status: {status}</p>
      {job.status === "completed" ? <p className="mt-1 text-sm text-cocoa-500">Available until {job.expiresAt ? new Date(job.expiresAt).toLocaleString() : "it expires"}.</p> : null}
      {job.status === "failed" ? <p className="mt-1 text-sm text-clay-deep">{job.errorMessageSafe ?? "We could not create this export. Please try again."}</p> : null}
      {job.status === "completed" && job.downloadUrl ? (
        <PrimaryButton className="mt-3" onClick={() => void onDownload(job).catch((reason: unknown) => setError(getErrorMessage(reason, "Download failed. Please try again.")))} type="button">
          Download private export
        </PrimaryButton>
      ) : null}
      {job.status === "failed" || job.status === "expired" ? <p className="mt-2 text-xs text-cocoa-500">Create a new export when you are ready.</p> : null}
      {error ? <div className="mt-3"><SaveNotice errorMessage={error} /></div> : null}
    </div>
  );
}
