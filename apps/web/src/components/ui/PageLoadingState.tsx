"use client";

import { useTranslations } from "next-intl";
import { cn } from "@/shared/lib/cn";
import { CardSkeleton } from "./CardSkeleton";

type PageLoadingStateProps = {
  label?: string;
  cardCount?: number;
  className?: string;
  showHeader?: boolean;
};

export function PageLoadingState({
  label,
  cardCount = 4,
  className,
  showHeader = true
}: PageLoadingStateProps) {
  const t = useTranslations("loading");
  const accessibleLabel = label ?? t("page");

  return (
    <div
      aria-busy="true"
      aria-label={accessibleLabel}
      className={cn("animate-pulse space-y-6", className)}
      role="status"
    >
      <span className="sr-only">{accessibleLabel}</span>
      {showHeader ? (
        <div aria-hidden="true" className="rounded-lg border border-slate-200 bg-white p-5">
          <div className="h-4 w-28 rounded-full bg-slate-200" />
          <div className="mt-4 h-8 w-full max-w-lg rounded-full bg-slate-200" />
          <div className="mt-3 h-4 w-full max-w-2xl rounded-full bg-slate-100" />
        </div>
      ) : null}
      <div className="grid gap-4 md:grid-cols-2">
        {Array.from({ length: cardCount }, (_, index) => (
          <CardSkeleton key={index} lines={index % 2 === 0 ? 3 : 4} />
        ))}
      </div>
    </div>
  );
}
