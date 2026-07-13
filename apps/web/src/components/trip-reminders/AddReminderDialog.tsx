import { FormEvent, useState } from "react";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Textarea } from "@/shared/ui/textarea";
import { ReminderAssigneeSelect } from "./ReminderAssigneeSelect";
import type {
  CreateReminderInput,
  TripReminder,
  TripReminderCategory,
  TripReminderPriority,
  UpdateReminderInput
} from "@/entities/trip-reminder/model";
import {
  TRIP_REMINDER_CATEGORIES,
  TRIP_REMINDER_PRIORITIES
} from "@/entities/trip-reminder/model";

export type ReminderFormLabels = {
  title: string;
  description: string;
  category: string;
  priority: string;
  triggerDate: string;
  triggerTime: string;
  timezone: string;
  assignee: string;
  assigneePlaceholder: string;
  assignMe: string;
  cancel: string;
  save: string;
  create: string;
  titleRequired: string;
  dateRequired: string;
  categories: Record<string, string>;
  priorities: Record<string, string>;
};

type ReminderFormState = {
  title: string;
  description: string;
  category: TripReminderCategory;
  priority: TripReminderPriority;
  triggerDate: string;
  triggerTime: string;
  timezone: string;
  assignedToUserId: string;
};

type AddReminderDialogProps = {
  busy?: boolean;
  currentUserId?: string | null;
  labels: ReminderFormLabels;
  onCancel: () => void;
  onError: (message: string) => void;
  onSubmit: (input: CreateReminderInput) => void;
};

export function AddReminderDialog({
  busy,
  currentUserId,
  labels,
  onCancel,
  onError,
  onSubmit
}: AddReminderDialogProps) {
  const [form, setForm] = useState<ReminderFormState>(emptyForm(defaultDate()));

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const title = form.title.trim();
    if (!title) {
      onError(labels.titleRequired);
      return;
    }
    if (!form.triggerDate) {
      onError(labels.dateRequired);
      return;
    }
    onSubmit(createPayload(form));
  }

  return (
    <ReminderForm
      busy={busy}
      currentUserId={currentUserId}
      form={form}
      labels={labels}
      mode="create"
      onCancel={onCancel}
      onChange={setForm}
      onSubmit={submit}
    />
  );
}

type EditReminderDialogProps = {
  busy?: boolean;
  currentUserId?: string | null;
  labels: ReminderFormLabels;
  reminder: TripReminder;
  onCancel: () => void;
  onError: (message: string) => void;
  onSubmit: (input: UpdateReminderInput) => void;
};

export function EditReminderDialog({
  busy,
  currentUserId,
  labels,
  reminder,
  onCancel,
  onError,
  onSubmit
}: EditReminderDialogProps) {
  const [form, setForm] = useState<ReminderFormState>(formFromReminder(reminder));

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const title = form.title.trim();
    if (!title) {
      onError(labels.titleRequired);
      return;
    }
    if (!form.triggerDate) {
      onError(labels.dateRequired);
      return;
    }
    onSubmit(updatePayload(form));
  }

  return (
    <ReminderForm
      busy={busy}
      currentUserId={currentUserId}
      form={form}
      labels={labels}
      mode="edit"
      onCancel={onCancel}
      onChange={setForm}
      onSubmit={submit}
    />
  );
}

