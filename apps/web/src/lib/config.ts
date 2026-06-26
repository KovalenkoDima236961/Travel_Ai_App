const DEFAULT_TRIP_SERVICE_URL = "http://localhost:8080";
const DEFAULT_AUTH_SERVICE_URL = "http://localhost:8082";
const DEFAULT_USER_SERVICE_URL = "http://localhost:8083";
const DEFAULT_EXTERNAL_INTEGRATIONS_SERVICE_URL = "http://localhost:8084";
const DEFAULT_NOTIFICATION_SERVICE_URL = "http://localhost:8086";

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

export function getExternalIntegrationsServiceInternalUrl() {
  const value = process.env.EXTERNAL_INTEGRATIONS_SERVICE_INTERNAL_URL?.trim();

  if (value) {
    return value.replace(/\/+$/, "");
  }

  return getExternalIntegrationsServiceUrl();
}

export function getExternalIntegrationsApiBaseUrl() {
  if (typeof window !== "undefined") {
    return "/api/external-integrations-service";
  }

  return getExternalIntegrationsServiceInternalUrl();
}

export function getNotificationServiceUrl() {
  const value = process.env.NEXT_PUBLIC_NOTIFICATION_SERVICE_URL?.trim();

  if (value) {
    return value.replace(/\/+$/, "");
  }

  if (process.env.NODE_ENV !== "production") {
    return DEFAULT_NOTIFICATION_SERVICE_URL;
  }

  throw new Error("NEXT_PUBLIC_NOTIFICATION_SERVICE_URL is not configured.");
}

export function getNotificationServiceInternalUrl() {
  const value = process.env.NOTIFICATION_SERVICE_INTERNAL_URL?.trim();

  if (value) {
    return value.replace(/\/+$/, "");
  }

  return getNotificationServiceUrl();
}

// On the client, notification requests go through the same-origin proxy route
// so the browser never needs CORS access to the Notification Service directly
// (mirrors the Trip Service proxy). On the server, calls go direct.
export function getNotificationApiBaseUrl() {
  if (typeof window !== "undefined") {
    return "/api/notification-service";
  }

  return getNotificationServiceInternalUrl();
}
