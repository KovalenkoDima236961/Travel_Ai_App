import type { ExportTrip } from "@/lib/export/trip-export-adapter";

const MAX_BASE_FILENAME_LENGTH = 90;

export function slugifyForFilename(value: string): string {
  const slug = value
    .normalize("NFKD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/&/g, " and ")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .replace(/-{2,}/g, "-")
    .slice(0, MAX_BASE_FILENAME_LENGTH)
    .replace(/-+$/g, "");

  return slug || "trip";
}

export function buildPdfFilename(exportTrip: ExportTrip): string {
  return `${buildBaseFilename(exportTrip)}.pdf`;
}

export function buildIcsFilename(exportTrip: ExportTrip): string {
  return `${buildBaseFilename(exportTrip)}.ics`;
}

export function buildTripCostReportFilename(title: string, extension: "csv" | "pdf"): string {
  return `${slugifyForFilename(title || "trip")}-cost-report.${extension}`;
}

export function buildWorkspaceCostReportFilename(title: string, extension: "csv" | "pdf"): string {
  return `${slugifyForFilename(title || "workspace")}-workspace-cost-report.${extension}`;
}

function buildBaseFilename(exportTrip: ExportTrip): string {
  const destination = slugifyForFilename(exportTrip.destination || "trip");
  const date = formatDateForFilename(exportTrip.startDate);
  return date ? `${destination}-itinerary-${date}` : `${destination}-itinerary`;
}

function formatDateForFilename(value: string | null | undefined): string | null {
  if (!value) {
    return null;
  }

  const match = value.match(/^(\d{4}-\d{2}-\d{2})/);
  return match?.[1] ?? null;
}
