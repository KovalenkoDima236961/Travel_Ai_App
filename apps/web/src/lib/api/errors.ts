export type ValidationFieldError = {
  field: string;
  message: string;
};

export type ApiErrorPayload = {
  error?: string | {
    code?: string;
    message?: string;
    details?: ValidationFieldError[];
    requestId?: string;
  };
  message?: string;
  fields?: Record<string, string>;
  requestId?: string;
  currentItineraryRevision?: number;
  [key: string]: unknown;
};

export type NormalizedApiError = {
  code: string;
  message: string;
  fieldErrors?: Record<string, string>;
  status: number;
  requestId?: string;
  currentItineraryRevision?: number;
};

/** Normalized error used by browser API wrappers and query mutations. */
export class AppApiError extends Error implements NormalizedApiError {
  readonly status: number;
  readonly code: string;
  readonly fieldErrors?: Record<string, string>;
  readonly requestId?: string;
  readonly currentItineraryRevision?: number;
  readonly payload?: ApiErrorPayload | null;

  constructor(normalized: NormalizedApiError, payload?: ApiErrorPayload | null) {
    super(normalized.message);
    this.name = "AppApiError";
    this.status = normalized.status;
    this.code = normalized.code;
    this.fieldErrors = normalized.fieldErrors;
    this.requestId = normalized.requestId;
    this.currentItineraryRevision = normalized.currentItineraryRevision;
    this.payload = payload;
  }
}

export function normalizeApiErrorPayload(
  payload: ApiErrorPayload | null | undefined,
  status: number,
  fallback = `Request failed with status ${status}`
): NormalizedApiError {
  const error = payload?.error;
  const structured = error && typeof error === "object" ? error : undefined;
  const details = structured?.details ?? [];
  const fieldErrors = {
    ...(payload?.fields ?? {}),
    ...Object.fromEntries(
      details
        .filter((detail) => detail.field.trim().length > 0 && detail.message.trim().length > 0)
        .map((detail) => [detail.field, detail.message])
    )
  };
  const message = firstNonBlank(
    payload?.message,
    structured?.message,
    typeof error === "string" ? error : undefined,
    fallback
  );
  const code = firstNonBlank(
    structured?.code,
    typeof error === "string" && isKnownErrorCode(error) ? error : undefined,
    statusCodeFallback(status)
  );

  return {
    code,
    message,
    ...(Object.keys(fieldErrors).length > 0 ? { fieldErrors } : {}),
    status,
    ...(structured?.requestId ?? payload?.requestId
      ? { requestId: structured?.requestId ?? payload?.requestId }
      : {}),
    ...(typeof payload?.currentItineraryRevision === "number"
      ? { currentItineraryRevision: payload.currentItineraryRevision }
      : {})
  };
}

export function normalizeApiError(error: unknown): NormalizedApiError {
  if (error instanceof AppApiError) {
    return error;
  }
  if (error instanceof Error) {
    return { code: "unknown_error", message: error.message, status: 0 };
  }
  return { code: "unknown_error", message: "Something went wrong. Please try again.", status: 0 };
}

function firstNonBlank(...values: Array<string | undefined>): string {
  return values.find((value) => value?.trim())?.trim() ?? "Something went wrong. Please try again.";
}

function statusCodeFallback(status: number): string {
  if (status === 401) return "unauthorized";
  if (status === 403) return "forbidden";
  if (status === 404) return "not_found";
  if (status === 409) return "conflict";
  if (status === 413) return "upload_too_large";
  if (status === 415) return "upload_invalid_type";
  if (status === 422) return "validation_error";
  if (status === 429) return "rate_limited";
  return "unknown_error";
}

function isKnownErrorCode(value: string): boolean {
  return [
    "unauthorized", "forbidden", "validation_error", "not_found", "conflict",
    "itinerary_conflict", "edit_lock_conflict", "rate_limited",
    "provider_rate_limited", "provider_quota_exceeded", "provider_unavailable",
    "generation_failed", "upload_invalid_type", "upload_too_large",
    "public_share_expired", "public_share_password_required", "internal_auth_required",
    "unknown_error"
  ].includes(value);
}
