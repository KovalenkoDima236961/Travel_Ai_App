import { getTripApiBaseUrl } from "@/lib/config";
import { refresh as refreshAuthTokens } from "@/lib/api/auth";
import {
  clearTokens,
  getAccessToken,
  getRefreshToken,
  saveTokens
} from "@/lib/auth/token-storage";

type ApiErrorPayload = {
  error?: string;
  message?: string;
  fields?: Record<string, string>;
  currentItineraryRevision?: number;
  [key: string]: unknown;
};

type ApiFetchOptions = {
  baseUrl?: string;
  serviceName?: string;
  auth?: boolean;
};

export class ApiError extends Error {
  status: number;
  code?: string;
  fields?: Record<string, string>;
  currentItineraryRevision?: number;
  payload?: ApiErrorPayload | null;

  constructor(
    message: string,
    status: number,
    fields?: Record<string, string>,
    code?: string,
    currentItineraryRevision?: number,
    payload?: ApiErrorPayload | null
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.fields = fields;
    this.code = code;
    this.currentItineraryRevision = currentItineraryRevision;
    this.payload = payload;
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

  if (init.body && !headers.has("Content-Type")) {
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
      `Could not reach ${serviceName}. Confirm the local stack is running and CORS allows this origin.`,
      0
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
        throw new ApiError("Your session expired. Please log in again.", 401);
      }
    }

    clearTokens();
    notifySessionExpired();
  }

  if (!response.ok) {
    const payload = await readJson<ApiErrorPayload>(response);
    const message =
      typeof payload?.message === "string" && payload.message.trim().length > 0
        ? payload.message
        : typeof payload?.error === "string" && payload.error.trim().length > 0
          ? payload.error
          : `Request failed with status ${response.status}`;

    throw new ApiError(
      message,
      response.status,
      payload?.fields,
      payload?.error,
      payload?.currentItineraryRevision,
      payload
    );
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
