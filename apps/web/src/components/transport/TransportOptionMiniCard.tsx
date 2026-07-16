import { useTranslations } from "next-intl";
import type { TransportOption } from "@/types/transport";
import { TransportConfidenceBadge } from "./TransportConfidenceBadge";
import { TransportModeBadge } from "./TransportModeBadge";
import { TransportWarningsList } from "./TransportWarningsList";
import {
  formatTransportDuration,
  formatTransportPrice,
  formatTransportTime,
  providerLabel
} from "./transport-format";

type TransportOptionMiniCardProps = {
  option: TransportOption;
  disabled?: boolean;
  selecting?: boolean;
  onSelect: (option: TransportOption) => void;
};

export function TransportOptionMiniCard({
  option,
  disabled = false,
  selecting = false,
  onSelect
}: TransportOptionMiniCardProps) {
  const t = useTranslations("transport");
  const operator = option.operatorName || option.serviceName || providerLabel(option.provider);
  return (
    <article className="rounded-[14px] border border-sand-300 bg-white p-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <TransportModeBadge mode={option.mode} />
            <TransportConfidenceBadge confidence={option.confidence} />
          </div>
          <h4 className="mt-2 text-[14px] font-semibold text-cocoa-900">{operator}</h4>
          <p className="mt-0.5 text-[12.5px] text-cocoa-500">
            {formatTransportTime(option.departureDate, option.departureTime)} →{" "}
            {formatTransportTime(option.arrivalDate, option.arrivalTime)}
          </p>
        </div>
        <div className="shrink-0 text-right text-[12.5px] font-semibold text-cocoa-600">
          <p>{formatTransportDuration(option.durationMinutes)}</p>
          <p>{formatTransportPrice(option.estimatedPrice)}</p>
          {option.transfers > 0 ? <p>{t("transferCount", { count: option.transfers })}</p> : null}
        </div>
      </div>
      <div className="mt-3 flex flex-wrap items-center justify-between gap-2">
        <p className="text-[12px] text-cocoa-500">
          {providerLabel(option.provider)} · {option.status}
        </p>
        <button
          aria-label={`${t("useThisOption")}: ${operator}`}
          className="rounded-md bg-cocoa-900 px-3 py-1.5 text-[12.5px] font-semibold text-white transition hover:bg-cocoa-700 disabled:opacity-60"
          disabled={disabled || selecting}
          onClick={() => onSelect(option)}
          type="button"
        >
          {selecting ? t("selecting") : t("useThisOption")}
        </button>
      </div>
      <div className="mt-2">
        <TransportWarningsList warnings={option.warnings} />
      </div>
    </article>
  );
}
