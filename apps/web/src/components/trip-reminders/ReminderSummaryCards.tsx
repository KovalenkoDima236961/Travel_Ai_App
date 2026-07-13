import type { ReminderSummary } from "@/entities/trip-reminder/model";

type ReminderSummaryCardsProps = {
  summary: ReminderSummary;
  staleLabel: string;
  labels: {
    pending: string;
    overdue: string;
    dueToday: string;
    highPriority: string;
    assignedToMe: string;
  };
};

export function ReminderSummaryCards({
  summary,
  staleLabel,
  labels
}: ReminderSummaryCardsProps) {
  const items = [
    { label: labels.pending, value: summary.pending },
    { label: labels.overdue, value: summary.overdue },
    { label: labels.dueToday, value: summary.dueToday },
    { label: labels.highPriority, value: summary.highPriorityPending },
    { label: labels.assignedToMe, value: summary.assignedToMe }
  ];

  return (
    <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-5">
      {items.map((item) => (
        <div
          key={item.label}
          className="rounded-md border border-slate-200 bg-slate-50 px-3 py-2"
        >
          <p className="text-xs font-medium uppercase tracking-normal text-slate-500">
            {item.label}
          </p>
          <p className="mt-1 text-xl font-semibold text-slate-950">{item.value}</p>
        </div>
      ))}
      {summary.stale ? (
        <div className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 sm:col-span-2 lg:col-span-5">
          <p className="text-sm font-medium text-amber-900">{staleLabel}</p>
        </div>
      ) : null}
    </div>
  );
}
