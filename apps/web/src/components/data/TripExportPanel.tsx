"use client";

import { useState } from "react";
import { ExportJobStatus } from "@/components/data/ExportJobStatus";
import { ReceiptExportWarning } from "@/components/data/ReceiptExportWarning";
import { PrimaryButton, SaveNotice, Switch } from "@/components/settings/controls";
import { downloadTripCsv, downloadTripExport, getBudgetCsvUrl, getExpenseCsvUrl, getReceiptMetadataCsvUrl, getSettlementCsvUrl } from "@/lib/api/data-export";
import { getErrorMessage } from "@/lib/utils";
import { useCreateTripArchiveExport, useTripExportStatus } from "@/hooks/useDataExport";
import { useFeatureFlag } from "@/lib/feature-flags/useFeatureFlags";

export function TripExportPanel({ tripId, disabled = false }: { tripId: string; disabled?: boolean }) {
  const exportsEnabled = useFeatureFlag("data_exports_enabled");
  const [includeReceiptFiles, setIncludeReceiptFiles] = useState(false);
  const [includeRecapPdf, setIncludeRecapPdf] = useState(false);
  const [exportId, setExportId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const create = useCreateTripArchiveExport(tripId);
  const status = useTripExportStatus(tripId, exportId);
  const job = status.data ?? create.data ?? null;
  const downloadCsv = async (path: string, filename: string) => {
    setError(null);
    try { await downloadTripCsv(path, filename); } catch (reason) { setError(getErrorMessage(reason, "Could not download the CSV.")); }
  };
  if (!exportsEnabled) return null;
  return (
    <div className="rounded-xl border border-sand-300 bg-sand-50/45 p-4" id="trip-export">
      <h3 className="text-base font-semibold text-cocoa-900">Export trip data</h3>
      <p className="mt-1 text-sm leading-6 text-cocoa-500">Editors can create a private archive or download portable CSV files.</p>
      <div className="mt-4 flex flex-col gap-3 text-sm text-cocoa-700">
        <div className="flex items-center justify-between gap-4"><span>Include receipt files</span><Switch checked={includeReceiptFiles} disabled={disabled} label="Include receipt files" onChange={setIncludeReceiptFiles} /></div>
        <div className="flex items-center justify-between gap-4"><span>Include recap PDF when available</span><Switch checked={includeRecapPdf} disabled={disabled} label="Include recap PDF" onChange={setIncludeRecapPdf} /></div>
      </div>
      {includeReceiptFiles ? <ReceiptExportWarning /> : null}
      <div className="mt-4 flex flex-wrap gap-2">
        <PrimaryButton disabled={disabled || create.isPending} onClick={() => create.mutate({ includeReceiptFiles, includeRecapPdf, includePrivateNotes: false }, { onSuccess: (job) => setExportId(job.exportId) })} type="button">{create.isPending ? "Preparing…" : "Create ZIP archive"}</PrimaryButton>
        <button className="rounded-full border border-sand-400 px-3 py-2 text-sm font-medium text-cocoa-700 disabled:opacity-60" disabled={disabled} onClick={() => void downloadCsv(getExpenseCsvUrl(tripId), "expenses.csv")} type="button">Expenses CSV</button>
        <button className="rounded-full border border-sand-400 px-3 py-2 text-sm font-medium text-cocoa-700 disabled:opacity-60" disabled={disabled} onClick={() => void downloadCsv(getSettlementCsvUrl(tripId), "settlements.csv")} type="button">Settlements CSV</button>
        <button className="rounded-full border border-sand-400 px-3 py-2 text-sm font-medium text-cocoa-700 disabled:opacity-60" disabled={disabled} onClick={() => void downloadCsv(getBudgetCsvUrl(tripId), "budget.csv")} type="button">Budget CSV</button>
        <button className="rounded-full border border-sand-400 px-3 py-2 text-sm font-medium text-cocoa-700 disabled:opacity-60" disabled={disabled} onClick={() => void downloadCsv(getReceiptMetadataCsvUrl(tripId), "receipt-metadata.csv")} type="button">Receipt metadata CSV</button>
      </div>
      {create.isError ? <div className="mt-3"><SaveNotice errorMessage={getErrorMessage(create.error, "Could not create the trip export.")} /></div> : null}
      {error ? <div className="mt-3"><SaveNotice errorMessage={error} /></div> : null}
      <ExportJobStatus job={job} onDownload={(readyJob) => downloadTripExport(tripId, readyJob)} />
    </div>
  );
}
