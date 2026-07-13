import { ReminderCard } from "./ReminderCard";
import type { TripReminder } from "@/entities/trip-reminder/model";

type ReminderTimelineProps = {
  reminders: TripReminder[];
  busy?: boolean;
  canEdit: boolean;
  currentUserId?: string | null;
  labels: {
    empty: string;
    overdue: string;
    today: string;
    thisWeek: string;
    later: string;
    completed: string;
    disabled: string;
    card: Parameters<typeof ReminderCard>[0]["labels"];
  };
  onComplete: (reminder: TripReminder) => void;
  onDisable: (reminder: TripReminder) => void;
  onEdit: (reminder: TripReminder) => void;
  onDelete: (reminder: TripReminder) => void;
};

export function ReminderTimeline({
  reminders,
  busy,
  canEdit,
  currentUserId,
  labels,
  onComplete,
  onDisable,
  onEdit,
  onDelete
}: ReminderTimelineProps) {
  const groups = groupReminders(reminders);
  const sections = [
    { key: "overdue", title: labels.overdue, items: groups.overdue },
    { key: "today", title: labels.today, items: groups.today },
    { key: "week", title: labels.thisWeek, items: groups.week },
    { key: "later", title: labels.later, items: groups.later },
    { key: "completed", title: labels.completed, items: groups.completed },
    { key: "disabled", title: labels.disabled, items: groups.disabled }
  ];

  if (reminders.length === 0) {
    return (
      <p className="rounded-md border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
        {labels.empty}
      </p>
    );
  }

  return (
    <div className="space-y-5">
      {sections
        .filter((section) => section.items.length > 0)
        .map((section) => (
          <section key={section.key} className="space-y-3">
            <h3 className="text-sm font-semibold uppercase tracking-normal text-slate-500">
              {section.title}
            </h3>
            <ul className="space-y-3">
              {section.items.map((reminder) => {
                const ownReminder =
                  Boolean(currentUserId) && reminder.assignedToUserId === currentUserId;
                const canAct = canEdit || ownReminder;
                return (
                  <ReminderCard
                    busy={busy}
                    canAct={canAct}
                    canEdit={canEdit}
                    key={reminder.id}
                    labels={labels.card}
                    onComplete={onComplete}
                    onDelete={onDelete}
                    onDisable={onDisable}
                    onEdit={onEdit}
                    reminder={reminder}
                  />
                );
              })}
            </ul>
          </section>
        ))}
    </div>
  );
}

function groupReminders(reminders: TripReminder[]) {
  const today = startOfDay(new Date());
  const weekEnd = new Date(today);
  weekEnd.setDate(weekEnd.getDate() + 7);

  const groups = {
    overdue: [] as TripReminder[],
    today: [] as TripReminder[],
    week: [] as TripReminder[],
    later: [] as TripReminder[],
    completed: [] as TripReminder[],
    disabled: [] as TripReminder[]
  };

  for (const reminder of [...reminders].sort(compareReminders)) {
    if (reminder.status === "completed") {
      groups.completed.push(reminder);
      continue;
    }
    if (reminder.status === "disabled" || reminder.status === "cancelled") {
      groups.disabled.push(reminder);
      continue;
    }
    const date = parseDate(reminder.triggerDate);
    if (date < today) {
      groups.overdue.push(reminder);
    } else if (date.getTime() === today.getTime()) {
      groups.today.push(reminder);
    } else if (date <= weekEnd) {
      groups.week.push(reminder);
    } else {
      groups.later.push(reminder);
    }
  }

  return groups;
}

function compareReminders(a: TripReminder, b: TripReminder) {
  const dateCompare = a.triggerDate.localeCompare(b.triggerDate);
  if (dateCompare !== 0) {
    return dateCompare;
  }
  return (a.triggerTime ?? "09:00").localeCompare(b.triggerTime ?? "09:00");
}

function parseDate(value: string) {
  const parsed = new Date(`${value}T00:00:00`);
  return Number.isNaN(parsed.getTime()) ? startOfDay(new Date()) : startOfDay(parsed);
}

function startOfDay(value: Date) {
  const next = new Date(value);
  next.setHours(0, 0, 0, 0);
  return next;
}
