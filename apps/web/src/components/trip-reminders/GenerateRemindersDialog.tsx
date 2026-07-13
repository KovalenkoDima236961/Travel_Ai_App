import { FormEvent, useState } from "react";
import { Button } from "@/shared/ui/button";
import { Select } from "@/shared/ui/select";
import { Textarea } from "@/shared/ui/textarea";
import type {
  GenerateRemindersInput,
  GenerateRemindersMode,
  TripReminderCategory
} from "@/entities/trip-reminder/model";
import { TRIP_REMINDER_CATEGORIES } from "@/entities/trip-reminder/model";

type GenerateRemindersDialogProps = {
  busy?: boolean;
  labels: {
    title: string;
    mode: string;
    categories: string;
    instructions: string;
    instructionsPlaceholder: string;
    preserveManual: string;
    preserveCompleted: string;
    replaceGenerated: string;
    warning: string;
    submit: string;
    cancel: string;
    modes: Record<string, string>;
    categoriesMap: Record<string, string>;
  };
  onCancel: () => void;
  onSubmit: (input: GenerateRemindersInput) => void;
};

export function GenerateRemindersDialog({
  busy,
  labels,
  onCancel,
  onSubmit
}: GenerateRemindersDialogProps) {
  const [mode, setMode] = useState<GenerateRemindersMode>("add_missing");
  const [selected, setSelected] = useState<TripReminderCategory[]>([]);
  const [instructions, setInstructions] = useState("");
  const [preserveManual, setPreserveManual] = useState(true);
  const [preserveCompleted, setPreserveCompleted] = useState(true);
  const [replaceGenerated, setReplaceGenerated] = useState(false);

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    onSubmit({
      mode,
      categories: mode === "category" ? selected : [],
      instructions,
      preserveManualReminders: preserveManual,
      preserveCompletedReminders: preserveCompleted,
      replaceGeneratedPendingReminders: replaceGenerated
    });
  }

  return (
    <form className="rounded-md border border-slate-200 bg-slate-50 p-3" onSubmit={submit}>
      <div className="flex flex-col gap-3">
        <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
          <h3 className="text-sm font-semibold text-slate-950">{labels.title}</h3>
          <p className="text-sm text-amber-900">{labels.warning}</p>
        </div>
        <label className="text-sm font-medium text-slate-700">
          {labels.mode}
          <Select
            className="mt-1"
            disabled={busy}
            onChange={(event) => setMode(event.target.value as GenerateRemindersMode)}
            value={mode}
          >
            <option value="add_missing">{labels.modes.add_missing}</option>
            <option value="full">{labels.modes.full}</option>
            <option value="category">{labels.modes.category}</option>
          </Select>
        </label>
        {mode === "category" ? (
          <fieldset className="grid gap-2 rounded-md border border-slate-200 bg-white p-3 sm:grid-cols-2 lg:grid-cols-3">
            <legend className="px-1 text-sm font-medium text-slate-700">
              {labels.categories}
            </legend>
            {TRIP_REMINDER_CATEGORIES.map((category) => (
              <label key={category} className="flex items-center gap-2 text-sm text-slate-700">
                <input
                  checked={selected.includes(category)}
                  disabled={busy}
                  onChange={(event) => {
                    setSelected((current) =>
                      event.target.checked
                        ? [...current, category]
                        : current.filter((value) => value !== category)
                    );
                  }}
                  type="checkbox"
                />
                {labels.categoriesMap[category] ?? category}
              </label>
            ))}
          </fieldset>
        ) : null}
        <label className="text-sm font-medium text-slate-700">
          {labels.instructions}
          <Textarea
            className="mt-1"
            disabled={busy}
            maxLength={1000}
            onChange={(event) => setInstructions(event.target.value)}
            placeholder={labels.instructionsPlaceholder}
            value={instructions}
          />
        </label>
        <div className="grid gap-2 md:grid-cols-3">
          <OptionCheckbox
            checked={preserveManual}
            disabled={busy}
            label={labels.preserveManual}
            onChange={setPreserveManual}
          />
          <OptionCheckbox
            checked={preserveCompleted}
            disabled={busy}
            label={labels.preserveCompleted}
            onChange={setPreserveCompleted}
          />
          <OptionCheckbox
            checked={replaceGenerated}
            disabled={busy}
            label={labels.replaceGenerated}
            onChange={setReplaceGenerated}
          />
        </div>
        <div className="flex flex-wrap justify-end gap-2">
          <Button disabled={busy} onClick={onCancel} type="button" variant="secondary">
            {labels.cancel}
          </Button>
          <Button disabled={busy} type="submit">
            {labels.submit}
          </Button>
        </div>
      </div>
    </form>
  );
}

function OptionCheckbox({
  checked,
  disabled,
  label,
  onChange
}: {
  checked: boolean;
  disabled?: boolean;
  label: string;
  onChange: (checked: boolean) => void;
}) {
  return (
    <label className="flex items-center gap-2 rounded-md border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-700">
      <input
        checked={checked}
        disabled={disabled}
        onChange={(event) => onChange(event.target.checked)}
        type="checkbox"
      />
      {label}
    </label>
  );
}
