import { getExternalIntegrationsServiceUrl } from "@/shared/config";
import type { RouteEstimate, RouteEstimateRequest } from "@/entities/route/model";

type ApiErrorPayload = {
  error?: string;
};

// Route estimates are non-critical UI sugar layered on top of the Haversine
// fallback, so we cap the wait. If the service is slow or down the request is
// aborted and the caller falls back to the straight-line estimate.
const ROUTE_ESTIMATE_TIMEOUT_MS = 8000;

/**
 * Request a route distance/time estimate from External Integrations Service.
 *
 * No auth is required for v1 (the service does not authenticate this endpoint).
 * Throws a readable Error on transport failure, timeout, or a non-2xx response
 * so TanStack Query can surface the per-day error and the UI can fall back to
 * the existing Haversine straight-line estimate.
 */
export async function estimateRoute(request: RouteEstimateRequest): Promise<RouteEstimate> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), ROUTE_ESTIMATE_TIMEOUT_MS);

  let response: Response;
  try {
    response = await fetch(
      new URL("/routes/estimate", getExternalIntegrationsServiceUrl()).toString(),
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json"
        },
        body: JSON.stringify(request),
        signal: controller.signal
      }
    );
  } catch {
    throw new Error(
      "Could not reach the route service. Confirm the local stack is running and CORS allows this origin."
    );
  } finally {
    clearTimeout(timeout);
  }

  if (!response.ok) {
    const payload = await readJson<ApiErrorPayload>(response);
    const message =
      typeof payload?.error === "string" && payload.error.trim().length > 0
        ? payload.error
        : `Route service request failed with status ${response.status}`;
    throw new Error(message);
  }

  return (await response.json()) as RouteEstimate;
}

async function readJson<T>(response: Response): Promise<T | null> {
  try {
    return (await response.json()) as T;
  } catch {
    return null;
  }
}
