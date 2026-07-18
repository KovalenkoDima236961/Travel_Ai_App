"use client";

import { useState } from "react";
import { ExportContentsChecklist } from "@/components/data/ExportContentsChecklist";
import { ExportJobStatus } from "@/components/data/ExportJobStatus";
import { ReceiptExportWarning } from "@/components/data/ReceiptExportWarning";
import { PrimaryButton, SaveNotice, SectionHeading, Switch } from "@/components/settings/controls";
import { downloadAccountExport } from "@/lib/api/data-export";
import { getErrorMessage } from "@/lib/utils";
import { useAccountExportStatus, useCreateAccountExport } from "@/hooks/useDataExport";
import { DEFAULT_ACCOUNT_EXPORT_SECTIONS, type AccountExportSections } from "@/types/data-export";

export function AccountExportPanel() {
  const [sections, setSections] = useState<AccountExportSections>(DEFAULT_ACCOUNT_EXPORT_SECTIONS);
  const [includeReceiptFiles, setIncludeReceiptFiles] = useState(false);
  const [includeWorkspaceData, setIncludeWorkspaceData] = useState(false);
  const [exportId, setExportId] = useState<string | null>(null);
  const create = useCreateAccountExport();
  const status = useAccountExportStatus(exportId);
  const job = status.data ?? create.data ?? null;
  return (
    <div className="border-t border-sand-300 pt-6 first:border-0 first:pt-0">
      <SectionHeading title="Download your account data" subtitle="Create a private, short-lived ZIP package. Nothing is shared publicly." />
      <ExportContentsChecklist onChange={setSections} sections={sections} />
      <div className="mt-4 flex flex-col gap-3 text-sm text-cocoa-700">
        <div className="flex items-center justify-between gap-4"><span>Include receipt files</span><Switch checked={includeReceiptFiles} label="Include receipt files" onChange={setIncludeReceiptFiles} /></div>
        <div className="flex items-center justify-between gap-4"><span>Include workspace data you can access</span><Switch checked={includeWorkspaceData} label="Include workspace data" onChange={setIncludeWorkspaceData} /></div>
      </div>
      {includeReceiptFiles ? <ReceiptExportWarning /> : null}
      <p className="mt-4 text-sm leading-6 text-cocoa-500">The account package nests authorized trip archives alongside your profile and preferences. Workspace trips are included only when you are an owner or editor.</p>
      <PrimaryButton className="mt-5" disabled={create.isPending} onClick={() => create.mutate({ sections, includeReceiptFiles, includeWorkspaceData }, { onSuccess: (job) => setExportId(job.exportId) })} type="button">
        {create.isPending ? "Preparing export…" : "Create account export"}
      </PrimaryButton>
      {create.isError ? <div className="mt-4"><SaveNotice errorMessage={getErrorMessage(create.error, "Could not create the export.")} /></div> : null}
      <ExportJobStatus job={job} onDownload={downloadAccountExport} />
    </div>
  );
}
