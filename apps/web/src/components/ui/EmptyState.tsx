import type { ReactNode } from "react";
import { cn } from "@/shared/lib/cn";
import { StateActionControl, type StateAction } from "./StateAction";

type EmptyStateProps = {
  title: string;
  description: string;
  icon?: ReactNode;
  primaryAction?: StateAction;
  secondaryAction?: StateAction;
  compact?: boolean;
  className?: string;
};

export function EmptyState({
  title,
  description,
  icon,
  primaryAction,
  secondaryAction,
  compact = false,
  className
}: EmptyStateProps) {
  return (
    <section
      className={cn(
        "rounded-lg border border-dashed border-slate-300 bg-slate-50 text-center",
        compact ? "p-4" : "px-5 py-8",
        className
      )}
    >
      {icon ? (
        <div aria-hidden="true" className="mx-auto mb-3 flex h-10 w-10 items-center justify-center text-slate-500">
          {icon}
        </div>
      ) : null}
      <h2 className={cn("font-semibold text-slate-950", compact ? "text-sm" : "text-base")}>
        {title}
      </h2>
      <p className={cn("mx-auto max-w-xl leading-6 text-slate-600", compact ? "mt-1 text-xs" : "mt-2 text-sm")}>
        {description}
      </p>
      {primaryAction || secondaryAction ? (
        <div className={cn("flex flex-wrap items-start justify-center gap-2", compact ? "mt-3" : "mt-5")}>
          {primaryAction ? <StateActionControl action={primaryAction} /> : null}
          {secondaryAction ? <StateActionControl action={secondaryAction} variant="secondary" /> : null}
        </div>
      ) : null}
    </section>
  );
}

export type { StateAction as EmptyStateAction } from "./StateAction";
