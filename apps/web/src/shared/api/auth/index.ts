import { getAuthServiceUrl } from "@/shared/config";
import { AppApiError, normalizeApiErrorPayload, type ApiErrorPayload } from "@/lib/api/errors";
import type { AuthResponse, AuthUser, TokenResponse } from "./types";

export type { AuthResponse, AuthUser, TokenResponse } from "./types";
export { clearTokens, getAccessToken, getRefreshToken, saveTokens } from "./token-storage";

export class AuthApiError extends AppApiError {}

export type Credentials = {
  email: string;
  password: string;
};

export function register(credentials: Credentials) {
  return authFetch<AuthResponse>("/auth/register", {
    method: "POST",
    body: JSON.stringify(credentials)
  });
}

export function login(credentials: Credentials) {
  return authFetch<AuthResponse>("/auth/login", {
    method: "POST",
    body: JSON.stringify(credentials)
  });
}

export function refresh(refreshToken: string) {
  return authFetch<TokenResponse>("/auth/refresh", {
    method: "POST",
    body: JSON.stringify({ refreshToken })
  });
}

export function logout(refreshToken: string) {
  return authFetch<{ success: boolean }>("/auth/logout", {
    method: "POST",
    body: JSON.stringify({ refreshToken })
  });
}

export function me(accessToken: string) {
  return authFetch<AuthUser>("/auth/me", {
    headers: {
      Authorization: `Bearer ${accessToken}`
    }
  });
}

async function authFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);

  if (!headers.has("Accept")) {
    headers.set("Accept", "application/json");
  }

  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  let response: Response;
  try {
    response = await fetch(buildAuthUrl(path), {
      ...init,
      headers,
      cache: "no-store"
    });
  } catch {
    throw new AuthApiError({
      code: "unknown_error",
      message: "Could not reach Auth Service. Confirm the local stack is running and CORS allows this origin.",
      status: 0
    });
  }

  if (!response.ok) {
    const payload = await readJson<ApiErrorPayload>(response);
    throw new AuthApiError(normalizeApiErrorPayload(payload, response.status), payload);
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

function buildAuthUrl(path: string) {
  const normalizedPath = path.startsWith("/") ? path : `/${path}`;
  return new URL(normalizedPath, getAuthServiceUrl()).toString();
}

async function readJson<T>(response: Response): Promise<T | null> {
  try {
    return (await response.json()) as T;
  } catch {
    return null;
  }
}
