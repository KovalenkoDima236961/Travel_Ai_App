import { getTripApiBaseUrl } from "@/lib/config";

type ApiErrorPayload = {
  error?: string;
  fields?: Record<string, string>;
};

export class ApiError extends Error {
  status: number;
  fields?: Record<string, string>;

  constructor(message: string, status: number, fields?: Record<string, string>) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.fields = fields;
  }
}

export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const url = buildApiUrl(path);
  const headers = new Headers(init.headers);

  if (!headers.has("Accept")) {
    headers.set("Accept", "application/json");
  }

  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(url, {
    ...init,
    headers
  });

  if (!response.ok) {
    const payload = await readJson<ApiErrorPayload>(response);
    const message =
      typeof payload?.error === "string" && payload.error.trim().length > 0
        ? payload.error
        : `Request failed with status ${response.status}`;

    throw new ApiError(message, response.status, payload?.fields);
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

function buildApiUrl(path: string) {
  const baseUrl = getTripApiBaseUrl();
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
