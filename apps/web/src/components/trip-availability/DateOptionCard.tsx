"use client";

import { Button } from "@/shared/ui/button";
import type { TripDateOption } from "@/types/trip-availability";

type DateOptionCardProps = {
  canApply: boolean;
  checked: boolean;
  disabled?: boolean;
  isRecommended?: boolean;
  option: TripDateOption;
  onApply: (option: TripDateOption) => void;
  onCheckedChange: (checked: boolean) => void;
};

export function DateOptionCard({
  canApply,
  checked,
  disabled = false,
  isRecommended = false,
  option,
  onApply,
  onCheckedChange
}: DateOptionCardProps) {
  return (
    <div className="rounded-[14px] border border-sand-300 bg-white p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <label className="flex min-w-0 items-start gap-3">
          <input
            checked={checked}
            className="mt-1 h-4 w-4 rounded border-sand-300 text-clay focus:ring-clay"
            disabled={disabled}
            onChange={(event) => onCheckedChange(event.target.checked)}
            type="checkbox"
          />
          <span className="min-w-0">
            <span className="block font-semibold text-cocoa-900">
              {formatDate(option.startDate)} - {formatDate(option.endDate)}
            </span>
            <span className="mt-1 block text-[13px] text-cocoa-500">
              {option.durationDays} days · score {option.score}
              {isRecommended ? " · recommended" : ""}
            </span>
          </span>
        </label>
        {canApply ? (
          <Button
            disabled={disabled}
            onClick={() => onApply(option)}
            size="sm"
            type="button"
          >
            Apply
          </Button>
        ) : null}
      </div>

      <div className="mt-3 grid gap-2 text-[13px] sm:grid-cols-4">
        <MiniMetric label="Available" value={`${option.availableUserCount}/${option.totalUserCount}`} />
        <MiniMetric label="Preferred" value={String(option.preferredUserCount)} />
        <MiniMetric label="Conflicts" value={String(option.conflictUserCount)} />
        <MiniMetric label="Missing" value={String(option.missingResponseUserCount)} />
      </div>

      {option.pros.length > 0 || option.cons.length > 0 ? (
        <div className="mt-3 grid gap-3 text-[13px] lg:grid-cols-2">
          <List title="Pros" items={option.pros} />
          <List title="Tradeoffs" items={[...option.cons, ...option.warnings]} />
        </div>
      ) : null}
    </div>
  );
}

function MiniMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[12px] bg-sand-50 px-3 py-2">
      <p className="text-[11px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
        {label}
      </p>
      <p className="mt-1 font-semibold text-cocoa-800">{value}</p>
    </div>
  );
}

function List({ title, items }: { title: string; items: string[] }) {
  if (items.length === 0) {
    return null;
  }
  return (
    <div>
      <p className="font-semibold text-cocoa-700">{title}</p>
      <ul className="mt-1 space-y-1 text-cocoa-500">
        {items.slice(0, 3).map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
    </div>
  );
}

function formatDate(value: string) {
  if (!value) {
    return "Open date";
  }
  return value;
}
