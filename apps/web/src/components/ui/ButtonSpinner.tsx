"use client";

import { useTranslations } from "next-intl";
import { cn } from "@/shared/lib/cn";

export function ButtonSpinner({ className }: { className?: string }) {
  const t = useTranslations("accessibility");
  return (
    <span aria-label={t("loading")} className={cn("inline-flex", className)} role="status">
      <span
        aria-hidden="true"
        className="h-4 w-4 animate-spin rounded-full border-2 border-current border-r-transparent"
      />
    </span>
  );
}
