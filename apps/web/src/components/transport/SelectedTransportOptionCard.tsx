import type { SelectedTransportOption } from "@/types/transport";
import { TransportConfidenceBadge } from "./TransportConfidenceBadge";
import { TransportModeBadge } from "./TransportModeBadge";
import { TransportWarningsList } from "./TransportWarningsList";
import {
  formatTransportDuration,
  formatTransportPrice,
  formatTransportTime,
  providerLabel
} from "./transport-format";

type Props = {
  option?: SelectedTransportOption | null;
  canEdit?: boolean;
  removing?: boolean;
  onRemove?: () => void;
};

export function SelectedTransportOptionCard({ option, canEdit = false, removing = false, onRemove }: Props) {
  if (!option) {
    return null;
  }
  const operator = option.operatorName || option.serviceName || providerLabel(option.provider);
  return (
    <div className="rounded-lg border border-clay/30 bg-white p-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <TransportModeBadge mode={option.mode} />
            <TransportConfidenceBadge confidence={option.confidence} />
          </div>
          <p className="mt-2 text-[14px] font-semibold text-cocoa-900">{operator}</p>
          <p className="mt-0.5 text-[12.5px] text-cocoa-500">
            {formatTransportTime(option.departureDate, option.departureTime)} to{" "}
            {formatTransportTime(option.arrivalDate, option.arrivalTime)}
          </p>
        </div>
        <div className="shrink-0 text-right text-[12.5px] font-semibold text-cocoa-600">
          <p>{formatTransportDuration(option.durationMinutes)}</p>
          <p>{formatTransportPrice(option.estimatedPrice)}</p>
        </div>
      </div>
      <div className="mt-2 flex flex-wrap items-center justify-between gap-2">
        <p className="text-[12px] text-cocoa-500">
          {providerLabel(option.provider)}
          {option.status ? ` | ${option.status}` : ""}
        </p>
        {canEdit && onRemove ? (
          <button
            className="rounded-md px-2 py-1 text-[12px] font-semibold text-red-700 transition hover:bg-red-50 disabled:opacity-60"
            disabled={removing}
            onClick={onRemove}
            type="button"
          >
            {removing ? "Removing" : "Remove"}
          </button>
        ) : null}
      </div>
      <div className="mt-2">
        <TransportWarningsList warnings={option.warnings} />
      </div>
      <p className="mt-2 text-[12px] font-medium text-cocoa-500">
        Not booked. Verify schedule, price, and ticket details before travel.
      </p>
    </div>
  );
}
