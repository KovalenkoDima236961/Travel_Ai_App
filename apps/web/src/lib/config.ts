const DEFAULT_TRIP_SERVICE_URL = "http://localhost:8080";
const DEFAULT_AUTH_SERVICE_URL = "http://localhost:8082";
const DEFAULT_USER_SERVICE_URL = "http://localhost:8083";
const DEFAULT_EXTERNAL_INTEGRATIONS_SERVICE_URL = "http://localhost:8084";

export function getTripServiceUrl() {
  const value = process.env.NEXT_PUBLIC_TRIP_SERVICE_URL?.trim();

  if (value) {
    return value.replace(/\/+$/, "");
  }

  if (process.env.NODE_ENV !== "production") {
    return DEFAULT_TRIP_SERVICE_URL;
  }

  throw new Error("NEXT_PUBLIC_TRIP_SERVICE_URL is not configured.");
}

export function getTripServiceInternalUrl() {
  const value = process.env.TRIP_SERVICE_INTERNAL_URL?.trim();

  if (value) {
    return value.replace(/\/+$/, "");
  }

  return getTripServiceUrl();
}

export function getTripApiBaseUrl() {
  if (typeof window !== "undefined") {
    return "/api/trip-service";
  }

  return getTripServiceInternalUrl();
}

export function getAuthServiceUrl() {
  const value = process.env.NEXT_PUBLIC_AUTH_SERVICE_URL?.trim();

  if (value) {
    return value.replace(/\/+$/, "");
  }

  if (process.env.NODE_ENV !== "production") {
    return DEFAULT_AUTH_SERVICE_URL;
  }

  throw new Error("NEXT_PUBLIC_AUTH_SERVICE_URL is not configured.");
}

export function getUserServiceUrl() {
  const value = process.env.NEXT_PUBLIC_USER_SERVICE_URL?.trim();

  if (value) {
    return value.replace(/\/+$/, "");
  }

  if (process.env.NODE_ENV !== "production") {
    return DEFAULT_USER_SERVICE_URL;
  }

  throw new Error("NEXT_PUBLIC_USER_SERVICE_URL is not configured.");
}

export function getExternalIntegrationsServiceUrl() {
  const value = process.env.NEXT_PUBLIC_EXTERNAL_INTEGRATIONS_SERVICE_URL?.trim();

  if (value) {
    return value.replace(/\/+$/, "");
  }

  return DEFAULT_EXTERNAL_INTEGRATIONS_SERVICE_URL;
}
