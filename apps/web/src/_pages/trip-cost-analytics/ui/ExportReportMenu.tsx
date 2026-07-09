"use client";

import { useState } from "react";
import { generateTripCostAnalyticsCsv } from "@/lib/export/cost-analytics-csv";
import { downloadTripCostAnalyticsPdf } from "@/lib/export/cost-analytics-pdf";
import { downloadTextFile } from "@/lib/export/download";
import { buildTripCostReportFilename } from "@/lib/export/export-filenames";
import type { TripCostAnalytics } from "@/entities/cost-analytics/model";
import { ExportIcon } from "./icons";
import { useAppLanguage } from "@/components/i18n/I18nProvider";

type ExportReportMenuProps = {
  analytics: TripCostAnalytics;
  title: string;
};

const PILL =
  "inline-flex h-[42px] items-center gap-2 rounded-full border border-sand-400 bg-white px-[18px] text-sm font-medium text-cocoa-700 transition hover:border-sand-600 hover:text-cocoa-900 disabled:cursor-not-allowed disabled:opacity-60";

/**
 * Slice-local restyle of the shared CostReportExportMenu. Keeps both the CSV and
 * PDF export paths (the mock shows a single "Export report" button) plus the
 * PDF-failure notice, styled as the mock's warm outline pills.
 */
export function ExportReportMenu({ analytics, title }: ExportReportMenuProps) {
  const { language } = useAppLanguage();
  const [message, setMessage] = useState<string | null>(null);
  const [isPdfLoading, setIsPdfLoading] = useState(false);

  function downloadCsv() {
    setMessage(null);
    downloadTextFile(
      generateTripCostAnalyticsCsv(analytics, language),
      buildTripCostReportFilename(title, "csv"),
      "text/csv;charset=utf-8"
    );
  }

  async function downloadPdf() {
    try {
      setMessage(null);
      setIsPdfLoading(true);
      downloadTripCostAnalyticsPdf(analytics, title);
    } catch {
      setMessage("PDF export failed. You can still export CSV.");
    } finally {
      setIsPdfLoading(false);
    }
  }

  return (
    <div className="flex flex-col items-end gap-2">
      <div className="flex items-center gap-2.5">
        <button className={PILL} onClick={downloadCsv} type="button">
          <ExportIcon className="h-4 w-4" />
          Export CSV
        </button>
        <button className={PILL} disabled={isPdfLoading} onClick={downloadPdf} type="button">
          <ExportIcon className="h-4 w-4" />
          {isPdfLoading ? "Preparing PDF…" : "Export PDF"}
        </button>
      </div>
      {message ? (
        <p className="rounded-full border border-[#EFD9B8] bg-[#FFFDF7] px-3.5 py-1.5 text-[13px] text-[#96682A]">
          {message}
        </p>
      ) : null}
    </div>
  );
}
