import { CalendarBusyDaySummaryList } from "./CalendarBusyDaySummaryList";
import type { CalendarImportPreview as CalendarImportPreviewType } from "@/types/calendar-free-busy";

type CalendarImportPreviewProps = {
  preview: CalendarImportPreviewType;
};

export function CalendarImportPreview({ preview }: CalendarImportPreviewProps) {
  const summary = preview.busyBlocksSummary;
  return (
    <div className="space-y-4">
      <div className="grid gap-2 sm:grid-cols-3">
        <Metric label="Busy blocks" value={summary.busyBlockCount} />
        <Metric label="Fully busy days" value={summary.fullyBusyDays} />
        <Metric label="Partially busy days" value={summary.partiallyBusyDays} />
      </div>

      <div>
        <h3 className="text-sm font-semibold text-slate-950">Suggested unavailable ranges</h3>
        <RangeList ranges={preview.suggestedUnavailableRanges} />
      </div>

      <div>
        <h3 className="text-sm font-semibold text-slate-950">Suggested preferred ranges</h3>
        <RangeList ranges={preview.suggestedPreferredRanges} />
      </div>

      <div>
        <h3 className="mb-2 text-sm font-semibold text-slate-950">Busy day summary</h3>
        <CalendarBusyDaySummaryList days={preview.daySummaries} />
      </div>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border border-slate-200 bg-white px-3 py-2">
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 text-lg font-semibold text-slate-950">{value}</p>
    </div>
  );
}

function RangeList({
  ranges
}: {
  ranges: { startDate: string; endDate: string }[];
}) {
  if (ranges.length === 0) {
    return <p className="mt-1 text-sm text-slate-600">None suggested.</p>;
  }
  return (
    <ul className="mt-2 space-y-1 text-sm text-slate-700">
      {ranges.map((range) => (
        <li key={`${range.startDate}-${range.endDate}`}>
          {range.startDate === range.endDate
            ? range.startDate
            : `${range.startDate} to ${range.endDate}`}
        </li>
      ))}
    </ul>
  );
}
