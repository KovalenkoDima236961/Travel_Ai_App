import { Select } from "@/shared/ui/select";
import type {
  ReminderListParams,
  TripReminderCategory,
  TripReminderStatus
} from "@/entities/trip-reminder/model";
import {
  TRIP_REMINDER_CATEGORIES,
  TRIP_REMINDER_STATUSES
} from "@/entities/trip-reminder/model";

type ReminderFiltersProps = {
  filters: ReminderListParams;
  labels: {
    category: string;
    status: string;
    allCategories: string;
    allStatuses: string;
    assignedToMe: string;
    highPriority: string;
    upcomingOnly: string;
    categories: Record<string, string>;
    statuses: Record<string, string>;
  };
  onChange: (filters: ReminderListParams) => void;
};

export function ReminderFilters({ filters, labels, onChange }: ReminderFiltersProps) {
  return (
    <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-5">
      <label className="text-sm font-medium text-slate-700">
        {labels.category}
        <Select
          className="mt-1"
          onChange={(event) =>
            onChange({
              ...filters,
              category: event.target.value as TripReminderCategory | "all"
            })
          }
          value={filters.category ?? "all"}
        >
          <option value="all">{labels.allCategories}</option>
          {TRIP_REMINDER_CATEGORIES.map((category) => (
            <option key={category} value={category}>
              {labels.categories[category] ?? category}
            </option>
          ))}
        </Select>
      </label>
      <label className="text-sm font-medium text-slate-700">
        {labels.status}
        <Select
          className="mt-1"
          onChange={(event) =>
            onChange({
              ...filters,
              status: event.target.value as TripReminderStatus | "all"
            })
          }
          value={filters.status ?? "all"}
        >
          <option value="all">{labels.allStatuses}</option>
          {TRIP_REMINDER_STATUSES.map((status) => (
            <option key={status} value={status}>
              {labels.statuses[status] ?? status}
            </option>
          ))}
        </Select>
      </label>
      <ToggleFilter
        checked={Boolean(filters.assignedToMe)}
        label={labels.assignedToMe}
        onChange={(checked) => onChange({ ...filters, assignedToMe: checked })}
      />
      <ToggleFilter
        checked={Boolean(filters.highPriority)}
        label={labels.highPriority}
        onChange={(checked) => onChange({ ...filters, highPriority: checked })}
      />
      <ToggleFilter
        checked={Boolean(filters.upcomingOnly)}
        label={labels.upcomingOnly}
        onChange={(checked) => onChange({ ...filters, upcomingOnly: checked })}
      />
    </div>
  );
}

function ToggleFilter({
  checked,
  label,
  onChange
}: {
  checked: boolean;
  label: string;
  onChange: (checked: boolean) => void;
}) {
  return (
    <label className="flex min-h-16 items-center gap-2 rounded-md border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-700">
      <input
        checked={checked}
        className="h-4 w-4 rounded border-slate-300 text-primary-600 focus:ring-primary-500"
        onChange={(event) => onChange(event.target.checked)}
        type="checkbox"
      />
      {label}
    </label>
  );
}
