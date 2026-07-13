"use client";

import { useMemo, useState } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { AddReminderDialog } from "./AddReminderDialog";
import { EditReminderDialog } from "./EditReminderDialog";
import { GenerateRemindersDialog } from "./GenerateRemindersDialog";
import { ReminderFilters } from "./ReminderFilters";
import { ReminderSummaryCards } from "./ReminderSummaryCards";
import { ReminderTimeline } from "./ReminderTimeline";
import {
  useCompleteTripReminder,
  useCreateTripReminder,
  useDeleteTripReminder,
  useDisableTripReminder,
  useGenerateTripReminders,
  useTripReminders,
  useUpdateTripReminder
} from "@/hooks/useTripReminders";
import { getErrorMessage } from "@/lib/utils";
import type {
  CreateReminderInput,
  GenerateRemindersInput,
  ReminderListParams,
  ReminderSummary,
  TripReminder,
  UpdateReminderInput
} from "@/entities/trip-reminder/model";

type TripRemindersPanelProps = {
  tripId: string;
  canEdit: boolean;
  currentUserId?: string | null;
  enabled?: boolean;
};

const EMPTY_SUMMARY: ReminderSummary = {
  total: 0,
  pending: 0,
  completed: 0,
  overdue: 0,
  dueToday: 0,
  highPriorityPending: 0,
  assignedToMe: 0,
  stale: false
};

export function TripRemindersPanel({
  tripId,
  canEdit,
  currentUserId,
  enabled = true
}: TripRemindersPanelProps) {
  const t = useTranslations("tripReminders");
  const [filters, setFilters] = useState<ReminderListParams>({});
  const [localError, setLocalError] = useState<string | null>(null);
  const [showGenerate, setShowGenerate] = useState(false);
  const [showAdd, setShowAdd] = useState(false);
  const [editingReminder, setEditingReminder] = useState<TripReminder | null>(null);

  const query = useTripReminders(tripId, filters, { enabled });
  const generateMutation = useGenerateTripReminders(tripId);
  const createMutation = useCreateTripReminder(tripId);
  const updateMutation = useUpdateTripReminder(tripId);
  const completeMutation = useCompleteTripReminder(tripId);
  const disableMutation = useDisableTripReminder(tripId);
  const deleteMutation = useDeleteTripReminder(tripId);

  const reminders = query.data?.reminders ?? [];
  const summary = query.data?.summary ?? EMPTY_SUMMARY;
  const busy =
    generateMutation.isPending ||
    createMutation.isPending ||
    updateMutation.isPending ||
    completeMutation.isPending ||
    disableMutation.isPending ||
    deleteMutation.isPending;

  const labelMaps = useMemo(() => buildLabelMaps(t), [t]);

  if (!enabled) {
    return (
      <Card>
        <h2 className="text-lg font-semibold text-slate-950">{t("title")}</h2>
        <p className="mt-2 text-sm text-slate-600">{t("onlineOnly")}</p>
      </Card>
    );
  }

  function generateReminders(input: GenerateRemindersInput) {
    setLocalError(null);
    generateMutation.mutate(input, {
      onSuccess: () => setShowGenerate(false),
      onError: (error) => setLocalError(getErrorMessage(error, t("errors.generate")))
    });
  }

  function createReminder(input: CreateReminderInput) {
    setLocalError(null);
    createMutation.mutate(input, {
      onSuccess: () => setShowAdd(false),
      onError: (error) => setLocalError(getErrorMessage(error, t("errors.save")))
    });
  }

  function updateReminder(input: UpdateReminderInput) {
    if (!editingReminder) {
      return;
    }
    setLocalError(null);
    updateMutation.mutate(
      { reminderId: editingReminder.id, input },
      {
        onSuccess: () => setEditingReminder(null),
        onError: (error) => setLocalError(getErrorMessage(error, t("errors.save")))
      }
    );
  }

  function completeReminder(reminder: TripReminder) {
    setLocalError(null);
    completeMutation.mutate(
      { reminderId: reminder.id, completed: reminder.status !== "completed" },
      { onError: (error) => setLocalError(getErrorMessage(error, t("errors.complete"))) }
    );
  }

  function disableReminder(reminder: TripReminder) {
    setLocalError(null);
    disableMutation.mutate(
      { reminderId: reminder.id, disabled: reminder.status !== "disabled" },
      { onError: (error) => setLocalError(getErrorMessage(error, t("errors.disable"))) }
    );
  }

  function deleteReminder(reminder: TripReminder) {
    if (!window.confirm(t("confirmDelete"))) {
      return;
    }
    setLocalError(null);
    deleteMutation.mutate(reminder.id, {
      onError: (error) => setLocalError(getErrorMessage(error, t("errors.delete")))
    });
  }

  return (
    <Card>
      <div className="flex flex-col gap-4">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h2 className="text-lg font-semibold text-slate-950">{t("title")}</h2>
            <p className="mt-1 text-sm leading-6 text-slate-600">{t("description")}</p>
          </div>
          <div className="flex flex-wrap gap-2">
            {canEdit ? (
              <>
                <Button
                  disabled={busy}
                  onClick={() => setShowGenerate((open) => !open)}
                  size="sm"
                  type="button"
                  variant="primary"
                >
                  {t("generate")}
                </Button>
                <Button
                  disabled={busy}
                  onClick={() => {
                    setShowAdd((open) => !open);
                    setEditingReminder(null);
                  }}
                  size="sm"
                  type="button"
                  variant="secondary"
                >
                  {showAdd ? t("hideAdd") : t("add")}
                </Button>
              </>
            ) : null}
          </div>
        </div>

        {query.isLoading ? <p className="text-sm text-slate-500">{t("loading")}</p> : null}
        {query.isError ? (
          <p className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
            {getErrorMessage(query.error, t("errors.load"))}
          </p>
        ) : null}
        {localError ? (
          <p className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
            {localError}
          </p>
        ) : null}

        <ReminderSummaryCards
          labels={{
            pending: t("summary.pending"),
            overdue: t("summary.overdue"),
            dueToday: t("summary.dueToday"),
            highPriority: t("summary.highPriority"),
            assignedToMe: t("summary.assignedToMe")
          }}
          staleLabel={t("staleWarning")}
          summary={summary}
        />

        {showGenerate && canEdit ? (
          <GenerateRemindersDialog
            busy={busy}
            labels={{
              title: t("generateTitle"),
              mode: t("fields.mode"),
              categories: t("fields.categories"),
              instructions: t("fields.instructions"),
              instructionsPlaceholder: t("instructionsPlaceholder"),
              preserveManual: t("fields.preserveManual"),
              preserveCompleted: t("fields.preserveCompleted"),
              replaceGenerated: t("fields.replaceGenerated"),
              warning: t("officialWarning"),
              submit: t("generate"),
              cancel: t("cancel"),
              modes: labelMaps.modes,
              categoriesMap: labelMaps.categories
            }}
            onCancel={() => setShowGenerate(false)}
            onSubmit={generateReminders}
          />
        ) : null}

        {showAdd && canEdit ? (
          <AddReminderDialog
            busy={busy}
            currentUserId={currentUserId}
            labels={buildFormLabels(t, labelMaps)}
            onCancel={() => setShowAdd(false)}
            onError={setLocalError}
            onSubmit={createReminder}
          />
        ) : null}

        {editingReminder ? (
          <EditReminderDialog
            busy={busy}
            currentUserId={currentUserId}
            labels={buildFormLabels(t, labelMaps)}
            onCancel={() => setEditingReminder(null)}
            onError={setLocalError}
            onSubmit={updateReminder}
            reminder={editingReminder}
          />
        ) : null}

        <ReminderFilters
          filters={filters}
          labels={{
            category: t("fields.category"),
            status: t("fields.status"),
            allCategories: t("allCategories"),
            allStatuses: t("allStatuses"),
            assignedToMe: t("filters.assignedToMe"),
            highPriority: t("filters.highPriority"),
            upcomingOnly: t("filters.upcomingOnly"),
            categories: labelMaps.categories,
            statuses: labelMaps.statuses
          }}
          onChange={setFilters}
        />

        <ReminderTimeline
          busy={busy}
          canEdit={canEdit}
          currentUserId={currentUserId}
          labels={{
            empty: canEdit ? t("emptyEditable") : t("emptyReadonly"),
            overdue: t("groups.overdue"),
            today: t("groups.today"),
            thisWeek: t("groups.thisWeek"),
            later: t("groups.later"),
            completed: t("groups.completed"),
            disabled: t("groups.disabled"),
            card: {
              assignedTo: t("assignedTo"),
              unassigned: t("unassigned"),
              day: t("day"),
              checklistLinked: t("checklistLinked"),
              complete: t("complete"),
              reopen: t("reopen"),
              disable: t("disable"),
              enable: t("enable"),
              edit: t("edit"),
              delete: t("delete"),
              categories: labelMaps.categories,
              priorities: labelMaps.priorities,
              statuses: labelMaps.statuses,
              sources: labelMaps.sources
            }
          }}
          onComplete={completeReminder}
          onDelete={deleteReminder}
          onDisable={disableReminder}
          onEdit={(reminder) => {
            setShowAdd(false);
            setEditingReminder(reminder);
          }}
          reminders={reminders}
        />
      </div>
    </Card>
  );
}

