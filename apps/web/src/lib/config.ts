const DEFAULT_TRIP_SERVICE_URL = "http://localhost:8080";
const DEFAULT_AUTH_SERVICE_URL = "http://localhost:8082";
const DEFAULT_USER_SERVICE_URL = "http://localhost:8083";
const DEFAULT_EXTERNAL_INTEGRATIONS_SERVICE_URL = "http://localhost:8084";
const DEFAULT_NOTIFICATION_SERVICE_URL = "http://localhost:8086";
const DEFAULT_WORKER_SERVICE_URL = "http://localhost:8090";

export function getTripServiceUrl() {
  const value = process.env.NEXT_PUBLIC_TRIP_SERVICE_URL?.trim();

  if (value) {
    return validatePublicServiceUrl("NEXT_PUBLIC_TRIP_SERVICE_URL", value);
  }

  if (!isStrictAppEnv()) {
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
    return validatePublicServiceUrl("NEXT_PUBLIC_AUTH_SERVICE_URL", value);
  }

  if (!isStrictAppEnv()) {
    return DEFAULT_AUTH_SERVICE_URL;
  }

  throw new Error("NEXT_PUBLIC_AUTH_SERVICE_URL is not configured.");
}

export function getUserServiceUrl() {
  const value = process.env.NEXT_PUBLIC_USER_SERVICE_URL?.trim();

  if (value) {
    return validatePublicServiceUrl("NEXT_PUBLIC_USER_SERVICE_URL", value);
  }

  if (!isStrictAppEnv()) {
    return DEFAULT_USER_SERVICE_URL;
  }

  throw new Error("NEXT_PUBLIC_USER_SERVICE_URL is not configured.");
}

export function getUserServiceInternalUrl() {
  const value = process.env.USER_SERVICE_INTERNAL_URL?.trim();

  if (value) {
    return value.replace(/\/+$/, "");
  }

  return getUserServiceUrl();
}

export function getUserApiBaseUrl() {
  if (typeof window !== "undefined") {
    return "/api/user-service";
  }

  return getUserServiceInternalUrl();
}

export function getExternalIntegrationsServiceUrl() {
  const value = process.env.NEXT_PUBLIC_EXTERNAL_INTEGRATIONS_SERVICE_URL?.trim();

  if (value) {
    return validatePublicServiceUrl("NEXT_PUBLIC_EXTERNAL_INTEGRATIONS_SERVICE_URL", value);
  }

  if (isStrictAppEnv()) {
    throw new Error("NEXT_PUBLIC_EXTERNAL_INTEGRATIONS_SERVICE_URL is not configured.");
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
    return validatePublicServiceUrl("NEXT_PUBLIC_NOTIFICATION_SERVICE_URL", value);
  }

  if (!isStrictAppEnv()) {
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

export function getWorkerServiceUrl() {
  const value = process.env.NEXT_PUBLIC_WORKER_SERVICE_URL?.trim();

  if (value) {
    return validatePublicServiceUrl("NEXT_PUBLIC_WORKER_SERVICE_URL", value);
  }

  if (isStrictAppEnv()) {
    throw new Error("NEXT_PUBLIC_WORKER_SERVICE_URL is not configured.");
  }

  return DEFAULT_WORKER_SERVICE_URL;
}

export function getWorkerServiceInternalUrl() {
  const value = process.env.WORKER_SERVICE_INTERNAL_URL?.trim();

  if (value) {
    return value.replace(/\/+$/, "");
  }

  return getWorkerServiceUrl();
}

export function getWorkerApiBaseUrl() {
  if (typeof window !== "undefined") {
    return "/api/worker-service";
  }

  return getWorkerServiceInternalUrl();
}

function isStrictAppEnv() {
  const appEnv = (
    process.env.NEXT_PUBLIC_APP_ENV ??
    process.env.APP_ENV ??
    "local"
  ).trim().toLowerCase();
  return appEnv === "staging" || appEnv === "production";
}

function isProductionAppEnv() {
  const appEnv = (
    process.env.NEXT_PUBLIC_APP_ENV ??
    process.env.APP_ENV ??
    "local"
  ).trim().toLowerCase();
  return appEnv === "production";
}

function validatePublicServiceUrl(name: string, value: string) {
  const normalized = value.replace(/\/+$/, "");
  let parsed: URL;
  try {
    parsed = new URL(normalized);
  } catch {
    throw new Error(`${name} must be a valid http/https URL.`);
  }

  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
    throw new Error(`${name} must use http or https.`);
  }
  if (isProductionAppEnv() && parsed.protocol !== "https:") {
    throw new Error(`${name} must use https in production.`);
  }
  if (isProductionAppEnv() && isLocalhost(parsed.hostname)) {
    throw new Error(`${name} must not use localhost in production.`);
  }

  return normalized;
}

function isLocalhost(hostname: string) {
  const normalized = hostname.toLowerCase();
  return normalized === "localhost" || normalized === "127.0.0.1" || normalized === "::1";
}
