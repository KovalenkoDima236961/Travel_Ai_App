import { useTranslations } from "next-intl";
import type { TripRouteStop } from "@/entities/route/model";
import type { StopDayMappingEntry } from "@/lib/route-builder/route-validation";

type RouteStopTimelineNodeProps = {
  stop: TripRouteStop;
  index: number;
  mapping?: StopDayMappingEntry;
  warningCount?: number;
  isOrigin?: boolean;
  canEdit?: boolean;
  onEdit?: () => void;
  onRemove?: () => void;
};

export function RouteStopTimelineNode({
  stop,
  index,
  mapping,
  warningCount = 0,
  isOrigin = false,
  canEdit = false,
  onEdit,
  onRemove
}: RouteStopTimelineNodeProps) {
  const t = useTranslations("route");
  const days = mapping?.days.map((day) => day.day) ?? [];
  const dateLabel = stopDateLabel(stop, days, t);

  return (
    <article
      id={`route-stop-${stop.id}`}
      className="relative scroll-mt-28 rounded-[16px] border border-sand-300 bg-white p-4 shadow-[0_1px_2px_rgba(34,26,20,0.04)]"
    >
      <div className="absolute -left-[35px] top-5 flex h-7 w-7 items-center justify-center rounded-full border-4 border-white bg-cocoa-900 text-[11px] font-bold text-sand-100 shadow-sm sm:-left-[39px]">
        {isOrigin ? "●" : index + 1}
      </div>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <p className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
              {isOrigin ? t("origin") : t("stopNumber", { number: index + 1 })}
            </p>
            {warningCount > 0 ? (
              <span className="rounded-full border border-amber-300 bg-amber-50 px-2 py-0.5 text-[11px] font-semibold text-amber-800">
                {t("warningCount", { count: warningCount })}
              </span>
            ) : null}
          </div>
          <h3 className="mt-1 text-[17px] font-semibold text-cocoa-900">
            {stop.city || stop.destination || t("unnamedStop")}
          </h3>
          {stop.country && stop.country !== stop.destination ? (
            <p className="mt-0.5 text-[13px] text-cocoa-500">{stop.country}</p>
          ) : null}
        </div>
        {canEdit && !isOrigin ? (
          <div className="flex items-center gap-1.5">
            {onEdit ? (
              <button
                aria-label={t("editStopLabel", { name: stop.city || stop.destination })}
                className="rounded-full border border-sand-300 px-3 py-1.5 text-[12px] font-semibold text-cocoa-600 transition hover:border-clay hover:text-clay-deep focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-clay"
                onClick={onEdit}
                type="button"
              >
                {t("edit")}
              </button>
            ) : null}
            {onRemove ? (
              <button
                aria-label={t("removeStopLabel", { name: stop.city || stop.destination })}
                className="rounded-full px-3 py-1.5 text-[12px] font-semibold text-red-700 transition hover:bg-red-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-400"
                onClick={onRemove}
                type="button"
              >
                {t("remove")}
              </button>
            ) : null}
          </div>
        ) : null}
      </div>

      {!isOrigin ? (
        <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-[12.5px] text-cocoa-500">
          <span>{dateLabel}</span>
          {mapping ? <span>{t("itineraryItemCount", { count: mapping.itemCount })}</span> : null}
          {(mapping?.transferDayCount ?? 0) > 0 ? (
            <span>{t("transferDayCount", { count: mapping?.transferDayCount ?? 0 })}</span>
          ) : null}
        </div>
      ) : null}
    </article>
  );
}

function stopDateLabel(
  stop: TripRouteStop,
  days: number[],
  t: ReturnType<typeof useTranslations<"route">>
): string {
  if (days.length > 0) {
    const first = Math.min(...days);
    const last = Math.max(...days);
    return first === last ? t("dayNumber", { number: first }) : t("dayRange", { first, last });
  }
  if (stop.arrivalDate || stop.departureDate) {
    return [stop.arrivalDate, stop.departureDate].filter(Boolean).join(" → ");
  }
  if (stop.nights != null) {
    return t("nightCount", { count: stop.nights });
  }
  return t("datesNotAssigned");
}
