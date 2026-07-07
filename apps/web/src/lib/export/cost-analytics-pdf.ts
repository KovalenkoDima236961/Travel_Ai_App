import {
  buildTripCostAnalyticsPdfLines,
  buildWorkspaceCostAnalyticsPdfLines
} from "@/lib/export/cost-analytics-pdf-lines";
import { downloadPdfLines } from "@/lib/export/pdf";
import {
  buildTripCostReportFilename,
  buildWorkspaceCostReportFilename
} from "@/lib/export/export-filenames";
import type {
  TripCostAnalytics,
  WorkspaceCostAnalytics
} from "@/entities/cost-analytics/model";

export function downloadTripCostAnalyticsPdf(
  analytics: TripCostAnalytics,
  title?: string
): void {
  downloadPdfLines(
    buildTripCostAnalyticsPdfLines(analytics, title),
    buildTripCostReportFilename(title ?? analytics.tripId, "pdf")
  );
}

export function downloadWorkspaceCostAnalyticsPdf(
  analytics: WorkspaceCostAnalytics,
  title?: string
): void {
  downloadPdfLines(
    buildWorkspaceCostAnalyticsPdfLines(analytics, title),
    buildWorkspaceCostReportFilename(title ?? analytics.workspaceId, "pdf")
  );
}