function buildLabelMaps(t: (key: string) => string) {
  const categories = mapKeys(t, "categories", [
    "documents",
    "packing",
    "transport",
    "accommodation",
    "weather",
    "activities",
    "group",
    "checklist",
    "before_departure",
    "route",
    "safety",
    "other"
  ]);
  return {
    categories,
    priorities: mapKeys(t, "priorities", ["low", "medium", "high", "critical"]),
    statuses: mapKeys(t, "statuses", [
      "pending",
      "sent",
      "completed",
      "disabled",
      "cancelled",
      "failed"
    ]),
    sources: mapKeys(t, "sources", [
      "checklist",
      "route",
      "transport",
      "accommodation",
      "weather",
      "manual",
      "system",
      "regenerated"
    ]),
    modes: mapKeys(t, "modes", ["full", "add_missing", "category"])
  };
}

function buildFormLabels(
  t: (key: string) => string,
  labelMaps: ReturnType<typeof buildLabelMaps>
) {
  return {
    title: t("fields.title"),
    description: t("fields.description"),
    category: t("fields.category"),
    priority: t("fields.priority"),
    triggerDate: t("fields.triggerDate"),
    triggerTime: t("fields.triggerTime"),
    timezone: t("fields.timezone"),
    assignee: t("fields.assignee"),
    assigneePlaceholder: t("fields.assigneePlaceholder"),
    assignMe: t("assignMe"),
    cancel: t("cancel"),
    save: t("saveChanges"),
    create: t("create"),
    titleRequired: t("errors.titleRequired"),
    dateRequired: t("errors.dateRequired"),
    categories: labelMaps.categories,
    priorities: labelMaps.priorities
  };
}

function mapKeys(t: (key: string) => string, prefix: string, keys: string[]) {
  return Object.fromEntries(keys.map((key) => [key, t(`${prefix}.${key}`)]));
}
