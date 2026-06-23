import { getExternalIntegrationsServiceUrl } from "@/lib/config";
import type { Place, SearchPlacesResponse } from "@/types/place";

type ApiErrorPayload = {
  error?: string;
};

export async function searchPlaces(params: {
  query: string;
  destination?: string;
}): Promise<SearchPlacesResponse> {
  const query = params.query.trim();
  const search = new URLSearchParams({ query });
  const destination = params.destination?.trim();
  if (destination) {
    search.set("destination", destination);
  }

  return placeFetch<SearchPlacesResponse>(`/places/search?${search.toString()}`);
}

export async function getPlaceDetails(placeId: string): Promise<Place> {
  return placeFetch<Place>(`/places/${encodeURIComponent(placeId)}`);
}

async function placeFetch<T>(path: string): Promise<T> {
  let response: Response;
  try {
    response = await fetch(new URL(path, getExternalIntegrationsServiceUrl()).toString(), {
      headers: {
        Accept: "application/json"
      }
    });
  } catch {
    throw new Error(
      "Could not reach Place Service. Confirm the local stack is running and CORS allows this origin."
    );
  }

  if (!response.ok) {
    const payload = await readJson<ApiErrorPayload>(response);
    const message =
      typeof payload?.error === "string" && payload.error.trim().length > 0
        ? payload.error
        : `Place Service request failed with status ${response.status}`;
    throw new Error(message);
  }

  return (await response.json()) as T;
}

async function readJson<T>(response: Response): Promise<T | null> {
  try {
    return (await response.json()) as T;
  } catch {
    return null;
  }
}
