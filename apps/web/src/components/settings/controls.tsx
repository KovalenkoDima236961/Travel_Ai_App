"use client";

import type { ButtonHTMLAttributes, ReactNode } from "react";
import { cn } from "@/shared/lib/cn";

/**
 * Warm-palette settings primitives translated from the Settings.dc.html design.
 * These replace the slate `@/shared/ui/*` primitives *inside the settings screen
 * only* — the shared primitives still render on un-redesigned pages, so they are
 * left untouched. Co-located with the settings sections that consume them.
 */

export const FIELD_LABEL_CLASS = "block text-[13.5px] font-semibold text-cocoa-700";

export const INPUT_CLASS =
  "h-12 w-full rounded-xl border border-sand-400 bg-[#FFFDFA] px-3.5 text-[14.5px] text-cocoa-900 outline-none transition placeholder:text-cocoa-400 focus:border-clay focus:ring-[3px] focus:ring-clay-tint disabled:cursor-not-allowed disabled:opacity-60";

// Keeps the native dropdown affordance (the design's select is un-restyled),
// just wrapped in the warm border/background/focus treatment.
export const SELECT_CLASS = INPUT_CLASS;

export function SettingsCard({
  children,
  className
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <section
      className={cn("rounded-[20px] border border-sand-300 bg-white p-7 sm:px-8", className)}
    >
      {children}
    </section>
  );
}

export function SectionHeading({ title, subtitle }: { title: string; subtitle?: string }) {
  return (
    <div>
      <h2 className="font-newsreader text-[22px] font-semibold text-cocoa-900">{title}</h2>
      {subtitle ? <p className="mt-1.5 text-[14px] text-cocoa-400">{subtitle}</p> : null}
    </div>
  );
}

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement>;

export function PrimaryButton({ className, children, ...props }: ButtonProps) {
  return (
    <button
      className={cn(
        "inline-flex h-11 items-center justify-center rounded-full bg-clay px-[22px] text-[14.5px] font-semibold text-sand-100 transition hover:bg-clay-dark disabled:cursor-not-allowed disabled:opacity-60",
        className
      )}
      {...props}
    >
      {children}
    </button>
  );
}

export function GhostButton({ className, children, ...props }: ButtonProps) {
  return (
    <button
      className={cn(
        "inline-flex h-10 items-center justify-center rounded-full border border-sand-400 bg-white px-4 text-[13.5px] font-medium text-cocoa-700 transition hover:border-sand-600 hover:text-cocoa-900 disabled:cursor-not-allowed disabled:opacity-60",
        className
      )}
      {...props}
    >
      {children}
    </button>
  );
}

/**
 * The design's signature pill toggle. Rendered as a real button with switch
 * semantics so it stays keyboard-accessible.
 */
export function Switch({
  checked,
  onChange,
  disabled,
  label
}: {
  checked: boolean;
  onChange: (next: boolean) => void;
  disabled?: boolean;
  label: string;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        "relative inline-flex h-[26px] w-[46px] shrink-0 rounded-full transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-clay/40 disabled:cursor-not-allowed disabled:opacity-60",
        checked ? "bg-clay" : "bg-sand-400"
      )}
    >
      <span
        className={cn(
          "absolute top-[3px] h-5 w-5 rounded-full bg-white shadow-sm transition-[left]",
          checked ? "left-[23px]" : "left-[3px]"
        )}
      />
    </button>
  );
}

export function SaveNotice({
  successMessage,
  errorMessage
}: {
  successMessage?: string | null;
  errorMessage?: string | null;
}) {
  if (errorMessage) {
    return (
      <div
        className="rounded-xl border border-clay/30 bg-clay-tint/50 px-4 py-3 text-[13.5px] text-clay-deep"
        role="alert"
      >
        {errorMessage}
      </div>
    );
  }

  if (successMessage) {
    return (
      <div
        className="rounded-xl border border-[#3E6B5A]/25 bg-[#EAF2ED] px-4 py-3 text-[13.5px] text-[#2F5546]"
        role="status"
      >
        {successMessage}
      </div>
    );
  }

  return null;
}
