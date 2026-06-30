import { useMemo } from "react";
import { useQueries } from "@tanstack/react-query";
import { estimateRoute } from "@/lib/api/routes";
import { getRouteStopsByDay, routeStopsCacheKey } from "@/lib/itinerary/route-estimate-utils";
import type { TripAccommodation } from "@/types/accommodation";
import type { RouteEstimate } from "@/types/route";
import type { Itinerary } from "@/types/trip";

export type DayRouteEstimateState = {
  dayNumber: number;
  estimate: RouteEstimate | null;
  isLoading: boolean;
  isError: boolean;
  usesAccommodationAnchor: boolean;
};

export type UseRouteEstimatesResult = {
  byDay: Map<number, DayRouteEstimateState>;
  isAnyLoading: boolean;
};

// Estimates only change when stop coordinates change, so they can be cached for
// a while without going stale in any meaningful way.
const ROUTE_ESTIMATE_STALE_TIME_MS = 5 * 60 * 1000;

const EMPTY_RESULT: UseRouteEstimatesResult = {
  byDay: new Map(),
  isAnyLoading: false
};

/**
 * Fetch route estimates for each day that has at least two mapped stops.
 *
 * One query per qualifying day, keyed by the day's ordered stop coordinates and
 * names so estimates are reused across renders and only refetch when the mapped
 * stops actually change. Retries are disabled so the "route service unavailable"
 * fallback appears quickly. Failures are isolated per day: one failing day never
 * blocks the others, and the caller always retains its Haversine fallback.
 */
export function useRouteEstimates(
  itinerary: Itinerary | null | undefined,
  enabled: boolean,
  accommodation?: TripAccommodation | null
): UseRouteEstimatesResult {
  const daysWithStops = useMemo(
    () => (itinerary ? getRouteStopsByDay(itinerary, accommodation) : []),
    [itinerary, accommodation]
  );

  const results = useQueries({
    queries: daysWithStops.map(({ stops }) => ({
      queryKey: ["route-estimate", "walking", routeStopsCacheKey(stops)],
      queryFn: () => estimateRoute({ mode: "walking" as const, stops }),
      enabled,
      staleTime: ROUTE_ESTIMATE_STALE_TIME_MS,
      gcTime: ROUTE_ESTIMATE_STALE_TIME_MS,
      retry: 0,
      refetchOnWindowFocus: false
    }))
  });

  return useMemo(() => {
    if (daysWithStops.length === 0) {
      return EMPTY_RESULT;
    }

    const byDay = new Map<number, DayRouteEstimateState>();
    let isAnyLoading = false;

    daysWithStops.forEach(({ dayNumber, usesAccommodationAnchor }, index) => {
      const result = results[index];
      const isLoading = Boolean(result?.isLoading);
      if (isLoading) {
        isAnyLoading = true;
      }
      byDay.set(dayNumber, {
        dayNumber,
        estimate: result?.data ?? null,
        isLoading,
        isError: Boolean(result?.isError),
        usesAccommodationAnchor
      });
    });

    return { byDay, isAnyLoading };
  }, [daysWithStops, results]);
}
