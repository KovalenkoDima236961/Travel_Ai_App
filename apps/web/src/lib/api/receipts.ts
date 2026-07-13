import { apiFetch } from "@/shared/api/client";
import { getAccessToken } from "@/shared/api/auth";
import { getTripApiBaseUrl } from "@/shared/config";
import type { CreateExpenseInput, TripExpense } from "@/entities/expense/model";
import type {
  ExpenseReceipt,
  ExtractReceiptInput,
  ListReceiptsParams,
  ReceiptUploadInput,
  TripReceiptsResponse
} from "@/entities/receipt/model";

export const receiptKeys = {
  all: (tripId: string) => ["receipts", tripId] as const,
  list: (tripId: string, params?: ListReceiptsParams) =>
    [...receiptKeys.all(tripId), "list", receiptParamsKey(params)] as const,
  detail: (tripId: string, receiptId: string) =>
    [...receiptKeys.all(tripId), "detail", receiptId] as const
};

export function uploadReceipt(tripId: string, input: ReceiptUploadInput) {
  const form = new FormData();
  form.set("file", input.file);
  if (input.expenseId) {
    form.set("expenseId", input.expenseId);
  }
  if (input.runOcr != null) {
    form.set("runOcr", String(input.runOcr));
  }
  return apiFetch<ExpenseReceipt>(`/trips/${tripId}/expenses/receipts/upload`, {
    method: "POST",
    body: form
  });
}

export function getTripReceipts(tripId: string, params?: ListReceiptsParams) {
  const query = receiptSearchParams(params).toString();
  return apiFetch<TripReceiptsResponse>(
    `/trips/${tripId}/expenses/receipts${query ? `?${query}` : ""}`
  );
}

export function getReceipt(tripId: string, receiptId: string) {
  return apiFetch<ExpenseReceipt>(`/trips/${tripId}/expenses/receipts/${receiptId}`);
}

export function getReceiptFileUrl(tripId: string, receiptId: string) {
  return buildTripApiUrl(`/trips/${tripId}/expenses/receipts/${receiptId}/file`);
}

export async function fetchReceiptFile(tripId: string, receiptId: string) {
  const headers = new Headers();
  const token = getAccessToken();
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  const response = await fetch(getReceiptFileUrl(tripId, receiptId), { headers });
  if (!response.ok) {
    throw new Error(`Could not load receipt file (${response.status}).`);
  }
  return response.blob();
}

export function extractReceipt(
  tripId: string,
  receiptId: string,
  input: ExtractReceiptInput = {}
) {
  return apiFetch<ExpenseReceipt>(`/trips/${tripId}/expenses/receipts/${receiptId}/extract`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function createExpenseFromReceipt(
  tripId: string,
  receiptId: string,
  input: CreateExpenseInput
) {
  return apiFetch<TripExpense>(
    `/trips/${tripId}/expenses/receipts/${receiptId}/create-expense`,
    {
      method: "POST",
      body: JSON.stringify(input)
    }
  );
}

export function attachReceiptToExpense(tripId: string, expenseId: string, receiptId: string) {
  return apiFetch<ExpenseReceipt>(`/trips/${tripId}/expenses/${expenseId}/receipts`, {
    method: "POST",
    body: JSON.stringify({ receiptId })
  });
}

export function deleteReceipt(tripId: string, receiptId: string) {
  return apiFetch<{ success: boolean }>(
    `/trips/${tripId}/expenses/receipts/${receiptId}`,
    { method: "DELETE" }
  );
}

function receiptSearchParams(params?: ListReceiptsParams) {
  const search = new URLSearchParams();
  if (!params) {
    return search;
  }
  if (params.expenseId) {
    search.set("expenseId", params.expenseId);
  }
  if (params.status) {
    search.set("status", params.status);
  }
  if (params.unlinkedOnly) {
    search.set("unlinkedOnly", "true");
  }
  return search;
}

function receiptParamsKey(params?: ListReceiptsParams) {
  return {
    expenseId: params?.expenseId ?? null,
    status: params?.status ?? null,
    unlinkedOnly: params?.unlinkedOnly ?? false
  };
}

function buildTripApiUrl(path: string) {
  const baseUrl = getTripApiBaseUrl();
  if (baseUrl.startsWith("/")) {
    return `${baseUrl}${path}`;
  }
  return new URL(path, baseUrl).toString();
}
