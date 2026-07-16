import { useTranslations } from "next-intl";
import { RouteLegTransportOptions } from "@/components/transport";
import { getCostAmount } from "@/entities/budget/model";
import type { TripRouteLeg } from "@/entities/route/model";
import type { RouteBuilderIssue } from "@/lib/route-builder/route-validation";
import { formatTransportDuration, formatTransportPrice } from "@/components/transport/transport-format";
import { transportModeLabel } from "@/components/routes/route-options";

type RouteLegTimelineConnectorProps = {
  leg?: TripRouteLeg;
  fromName: string;
  toName: string;
  issues?: RouteBuilderIssue[];
  tripId?: string;
  currency?: string;
  travelers?: number;
  canEditTransport?: boolean;
  expectedItineraryRevision?: number;
  online?: boolean;
};

export function RouteLegTimelineConnector({
  leg,
  fromName,
  toName,
  issues = [],
  tripId,
  currency = "EUR",
  travelers = 1,
  canEditTransport = false,
  expectedItineraryRevision,
  online = true
}: RouteLegTimelineConnectorProps) {
  const t = useTranslations("route");
  if (!leg) {
    return (
      <div className="relative ml-4 border-l-2 border-dashed border-amber-300 py-4 pl-8 sm:ml-3 sm:pl-10">
        <div className="rounded-[14px] border border-amber-300 bg-amber-50 p-3 text-[13px] text-amber-900">
          <p className="font-semibold">{t("missingLegData")}</p>
          <p className="mt-1">{t("missingLegBetween", { from: fromName, to: toName })}</p>
        </div>
      </div>
    );
  }

  const option = leg.selectedTransportOption;
  const duration = option?.durationMinutes ?? leg.estimatedDurationMinutes;
  const estimatedPrice = option?.estimatedPrice ??
    (getCostAmount(leg.estimatedCost) != null
      ? { amount: getCostAmount(leg.estimatedCost) ?? 0, currency: leg.estimatedCost?.currency || currency }
      : null);
  const operator = option?.operatorName || option?.serviceName;

  return (
    <div
      id={`route-leg-${leg.id}`}
      className="relative ml-4 scroll-mt-28 border-l-2 border-sand-300 py-4 pl-8 sm:ml-3 sm:pl-10"
    >
      <span aria-hidden className="absolute -left-[7px] top-7 text-[18px] text-clay-deep">↓</span>
      <article className="rounded-[16px] border border-sand-300 bg-[#FFFDFA] p-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
              {t("routeLeg")}
            </p>
            <p className="mt-1 text-[14.5px] font-semibold text-cocoa-900">
              {fromName} → {toName}
            </p>
            <p className="mt-1 text-[12.5px] text-cocoa-500">
              {transportModeLabel(leg.mode)}
              {operator ? ` · ${operator}` : ""}
              {duration ? ` · ${formatTransportDuration(duration)}` : ""}
              {estimatedPrice ? ` · ${formatTransportPrice(estimatedPrice)}` : ""}
            </p>
          </div>
          <div className="flex flex-wrap gap-1.5">
            {!option ? (
              <span className="rounded-full border border-amber-300 bg-amber-50 px-2.5 py-1 text-[11.5px] font-semibold text-amber-800">
                {t("missingTransportOption")}
              </span>
            ) : null}
            {issues.length > 0 ? (
              <span className="rounded-full border border-amber-300 bg-amber-50 px-2.5 py-1 text-[11.5px] font-semibold text-amber-800">
                {t("warningCount", { count: issues.length })}
              </span>
            ) : null}
          </div>
        </div>

        <RouteLegTransportOptions
          canEdit={canEditTransport}
          currency={currency}
          expectedItineraryRevision={expectedItineraryRevision}
          leg={leg}
          online={online}
          travelers={travelers}
          tripId={tripId}
        />

        {issues.length > 0 ? (
          <ul className="mt-3 space-y-1.5 border-t border-sand-200 pt-3 text-[12.5px] text-amber-800">
            {issues.slice(0, 3).map((issue) => (
              <li key={issue.id}>⚠ {issue.title}</li>
            ))}
          </ul>
        ) : null}
      </article>
    </div>
  );
}
