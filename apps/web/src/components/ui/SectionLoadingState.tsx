"use client";

import { useTranslations } from "next-intl";
import { cn } from "@/shared/lib/cn";
import { CardSkeleton } from "./CardSkeleton";

type SectionLoadingStateProps = {
  label?: string;
  cards?: number;
  className?: string;
  compact?: boolean;
};

export function SectionLoadingState({
  label,
  cards = 1,
  className,
  compact = false
}: SectionLoadingStateProps) {
  const t = useTranslations("loading");

  return (
    <section
      aria-busy="true"
      aria-label={label ?? t("section")}
      className={cn("space-y-3", className)}
      role="status"
    >
      <span className="sr-only">{label ?? t("section")}</span>
      <div className={cn("grid gap-3", cards > 1 && "md:grid-cols-2")}>
        {Array.from({ length: cards }, (_, index) => (
          <CardSkeleton compact={compact} key={index} />
        ))}
      </div>
    </section>
  );
}
