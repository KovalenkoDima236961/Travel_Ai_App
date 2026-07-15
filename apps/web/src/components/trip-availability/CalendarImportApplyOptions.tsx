import type {
  CalendarImportConversionSettings,
  CalendarImportMode
} from "@/types/calendar-free-busy";

type CalendarImportApplyOptionsProps = {
  conversion: CalendarImportConversionSettings;
  mode: CalendarImportMode;
  onConversionChange: (conversion: CalendarImportConversionSettings) => void;
  onModeChange: (mode: CalendarImportMode) => void;
};

export function CalendarImportApplyOptions({
  conversion,
  mode,
  onConversionChange,
  onModeChange
}: CalendarImportApplyOptionsProps) {
  return (
    <div className="space-y-3">
      <label className="block text-sm font-medium text-slate-700">
        Fully busy threshold hours
        <input
          className="mt-1 h-10 w-full rounded-md border border-slate-300 px-3 text-sm"
          min={1}
          max={24}
          onChange={(event) =>
            onConversionChange({
              ...conversion,
              fullyBusyThresholdHours: Number(event.target.value) || 6
            })
          }
          type="number"
          value={conversion.fullyBusyThresholdHours}
        />
      </label>
      <Checkbox
        checked={conversion.markFullyBusyDaysUnavailable}
        label="Mark fully busy days as unavailable"
        onChange={(checked) =>
          onConversionChange({ ...conversion, markFullyBusyDaysUnavailable: checked })
        }
      />
      <Checkbox
        checked={conversion.markPartiallyBusyDaysUnavailable}
        label="Mark partially busy days as unavailable"
        onChange={(checked) =>
          onConversionChange({ ...conversion, markPartiallyBusyDaysUnavailable: checked })
        }
      />
      <Checkbox
        checked={conversion.includeWeekendsAsPreferredIfFree}
        label="Suggest free weekends as preferred dates"
        onChange={(checked) =>
          onConversionChange({ ...conversion, includeWeekendsAsPreferredIfFree: checked })
        }
      />
      <div className="grid gap-2 sm:grid-cols-2">
        <label className="flex items-center gap-2 rounded-md border border-slate-200 p-3 text-sm text-slate-700">
          <input
            checked={mode === "merge"}
            onChange={() => onModeChange("merge")}
            type="radio"
          />
          Merge with my availability
        </label>
        <label className="flex items-center gap-2 rounded-md border border-slate-200 p-3 text-sm text-slate-700">
          <input
            checked={mode === "overwrite_all_my_availability"}
            onChange={() => onModeChange("overwrite_all_my_availability")}
            type="radio"
          />
          Replace my availability
        </label>
      </div>
    </div>
  );
}

function Checkbox({
  checked,
  label,
  onChange
}: {
  checked: boolean;
  label: string;
  onChange: (checked: boolean) => void;
}) {
  return (
    <label className="flex items-center gap-2 text-sm text-slate-700">
      <input
        checked={checked}
        onChange={(event) => onChange(event.target.checked)}
        type="checkbox"
      />
      {label}
    </label>
  );
}
