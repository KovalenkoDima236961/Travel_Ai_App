export function getTripServiceUrl() {
  const value = process.env.NEXT_PUBLIC_TRIP_SERVICE_URL?.trim();

  if (!value) {
    throw new Error("NEXT_PUBLIC_TRIP_SERVICE_URL is not configured.");
  }

  return value.replace(/\/+$/, "");
}

export function getTripApiBaseUrl() {
  if (typeof window !== "undefined") {
    return "/api/trip-service";
  }

  return getTripServiceUrl();
}