function ReminderForm({
  busy,
  currentUserId,
  form,
  labels,
  mode,
  onCancel,
  onChange,
  onSubmit
}: {
  busy?: boolean;
  currentUserId?: string | null;
  form: ReminderFormState;
  labels: ReminderFormLabels;
  mode: "create" | "edit";
  onCancel: () => void;
  onChange: (form: ReminderFormState) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  return (
    <form className="rounded-md border border-slate-200 bg-slate-50 p-3" onSubmit={onSubmit}>
      <div className="grid gap-3 md:grid-cols-2">
        <label className="text-sm font-medium text-slate-700">
          {labels.title}
          <Input
            className="mt-1"
            disabled={busy}
            maxLength={140}
            onChange={(event) => onChange({ ...form, title: event.target.value })}
            value={form.title}
          />
        </label>
        <label className="text-sm font-medium text-slate-700">
          {labels.category}
          <Select
            className="mt-1"
            disabled={busy}
            onChange={(event) =>
              onChange({ ...form, category: event.target.value as TripReminderCategory })
            }
            value={form.category}
          >
            {TRIP_REMINDER_CATEGORIES.map((category) => (
              <option key={category} value={category}>
                {labels.categories[category] ?? category}
              </option>
            ))}
          </Select>
        </label>
        <label className="text-sm font-medium text-slate-700">
          {labels.priority}
          <Select
            className="mt-1"
            disabled={busy}
            onChange={(event) =>
              onChange({ ...form, priority: event.target.value as TripReminderPriority })
            }
            value={form.priority}
          >
            {TRIP_REMINDER_PRIORITIES.map((priority) => (
              <option key={priority} value={priority}>
                {labels.priorities[priority] ?? priority}
              </option>
            ))}
          </Select>
        </label>
        <label className="text-sm font-medium text-slate-700">
          {labels.triggerDate}
          <Input
            className="mt-1"
            disabled={busy}
            onChange={(event) => onChange({ ...form, triggerDate: event.target.value })}
            type="date"
            value={form.triggerDate}
          />
        </label>
        <label className="text-sm font-medium text-slate-700">
          {labels.triggerTime}
          <Input
            className="mt-1"
            disabled={busy}
            onChange={(event) => onChange({ ...form, triggerTime: event.target.value })}
            type="time"
            value={form.triggerTime}
          />
        </label>
        <label className="text-sm font-medium text-slate-700">
          {labels.timezone}
          <Input
            className="mt-1"
            disabled={busy}
            maxLength={80}
            onChange={(event) => onChange({ ...form, timezone: event.target.value })}
            placeholder="Europe/Bratislava"
            value={form.timezone}
          />
        </label>
        <div className="md:col-span-2">
          <ReminderAssigneeSelect
            assignMeLabel={labels.assignMe}
            currentUserId={currentUserId}
            disabled={busy}
            label={labels.assignee}
            onChange={(value) => onChange({ ...form, assignedToUserId: value })}
            placeholder={labels.assigneePlaceholder}
            value={form.assignedToUserId}
          />
        </div>
        <label className="text-sm font-medium text-slate-700 md:col-span-2">
          {labels.description}
          <Textarea
            className="mt-1"
            disabled={busy}
            maxLength={600}
            onChange={(event) => onChange({ ...form, description: event.target.value })}
            value={form.description}
          />
        </label>
      </div>
      <div className="mt-4 flex flex-wrap justify-end gap-2">
        <Button disabled={busy} onClick={onCancel} type="button" variant="secondary">
          {labels.cancel}
        </Button>
        <Button disabled={busy} type="submit">
          {mode === "edit" ? labels.save : labels.create}
        </Button>
      </div>
    </form>
  );
}

function emptyForm(triggerDate: string): ReminderFormState {
  return {
    title: "",
    description: "",
    category: "other",
    priority: "medium",
    triggerDate,
    triggerTime: "09:00",
    timezone: "",
    assignedToUserId: ""
  };
}

function formFromReminder(reminder: TripReminder): ReminderFormState {
  return {
    title: reminder.title,
    description: reminder.description ?? "",
    category: reminder.category,
    priority: reminder.priority,
    triggerDate: reminder.triggerDate,
    triggerTime: reminder.triggerTime ?? "",
    timezone: reminder.timezone ?? "",
    assignedToUserId: reminder.assignedToUserId ?? ""
  };
}

function createPayload(form: ReminderFormState): CreateReminderInput {
  return {
    title: form.title,
    description: form.description || null,
    category: form.category,
    priority: form.priority,
    triggerDate: form.triggerDate,
    triggerTime: form.triggerTime || null,
    timezone: form.timezone || null,
    assignedToUserId: form.assignedToUserId || null
  };
}

function updatePayload(form: ReminderFormState): UpdateReminderInput {
  return {
    title: form.title,
    description: form.description || null,
    clearDescription: !form.description.trim(),
    category: form.category,
    priority: form.priority,
    triggerDate: form.triggerDate,
    triggerTime: form.triggerTime || null,
    clearTriggerTime: !form.triggerTime.trim(),
    timezone: form.timezone || null,
    clearTimezone: !form.timezone.trim(),
    assignedToUserId: form.assignedToUserId || null,
    clearAssignee: !form.assignedToUserId.trim()
  };
}

function defaultDate() {
  return new Date().toISOString().slice(0, 10);
}
