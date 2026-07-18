import {
  clearTokens,
  getAccessToken,
  getRefreshToken,
  refresh as refreshAuthTokens,
  saveTokens
} from "@/shared/api/auth";
import { getTripApiBaseUrl } from "@/shared/config";
import {
  AppApiError,
  normalizeApiErrorPayload,
  type ApiErrorPayload,
  type NormalizedApiError
} from "@/lib/api/errors";

export { AppApiError } from "@/lib/api/errors";

type ApiFetchOptions = {
  baseUrl?: string;
  serviceName?: string;
  auth?: boolean;
};

export class ApiError extends AppApiError {
  constructor(normalized: NormalizedApiError, payload?: ApiErrorPayload | null);
  constructor(
    message: string,
    status: number,
    fields?: Record<string, string>,
    code?: string,
    currentItineraryRevision?: number,
    payload?: ApiErrorPayload | null
  );
  constructor(
    input: NormalizedApiError | string,
    statusOrPayload?: number | ApiErrorPayload | null,
    fields?: Record<string, string>,
    code?: string,
    currentItineraryRevision?: number,
    legacyPayload?: ApiErrorPayload | null
  ) {
    if (typeof input === "string") {
      const status = typeof statusOrPayload === "number" ? statusOrPayload : 0;
      const normalized = normalizeApiErrorPayload(null, status, input);
      super(
        {
          ...normalized,
          ...(fields ? { fieldErrors: fields } : {}),
          ...(code ? { code } : {}),
          ...(typeof currentItineraryRevision === "number" ? { currentItineraryRevision } : {})
        },
        legacyPayload
      );
    } else {
      super(input, (statusOrPayload as ApiErrorPayload | null | undefined) ?? undefined);
    }
    this.name = "ApiError";
  }

  get fields() {
    return this.fieldErrors;
  }
}

export type ItineraryConflictError = ApiError & {
  code: "itinerary_conflict";
  currentItineraryRevision: number;
};

export function isItineraryConflictError(error: unknown): error is ItineraryConflictError {
  return (
    error instanceof ApiError &&
    error.status === 409 &&
    error.code === "itinerary_conflict" &&
    typeof error.currentItineraryRevision === "number"
  );
}

// getApiErrorMessage returns a user-facing message for any thrown value, using
// the ApiError message when available and falling back to a generic string.
export function getApiErrorMessage(
  error: unknown,
  fallback = "Something went wrong. Please try again."
): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return fallback;
}

export async function apiFetch<T>(
  path: string,
  init: RequestInit = {},
  options: ApiFetchOptions = {}
): Promise<T> {
  return apiFetchInternal<T>(path, init, true, options);
}

export async function apiFetchPublic<T>(
  path: string,
  init: RequestInit = {},
  options: ApiFetchOptions = {}
): Promise<T> {
  return apiFetchInternal<T>(path, init, false, { ...options, auth: false });
}

async function apiFetchInternal<T>(
  path: string,
  init: RequestInit = {},
  allowRefresh: boolean,
  options: ApiFetchOptions
): Promise<T> {
  const url = buildApiUrl(path, options.baseUrl);
  const headers = new Headers(init.headers);
  const accessToken = getAccessToken();
  const includeAuth = options.auth !== false;

  if (!headers.has("Accept")) {
    headers.set("Accept", "application/json");
  }

  if (includeAuth && accessToken && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${accessToken}`);
  }

  if (init.body && !headers.has("Content-Type") && !(init.body instanceof FormData)) {
    headers.set("Content-Type", "application/json");
  }

  let response: Response;
  try {
    response = await fetch(url, {
      ...init,
      headers
    });
  } catch {
    const serviceName = options.serviceName ?? "Trip Service";
    throw new ApiError(
      {
        code: "unknown_error",
        message: `Could not reach ${serviceName}. Confirm the local stack is running and CORS allows this origin.`,
        status: 0
      }
    );
  }

  if (includeAuth && response.status === 401 && allowRefresh) {
    const refreshToken = getRefreshToken();
    if (refreshToken) {
      try {
        const refreshed = await refreshAuthTokens(refreshToken);
        saveTokens(refreshed.accessToken, refreshed.refreshToken);
        return apiFetchInternal<T>(path, init, false, options);
      } catch {
        clearTokens();
        notifySessionExpired();
        throw new ApiError({ code: "unauthorized", message: "Your session expired. Please log in again.", status: 401 });
      }
    }

    clearTokens();
    notifySessionExpired();
  }

  if (!response.ok) {
    const payload = await readJson<ApiErrorPayload>(response);
    throw new ApiError(normalizeApiErrorPayload(payload, response.status), payload);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const text = await response.text();
  if (!text) {
    return undefined as T;
  }

  return JSON.parse(text) as T;
}

function buildApiUrl(path: string, baseUrl = getTripApiBaseUrl()) {
  const normalizedPath = path.startsWith("/") ? path : `/${path}`;

  if (baseUrl.startsWith("/")) {
    return `${baseUrl}${normalizedPath}`;
  }

  return new URL(normalizedPath, baseUrl).toString();
}

async function readJson<T>(response: Response): Promise<T | null> {
  try {
    return (await response.json()) as T;
  } catch {
    return null;
  }
}

function notifySessionExpired() {
  if (typeof window !== "undefined") {
    window.dispatchEvent(new Event("auth:session-expired"));
  }
}
