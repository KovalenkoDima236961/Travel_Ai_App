import type { CalendarBusyDaySummary } from "@/types/calendar-free-busy";

type CalendarBusyDaySummaryListProps = {
  days: CalendarBusyDaySummary[];
};

export function CalendarBusyDaySummaryList({ days }: CalendarBusyDaySummaryListProps) {
  if (days.length === 0) {
    return <p className="text-sm text-slate-600">No busy days found in this range.</p>;
  }
  return (
    <div className="max-h-48 overflow-auto rounded-md border border-slate-200">
      {days.map((day) => (
        <div
          className="flex items-center justify-between gap-3 border-b border-slate-100 px-3 py-2 last:border-b-0"
          key={day.date}
        >
          <div>
            <p className="text-sm font-medium text-slate-900">{day.date}</p>
            <p className="text-xs text-slate-500">
              {day.status === "fully_busy" ? "Fully busy" : "Partially busy"}
            </p>
          </div>
          <p className="text-right text-xs text-slate-600">
            {day.busyHours} busy hours
            <br />
            {day.busyBlockCount} block{day.busyBlockCount === 1 ? "" : "s"}
          </p>
        </div>
      ))}
    </div>
  );
}
