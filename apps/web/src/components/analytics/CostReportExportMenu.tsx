"use client";

import { useState } from "react";
import { Button } from "@/shared/ui/button";
import {
  generateTripCostAnalyticsCsv,
  generateWorkspaceCostAnalyticsCsv
} from "@/lib/export/cost-analytics-csv";
import {
  downloadTripCostAnalyticsPdf,
  downloadWorkspaceCostAnalyticsPdf
} from "@/lib/export/cost-analytics-pdf";
import { downloadTextFile } from "@/lib/export/download";
import {
  buildTripCostReportFilename,
  buildWorkspaceCostReportFilename
} from "@/lib/export/export-filenames";
import type {
  TripCostAnalytics,
  WorkspaceCostAnalytics
} from "@/entities/cost-analytics/model";
import { useAppLanguage } from "@/components/i18n/I18nProvider";

type CostReportExportMenuProps =
  | {
      scope: "trip";
      analytics: TripCostAnalytics;
      title: string;
    }
  | {
      scope: "workspace";
      analytics: WorkspaceCostAnalytics;
      title: string;
    };

export function CostReportExportMenu(props: CostReportExportMenuProps) {
  const { language } = useAppLanguage();
  const [message, setMessage] = useState<string | null>(null);
  const [isPdfLoading, setIsPdfLoading] = useState(false);

  function downloadCsv() {
    setMessage(null);
    if (props.scope === "trip") {
      downloadTextFile(
        generateTripCostAnalyticsCsv(props.analytics, language),
        buildTripCostReportFilename(props.title, "csv"),
        "text/csv;charset=utf-8"
      );
      return;
    }
    downloadTextFile(
      generateWorkspaceCostAnalyticsCsv(props.analytics, language),
      buildWorkspaceCostReportFilename(props.title, "csv"),
      "text/csv;charset=utf-8"
    );
  }

  async function downloadPdf() {
    try {
      setMessage(null);
      setIsPdfLoading(true);
      if (props.scope === "trip") {
        downloadTripCostAnalyticsPdf(props.analytics, props.title);
      } else {
        downloadWorkspaceCostAnalyticsPdf(props.analytics, props.title);
      }
    } catch {
      setMessage("PDF export failed. You can still export CSV.");
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
