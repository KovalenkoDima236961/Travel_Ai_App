import { buildWorkspaceBudgetPdfLines } from "@/lib/export/workspace-budget-pdf-lines";
import { downloadPdfLines } from "@/lib/export/pdf";
import { buildWorkspaceBudgetReportFilename } from "@/lib/export/export-filenames";
import type { WorkspaceBudgetSummary } from "@/types/workspace-budget";

export function downloadWorkspaceBudgetPdf(
  summary: WorkspaceBudgetSummary,
  title?: string
): void {
  downloadPdfLines(
    buildWorkspaceBudgetPdfLines(summary, title),
    buildWorkspaceBudgetReportFilename(title ?? summary.budget.name, "pdf")
  );
}
