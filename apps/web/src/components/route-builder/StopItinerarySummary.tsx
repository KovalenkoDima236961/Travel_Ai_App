import { useTranslations } from "next-intl";
import type { StopDayMappingEntry } from "@/lib/route-builder/route-validation";

export function StopItinerarySummary({ entry }: { entry: StopDayMappingEntry }) {
  const t = useTranslations("route");
  return (
    <article className="rounded-[14px] border border-sand-300 bg-[#FFFDFA] p-3">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div>
          <h4 className="text-[14px] font-semibold text-cocoa-900">{entry.stop.city || entry.stop.destination}</h4>
          <p className="mt-1 text-[12.5px] text-cocoa-500">
            {entry.days.length > 0
              ? entry.days.map((day) => t("dayNumber", { number: day.day })).join(", ")
              : t("noAssignedDays")}
          </p>
        </div>
        <span className="rounded-full bg-sand-200 px-2.5 py-1 text-[11.5px] font-semibold text-cocoa-600">
          {t("itineraryItemCount", { count: entry.itemCount })}
        </span>
      </div>
      {entry.transferDayCount > 0 ? (
        <p className="mt-2 text-[12px] text-cocoa-500">{t("transferDayCount", { count: entry.transferDayCount })}</p>
      ) : null}
      {entry.warnings.map((warning) => (
        <p className="mt-2 text-[12px] font-medium text-amber-800" key={warning}>⚠ {warning}</p>
      ))}
    </article>
  );
}
