import { useTranslations } from "next-intl";
import type { SelectedTransportOption } from "@/types/transport";
import { cn } from "@/shared/lib/cn";
import { TransportConfidenceBadge } from "./TransportConfidenceBadge";
import { TransportModeBadge } from "./TransportModeBadge";
import { TransportWarningsList } from "./TransportWarningsList";
import {
  formatTransportDuration,
  formatTransportPrice,
  formatTransportTime,
  providerLabel
} from "./transport-format";

type SelectedTransportSummaryProps = {
  option?: SelectedTransportOption | null;
  stale?: boolean;
  compact?: boolean;
  canRemove?: boolean;
  removing?: boolean;
  onRemove?: () => void;
  className?: string;
};

export function SelectedTransportSummary({
  option,
  stale = false,
  compact = false,
  canRemove = false,
  removing = false,
  onRemove,
  className
}: SelectedTransportSummaryProps) {
  const t = useTranslations("transport");
  if (!option) {
    return null;
  }
  const operator = option.operatorName || option.serviceName || providerLabel(option.provider);
  const url = option.bookingUrl || option.providerUrl;

  return (
    <article
      aria-label={`${t("selectedTransport")}: ${operator}`}
      className={cn(
        "rounded-[14px] border bg-white p-3",
        stale ? "border-amber-300 bg-amber-50/50" : "border-clay/30",
        className
      )}
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <TransportModeBadge mode={option.mode} />
            <TransportConfidenceBadge confidence={option.confidence} />
            {option.provider === "mock" ? (
              <span className="rounded-md border border-violet-200 bg-violet-50 px-2 py-0.5 text-[12px] font-semibold text-violet-700">
                {t("estimated")}
              </span>
            ) : null}
            {stale ? (
              <span className="rounded-md border border-amber-300 bg-amber-100 px-2 py-0.5 text-[12px] font-semibold text-amber-800">
                {t("stale")}
              </span>
            ) : null}
          </div>
          <p className="mt-2 truncate text-[14px] font-semibold text-cocoa-900">{operator}</p>
          <p className="mt-0.5 text-[12.5px] text-cocoa-500">
            {formatTransportTime(option.departureDate, option.departureTime)} →{" "}
            {formatTransportTime(option.arrivalDate, option.arrivalTime)}
          </p>
        </div>
        <div className="shrink-0 text-right text-[12.5px] font-semibold text-cocoa-600">
          <p>{formatTransportDuration(option.durationMinutes)}</p>
          <p>{formatTransportPrice(option.estimatedPrice)}</p>
          {!compact && (option.transfers ?? 0) > 0 ? (
            <p>{t("transferCount", { count: option.transfers ?? 0 })}</p>
          ) : null}
        </div>
      </div>

      {stale ? (
        <p role="status" className="mt-2 text-[12.5px] font-medium text-amber-800">
          {t("staleWarning")}
        </p>
      ) : null}
      {!compact ? (
        <div className="mt-2">
          <TransportWarningsList warnings={option.warnings} />
        </div>
      ) : null}
      <div className="mt-2 flex flex-wrap items-center justify-between gap-2">
        <p className="text-[12px] font-medium text-cocoa-500">{t("notBooked")}</p>
        <div className="flex items-center gap-2">
          {url ? (
            <a
              className="text-[12px] font-semibold text-clay-deep underline-offset-2 hover:underline"
              href={url}
              rel="noopener noreferrer"
              target="_blank"
            >
              {t("openProvider")}
            </a>
          ) : null}
          {canRemove && onRemove ? (
            <button
              className="rounded-md px-2 py-1 text-[12px] font-semibold text-red-700 transition hover:bg-red-50 disabled:opacity-60"
              disabled={removing}
              onClick={onRemove}
              type="button"
            >
              {removing ? t("removing") : t("removeOption")}
            </button>
          ) : null}
        </div>
      </div>
      {!compact ? (
        <p className="mt-2 text-[12px] text-cocoa-400">{t("verifyScheduleBeforeBooking")}</p>
      ) : null}
    </article>
  );
}
