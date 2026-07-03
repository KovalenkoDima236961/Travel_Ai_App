"use client";

import type { ReactNode } from "react";

type DisabledActionTooltipProps = {
  children: ReactNode;
  disabled: boolean;
  message?: string;
};

export function DisabledActionTooltip({
  children,
  disabled,
  message = "This action requires an internet connection."
}: DisabledActionTooltipProps) {
  return (
    <span className="inline-flex" title={disabled ? message : undefined}>
      {children}
    </span>
  );
}
