"use client";

import Link from "next/link";
import { useId } from "react";
import { buttonStyles } from "@/shared/ui/button";

export type StateAction = {
  label: string;
  onClick?: () => void;
  href?: string;
  disabled?: boolean;
  disabledReason?: string;
  ariaLabel?: string;
};

export function StateActionControl({
  action,
  variant = "primary"
}: {
  action: StateAction;
  variant?: "primary" | "secondary" | "ghost" | "danger";
}) {
  const helpId = useId();
  const commonProps = {
    "aria-label": action.ariaLabel,
    "aria-describedby": action.disabledReason ? helpId : undefined,
    className: buttonStyles({ variant, size: "sm" })
  };

  return (
    <span className="inline-flex flex-col items-start gap-1">
      {action.href && !action.disabled ? (
        <Link {...commonProps} href={action.href}>
          {action.label}
        </Link>
      ) : (
        <button
          {...commonProps}
          disabled={action.disabled}
          onClick={action.onClick}
          type="button"
        >
          {action.label}
        </button>
      )}
      {action.disabled && action.disabledReason ? (
        <span id={helpId} className="max-w-xs text-xs text-slate-500">
          {action.disabledReason}
        </span>
      ) : null}
    </span>
  );
}
