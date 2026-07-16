import { useTranslations } from "next-intl";
import { formatMoney } from "@/entities/budget/model";
import type { TripRoute } from "@/entities/route/model";
import { getRouteMetrics } from "@/lib/route-builder/route-metrics";
import { formatTransportDuration } from "@/components/transport/transport-format";

export function RouteMetricsCard({
  route,
  totalDays,
  currency = "EUR"
}: {
  route: TripRoute;
  totalDays: number;
  currency?: string;
}) {
  const t = useTranslations("route");
  const metrics = getRouteMetrics(route, totalDays, currency);
  const intensityTone = metrics.intensity === "intense"
    ? "border-red-200 bg-red-50 text-red-700"
    : metrics.intensity === "relaxed"
      ? "border-emerald-200 bg-emerald-50 text-emerald-700"
      : "border-sky-200 bg-sky-50 text-sky-700";
  const values = [
    [t("stops"), String(metrics.stopCount)],
    [t("legs"), String(metrics.legCount)],
    [t("transferTime"), formatTransportDuration(metrics.totalTransferMinutes)],
    [t("transportCost"), metrics.estimatedTransportCost > 0 ? formatMoney(metrics.estimatedTransportCost, metrics.currency) : t("notEstimated")],
    [t("transportCoverage"), `${Math.round(metrics.selectedTransportCoverage * 100)}%`],
    [t("longestTransfer"), formatTransportDuration(metrics.longestTransferMinutes)]
  ];
  return (
    <section className="rounded-[18px] border border-sand-300 bg-white p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">{t("metrics")}</p>
          <h3 className="mt-1 font-newsreader text-[21px] font-semibold text-cocoa-900">{t("atAGlance")}</h3>
        </div>
        <span className={`rounded-full border px-3 py-1 text-[12px] font-semibold ${intensityTone}`}>
          {t("intensityLabel", { intensity: t(metrics.intensity) })}
        </span>
      </div>
      <dl className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3">
        {values.map(([label, value]) => (
          <div className="rounded-[12px] bg-sand-50 p-3" key={label}>
            <dt className="text-[11.5px] font-semibold uppercase tracking-[0.06em] text-cocoa-400">{label}</dt>
            <dd className="mt-1 text-[15px] font-semibold text-cocoa-900">{value}</dd>
          </div>
        ))}
      </dl>
      {metrics.lowConfidenceLegCount > 0 ? (
        <p className="mt-3 text-[12.5px] font-medium text-amber-800">
          {t("lowConfidenceLegCount", { count: metrics.lowConfidenceLegCount })}
        </p>
      ) : null}
    </section>
  );
}
