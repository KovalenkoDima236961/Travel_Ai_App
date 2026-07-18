"use client";

import { useState } from "react";
import { DataCleanupConfirmDialog } from "@/components/data/DataCleanupConfirmDialog";
import { PrimaryButton, SaveNotice, SectionHeading } from "@/components/settings/controls";
import { getErrorMessage } from "@/lib/utils";
import { useAccountCleanupRequest } from "@/hooks/useDataExport";

export function AccountCleanupPanel() {
  const [open, setOpen] = useState(false);
  const request = useAccountCleanupRequest();
  return (
    <div className="border-t border-sand-300 pt-6">
      <SectionHeading title="Account cleanup request" subtitle="Requests are reviewed; this version never deletes your account or trips automatically." />
      <p className="mt-3 text-sm leading-6 text-cocoa-500">Download your data first if you may need it. We record the request and show the next steps instead of silently deleting data.</p>
      <PrimaryButton className="mt-4 bg-clay-deep hover:bg-clay-dark" onClick={() => setOpen(true)} type="button">Request account cleanup</PrimaryButton>
      {request.isSuccess ? <div className="mt-3"><SaveNotice successMessage={request.data.message} /></div> : null}
      {request.isError ? <div className="mt-3"><SaveNotice errorMessage={getErrorMessage(request.error, "Could not record your request.")} /></div> : null}
      <DataCleanupConfirmDialog confirmLabel="Send cleanup request" description="This sends a request for review. It does not automatically delete your account, trips, receipts, or other cloud data." onCancel={() => setOpen(false)} onConfirm={() => request.mutate({ reason: "Requested from data privacy settings", exportRequestedFirst: true }, { onSuccess: () => setOpen(false) })} open={open} pending={request.isPending} title="Request account cleanup?" />
    </div>
  );
}
