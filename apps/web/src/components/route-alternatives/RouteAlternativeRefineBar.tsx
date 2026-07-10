"use client";

import { FormEvent, useState } from "react";

const QUICK_REFINEMENTS = [
  "Cheaper",
  "More relaxed",
  "More nature",
  "Fewer stops",
  "No flights",
  "More train-friendly",
  "More camping",
  "More hiking",
  "Shorter transfers",
  "Different countries"
];

type RouteAlternativeRefineBarProps = {
  disabled?: boolean;
  isPending?: boolean;
  onRefine: (instruction: string) => void;
};

export function RouteAlternativeRefineBar({
  disabled = false,
  isPending = false,
  onRefine
}: RouteAlternativeRefineBarProps) {
  const [instruction, setInstruction] = useState("");

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const value = instruction.trim();
    if (!value) {
      return;
    }
    onRefine(value);
    setInstruction("");
  }

  return (
    <div className="rounded-[18px] border border-sand-300 bg-white p-4">
      <div className="flex flex-wrap gap-2">
        {QUICK_REFINEMENTS.map((label) => (
          <button
            key={label}
            type="button"
            disabled={disabled || isPending}
            onClick={() => onRefine(label)}
            className="h-9 rounded-full border border-sand-300 px-3 text-[12.5px] font-semibold text-cocoa-600 transition hover:border-clay hover:text-clay disabled:cursor-not-allowed disabled:opacity-50"
          >
            {label}
          </button>
        ))}
      </div>
      <form onSubmit={submit} className="mt-3 flex flex-col gap-2 sm:flex-row">
        <input
          value={instruction}
          onChange={(event) => setInstruction(event.target.value)}
          disabled={disabled || isPending}
          placeholder="Tell us what to change..."
          className="h-11 min-w-0 flex-1 rounded-[12px] border border-sand-400 bg-[#FFFDFA] px-3 text-[14px] text-cocoa-900 outline-none transition placeholder:text-cocoa-400 focus:border-clay focus:ring-[3px] focus:ring-clay-tint disabled:cursor-not-allowed disabled:opacity-60"
        />
        <button
          type="submit"
          disabled={disabled || isPending || !instruction.trim()}
          className="h-11 rounded-full bg-cocoa-900 px-5 text-[13px] font-semibold text-sand-100 transition hover:bg-cocoa-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {isPending ? "Refining..." : "Refine routes"}
        </button>
      </form>
    </div>
  );
}
