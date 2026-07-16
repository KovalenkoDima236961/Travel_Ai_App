import { useTranslations } from "next-intl";
import type { TripRoute } from "@/entities/route/model";
import type { Itinerary } from "@/entities/trip/model";
import type { RouteBuilderIssue } from "@/lib/route-builder/route-validation";
import { mapItineraryToStops } from "@/lib/route-builder/route-validation";
import { RouteLegTimelineConnector } from "./RouteLegTimelineConnector";
import { RouteStopTimelineNode } from "./RouteStopTimelineNode";

type RouteTimelineProps = {
  route?: TripRoute | null;
  itinerary?: Itinerary | null;
  issues?: RouteBuilderIssue[];
  tripId?: string;
  currency?: string;
  travelers?: number;
  canEdit?: boolean;
  canEditTransport?: boolean;
  expectedItineraryRevision?: number;
  online?: boolean;
  onEditStop?: (stopId: string) => void;
  onRemoveStop?: (stopId: string) => void;
};

export function RouteTimeline({
  route,
  itinerary,
  issues = [],
  tripId,
  currency = "EUR",
  travelers = 1,
  canEdit = false,
  canEditTransport = false,
  expectedItineraryRevision,
  online = true,
  onEditStop,
  onRemoveStop
}: RouteTimelineProps) {
  const t = useTranslations("route");
  if (!route || route.stops.length === 0) {
    return (
      <div className="rounded-[16px] border border-dashed border-sand-400 bg-sand-50 p-6 text-center">
        <p className="text-[15px] font-semibold text-cocoa-800">{t("noRouteYet")}</p>
        <p className="mt-1 text-[13px] text-cocoa-500">{t("noRouteDescription")}</p>
      </div>
    );
  }

  const mapping = new Map(mapItineraryToStops(route, itinerary).map((entry) => [entry.stop.id, entry]));
  const originName = route.origin?.name?.trim() || t("origin");
  const originStop = {
    id: "origin",
    destination: originName,
    country: route.origin?.country ?? null
  };

  return (
    <section aria-label={t("timeline")} className="pl-8 sm:pl-9">
      <div className="relative">
        <RouteStopTimelineNode isOrigin index={-1} stop={originStop} />
        {route.stops.map((stop, index) => {
          const leg = route.legs?.[index];
          const legIssues = issues.filter((issue) => issue.legId === leg?.id);
          const stopIssues = issues.filter(
            (issue) => issue.stopId === stop.id || mapping.get(stop.id)?.days.some((day) => issue.dayNumber === day.day)
          );
          return (
            <div key={stop.id}>
              <RouteLegTimelineConnector
                canEditTransport={canEditTransport}
                currency={currency}
                expectedItineraryRevision={expectedItineraryRevision}
                fromName={index === 0 ? originName : route.stops[index - 1]?.city || route.stops[index - 1]?.destination}
                issues={legIssues}
                leg={leg}
                online={online}
                toName={stop.city || stop.destination || t("unnamedStop")}
                travelers={travelers}
                tripId={tripId}
              />
              <RouteStopTimelineNode
                canEdit={canEdit}
                index={index}
                mapping={mapping.get(stop.id)}
                onEdit={onEditStop ? () => onEditStop(stop.id) : undefined}
                onRemove={onRemoveStop ? () => onRemoveStop(stop.id) : undefined}
                stop={stop}
                warningCount={stopIssues.length}
              />
            </div>
          );
        })}
      </div>
    </section>
  );
}
