"use client";

import { useId } from "react";
import { cn } from "@/shared/lib/cn";
import { FIELD_LABEL_CLASS } from "@/components/settings/controls";

type CheckboxOption = {
  label: string;
  value: string;
};

type CheckboxGroupProps = {
  label: string;
  description?: string;
  options: CheckboxOption[];
  value: string[];
  onChange: (next: string[]) => void;
  error?: string;
  disabled?: boolean;
};

/**
 * Multi-select rendered as toggleable pills, matching the "Default interests"
 * control in the Settings design. Kept as a fieldset of pressable buttons so it
 * stays keyboard-accessible while dropping the boxy checkbox look.
 */
export function CheckboxGroup({
  label,
  description,
  options,
  value,
  onChange,
  error,
  disabled = false
}: CheckboxGroupProps) {
  const groupId = useId();
  const selected = new Set(value);

  function toggle(optionValue: string) {
    if (selected.has(optionValue)) {
      onChange(value.filter((item) => item !== optionValue));
      return;
    }
    onChange([...value, optionValue]);
  }

  return (
    <fieldset aria-describedby={description ? `${groupId}-description` : undefined}>
      <legend className={FIELD_LABEL_CLASS}>{label}</legend>
      {description ? (
        <p id={`${groupId}-description`} className="mt-1 text-[13px] text-cocoa-400">
          {description}
        </p>
      ) : null}
      <div className="mt-2.5 flex flex-wrap gap-2">
        {options.map((option) => {
          const isSelected = selected.has(option.value);
          return (
            <button
              key={option.value}
              type="button"
              aria-pressed={isSelected}
              disabled={disabled}
              onClick={() => toggle(option.value)}
              className={cn(
                "rounded-full border px-3.5 py-1.5 text-[13px] transition disabled:cursor-not-allowed disabled:opacity-60",
                isSelected
                  ? "border-clay bg-clay-tint font-semibold text-clay-deep"
                  : "border-sand-400 bg-white font-medium text-cocoa-500 hover:border-sand-600 hover:text-cocoa-700"
              )}
            >
              {option.label}
            </button>
          );
        })}
      </div>
      {error ? <p className="mt-2 text-[13px] text-clay-deep">{error}</p> : null}
    </fieldset>
  );
}
