"use client";

import { useState } from "react";
import { Button } from "@/shared/ui/button";
import { generateCostSplittingCsv } from "@/lib/export/cost-splitting-csv";
import { downloadCostSplittingPdf } from "@/lib/export/cost-splitting-pdf";
import { downloadTextFile } from "@/lib/export/download";
import { slugifyForFilename } from "@/lib/export/export-filenames";
import type { CostSplittingSummary } from "@/entities/cost-splitting/model";

type CostSplitExportMenuProps = {
  summary: CostSplittingSummary;
  title: string;
};

export function CostSplitExportMenu({ summary, title }: CostSplitExportMenuProps) {
  const [message, setMessage] = useState<string | null>(null);
  const [isPdfLoading, setIsPdfLoading] = useState(false);

  function downloadCsv() {
    setMessage(null);
    downloadTextFile(
      generateCostSplittingCsv(summary),
      `${slugifyForFilename(title || "trip")}-cost-split-report.csv`,
      "text/csv;charset=utf-8"
    );
  }

  async function downloadPdf() {
    try {
      setMessage(null);
      setIsPdfLoading(true);
      downloadCostSplittingPdf(summary, title);
    } catch {
      setMessage("PDF export failed. You can still export CSV.");
    } finally {
      setIsPdfLoading(false);
    }
  }

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap gap-2">
        <Button onClick={downloadCsv} size="sm" type="button" variant="secondary">
          Download CSV
        </Button>
        <Button disabled={isPdfLoading} onClick={downloadPdf} size="sm" type="button" variant="secondary">
          {isPdfLoading ? "Preparing PDF..." : "Download PDF"}
        </Button>
      </div>
      {message ? (
        <p className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900">
          {message}
        </p>
      ) : null}
    </div>
  );
}
