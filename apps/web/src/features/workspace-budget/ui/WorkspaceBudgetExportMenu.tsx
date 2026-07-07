"use client";

import { useState } from "react";
import { Button } from "@/shared/ui/button";
import { downloadTextFile } from "@/lib/export/download";
import { buildWorkspaceBudgetReportFilename } from "@/lib/export/export-filenames";
import { generateWorkspaceBudgetCsv } from "@/lib/export/workspace-budget-csv";
import { downloadWorkspaceBudgetPdf } from "@/lib/export/workspace-budget-pdf";
import type { WorkspaceBudgetSummary } from "@/entities/workspace-budget/model";

type WorkspaceBudgetExportMenuProps = {
  summary: WorkspaceBudgetSummary;
  title: string;
};

export function WorkspaceBudgetExportMenu({ summary, title }: WorkspaceBudgetExportMenuProps) {
  const [message, setMessage] = useState<string | null>(null);
  const [isPdfLoading, setIsPdfLoading] = useState(false);

  function downloadCsv() {
    setMessage(null);
    downloadTextFile(
      generateWorkspaceBudgetCsv(summary),
      buildWorkspaceBudgetReportFilename(title, "csv"),
      "text/csv;charset=utf-8"
    );
  }

  async function downloadPdf() {
    try {
      setMessage(null);
      setIsPdfLoading(true);
      downloadWorkspaceBudgetPdf(summary, title);
    } catch {
      setMessage("PDF export failed. CSV export is still available.");
    } finally {
      setIsPdfLoading(false);
    }
  }

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap gap-2">
        <Button onClick={downloadCsv} type="button" variant="secondary">
          Download CSV
        </Button>
        <Button disabled={isPdfLoading} onClick={downloadPdf} type="button" variant="secondary">
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
