"use client";

import { useEffect, useRef } from "react";
import { useTranslations } from "next-intl";
import { cn } from "@/shared/lib/cn";

export type FormErrorSummaryItem = {
  fieldId: string;
  message: string;
  label?: string;
};

export function FormErrorSummary({
  errors,
  title,
  focusOnMount = true,
  className
}: {
  errors: FormErrorSummaryItem[];
  title?: string;
  focusOnMount?: boolean;
  className?: string;
}) {
  const t = useTranslations("forms");
  const summaryRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (focusOnMount && errors.length > 0) {
      summaryRef.current?.focus();
    }
  }, [errors.length, focusOnMount]);

  if (errors.length === 0) {
    return null;
  }

  return (
    <div
      className={cn("rounded-lg border border-red-200 bg-red-50 p-4", className)}
      ref={summaryRef}
      role="alert"
      tabIndex={-1}
    >
      <h2 className="text-sm font-semibold text-red-950">{title ?? t("errorSummaryTitle")}</h2>
      <ul className="mt-2 list-disc space-y-1 pl-5 text-sm text-red-800">
        {errors.map((error) => (
          <li key={`${error.fieldId}:${error.message}`}>
            <a className="underline underline-offset-2" href={`#${error.fieldId}`}>
              {error.label ? `${error.label}: ${error.message}` : error.message}
            </a>
          </li>
        ))}
      </ul>
    </div>
  );
}
