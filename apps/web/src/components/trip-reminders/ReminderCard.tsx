import { Button } from "@/shared/ui/button";
import { formatDate } from "@/lib/utils";
import { isOfflinePending } from "@/lib/offline/cache-writer";
import {
  ReminderStatusBadge,
  reminderPriorityTone,
  reminderStatusTone
} from "./ReminderStatusBadge";
import type { TripReminder } from "@/entities/trip-reminder/model";

type ReminderCardProps = {
  reminder: TripReminder;
  busy?: boolean;
  canEdit: boolean;
  canAct: boolean;
  labels: {
    assignedTo: string;
    unassigned: string;
    day: string;
    checklistLinked: string;
    complete: string;
    reopen: string;
    disable: string;
    enable: string;
    edit: string;
    delete: string;
    categories: Record<string, string>;
    priorities: Record<string, string>;
    statuses: Record<string, string>;
    sources: Record<string, string>;
  };
  onComplete: (reminder: TripReminder) => void;
  onDisable: (reminder: TripReminder) => void;
  onEdit: (reminder: TripReminder) => void;
  onDelete: (reminder: TripReminder) => void;
};

export function ReminderCard({
  reminder,
  busy,
  canEdit,
  canAct,
  labels,
  onComplete,
  onDisable,
  onEdit,
  onDelete
}: ReminderCardProps) {
  const isCompleted = reminder.status === "completed";
  const isDisabled = reminder.status === "disabled";
  const canComplete = canAct && !isDisabled && reminder.status !== "cancelled";
  const canDisable = canAct && reminder.status !== "cancelled";
  const dueText = [formatDate(reminder.triggerDate), reminder.triggerTime]
    .filter(Boolean)
    .join(" ");

  return (
    <li className="rounded-lg border border-slate-200 bg-white p-3">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <p className="font-medium text-slate-950">{reminder.title}</p>
            <ReminderStatusBadge tone={reminderPriorityTone(reminder.priority)}>
              {labels.priorities[reminder.priority] ?? reminder.priority}
            </ReminderStatusBadge>
            <ReminderStatusBadge>
              {labels.categories[reminder.category] ?? reminder.category}
            </ReminderStatusBadge>
            <ReminderStatusBadge tone={reminderStatusTone(reminder.status)}>
              {labels.statuses[reminder.status] ?? reminder.status}
            </ReminderStatusBadge>
            {isOfflinePending(reminder.metadata) ? (
              <ReminderStatusBadge tone="warning">Pending sync</ReminderStatusBadge>
            ) : null}
          </div>
          {reminder.description ? (
            <p className="mt-2 text-sm leading-6 text-slate-600">{reminder.description}</p>
          ) : null}
          <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500">
            <span>{dueText}</span>
            <span>{labels.sources[reminder.source] ?? reminder.source}</span>
            <span>
              {labels.assignedTo}{" "}
              {reminder.assignedToDisplayName ||
                reminder.assignedToUserId ||
                labels.unassigned}
            </span>
            {reminder.relatedDayNumber ? (
              <span>{labels.day.replace("{day}", String(reminder.relatedDayNumber))}</span>
            ) : null}
            {reminder.checklistItemId ? <span>{labels.checklistLinked}</span> : null}
          </div>
          {reminder.failureReason ? (
            <p className="mt-2 text-xs leading-5 text-red-700">{reminder.failureReason}</p>
          ) : null}
        </div>
        <div className="flex flex-wrap gap-2 lg:justify-end">
          {canComplete ? (
            <Button
              disabled={busy}
              onClick={() => onComplete(reminder)}
              size="sm"
              type="button"
              variant={isCompleted ? "secondary" : "primary"}
            >
              {isCompleted ? labels.reopen : labels.complete}
            </Button>
          ) : null}
          {canDisable ? (
            <Button
              disabled={busy}
              onClick={() => onDisable(reminder)}
              size="sm"
              type="button"
              variant="secondary"
            >
              {isDisabled ? labels.enable : labels.disable}
            </Button>
          ) : null}
          {canEdit ? (
            <>
              <Button
                disabled={busy}
                onClick={() => onEdit(reminder)}
                size="sm"
                type="button"
                variant="secondary"
              >
                {labels.edit}
              </Button>
              <Button
                disabled={busy}
                onClick={() => onDelete(reminder)}
                size="sm"
                type="button"
                variant="danger"
              >
                {labels.delete}
              </Button>
            </>
          ) : null}
        </div>
      </div>
    </li>
  );
}
