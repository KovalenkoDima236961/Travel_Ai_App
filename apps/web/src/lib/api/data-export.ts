import { getAccessToken } from "@/shared/api/auth";
import { apiFetch, ApiError } from "@/shared/api/client";
import { getTripApiBaseUrl, getUserApiBaseUrl } from "@/shared/config";
import type {
  CreateAccountExportInput,
  CreateTripArchiveExportInput,
  DataExportJob
} from "@/types/data-export";

function userOptions() {
  return { baseUrl: getUserApiBaseUrl(), serviceName: "User Service" };
}

function tripOptions() {
  return { baseUrl: getTripApiBaseUrl(), serviceName: "Trip Service" };
}

export function createAccountExport(input: CreateAccountExportInput) {
  return apiFetch<DataExportJob>("/users/me/export", {
    method: "POST",
    body: JSON.stringify(input)
  }, userOptions());
}

export function getAccountExportStatus(exportId: string) {
  return apiFetch<DataExportJob>(`/users/me/export/${encodeURIComponent(exportId)}`, {}, userOptions());
}

export function createTripArchiveExport(tripId: string, input: CreateTripArchiveExportInput) {
  return apiFetch<DataExportJob>(`/trips/${encodeURIComponent(tripId)}/export/archive`, {
    method: "POST",
    body: JSON.stringify(input)
  }, tripOptions());
}

export function getTripExportStatus(tripId: string, exportId: string) {
  return apiFetch<DataExportJob>(
    `/trips/${encodeURIComponent(tripId)}/export/${encodeURIComponent(exportId)}`,
    {},
    tripOptions()
  );
}

export function getExpenseCsvUrl(tripId: string) {
  return `/trips/${encodeURIComponent(tripId)}/expenses/export.csv`;
}

export function getSettlementCsvUrl(tripId: string) {
  return `/trips/${encodeURIComponent(tripId)}/settlements/export.csv`;
}

export function getBudgetCsvUrl(tripId: string) {
  return `/trips/${encodeURIComponent(tripId)}/budget/export.csv`;
}

export function getReceiptMetadataCsvUrl(tripId: string) {
  return `/trips/${encodeURIComponent(tripId)}/expenses/receipts/export-metadata.csv`;
}

export async function downloadAccountExport(job: DataExportJob): Promise<void> {
  if (!job.downloadUrl || !job.fileName) {
    throw new Error("This export is not ready for download.");
  }
  await downloadPrivateFile(job.downloadUrl, getUserApiBaseUrl(), job.fileName);
}

export async function downloadTripExport(tripId: string, job: DataExportJob): Promise<void> {
  if (!job.downloadUrl || !job.fileName) {
    throw new Error("This export is not ready for download.");
  }
  await downloadPrivateFile(job.downloadUrl, getTripApiBaseUrl(), job.fileName);
}

export async function downloadTripCsv(path: string, fallbackFilename: string): Promise<void> {
  await downloadPrivateFile(path, getTripApiBaseUrl(), fallbackFilename);
}

async function downloadPrivateFile(path: string, baseUrl: string, fallbackFilename: string) {
  const headers = new Headers({ Accept: "application/octet-stream" });
  const accessToken = getAccessToken();
  if (accessToken) {
    headers.set("Authorization", `Bearer ${accessToken}`);
  }

  let response: Response;
  try {
    response = await fetch(buildUrl(path, baseUrl), { headers, cache: "no-store" });
  } catch {
    throw new Error("Could not download this export. Please try again.");
  }
  if (!response.ok) {
    throw new ApiError(`Download failed with status ${response.status}`, response.status);
  }
  const blob = await response.blob();
  const filename = filenameFromContentDisposition(response.headers.get("content-disposition")) ?? fallbackFilename;
  const objectUrl = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = objectUrl;
  anchor.download = filename;
  anchor.style.display = "none";
  document.body.append(anchor);
  anchor.click();
  anchor.remove();
  window.setTimeout(() => URL.revokeObjectURL(objectUrl), 0);
}

function buildUrl(path: string, baseUrl: string) {
  const normalizedPath = path.startsWith("/") ? path : `/${path}`;
  return baseUrl.startsWith("/") ? `${baseUrl}${normalizedPath}` : new URL(normalizedPath, baseUrl).toString();
}

function filenameFromContentDisposition(value: string | null) {
  const match = value?.match(/filename\*?=(?:UTF-8''|\")?([^;\"]+)/i);
  return match?.[1] ? decodeURIComponent(match[1].trim()) : null;
}
