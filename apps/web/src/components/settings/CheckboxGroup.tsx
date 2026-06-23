"use client";

import { useId } from "react";
import { cn } from "@/lib/utils";

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

  function toggle(optionValue: string, checked: boolean) {
    if (checked) {
      onChange([...value, optionValue]);
      return;
    }

    onChange(value.filter((item) => item !== optionValue));
  }

  return (
    <fieldset aria-describedby={description ? `${groupId}-description` : undefined}>
      <legend className="text-sm font-medium text-slate-800">{label}</legend>
      {description ? (
        <p id={`${groupId}-description`} className="mt-1 text-sm leading-6 text-slate-600">
          {description}
        </p>
      ) : null}
      <div className="mt-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {options.map((option) => {
          const inputId = `${groupId}-${option.value}`;
          return (
            <label
              key={option.value}
              className={cn(
                "flex min-h-11 items-center gap-3 rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-sm font-medium text-slate-700",
                disabled && "cursor-not-allowed opacity-70"
              )}
              htmlFor={inputId}
            >
              <input
                checked={selected.has(option.value)}
                className="h-4 w-4 rounded border-slate-300 text-primary-600 focus:ring-primary-600 disabled:cursor-not-allowed"
                disabled={disabled}
                id={inputId}
                type="checkbox"
                value={option.value}
                onChange={(event) => toggle(option.value, event.target.checked)}
              />
              <span>{option.label}</span>
            </label>
          );
        })}
      </div>
      {error ? <p className="mt-2 text-sm text-red-700">{error}</p> : null}
    </fieldset>
  );
}
