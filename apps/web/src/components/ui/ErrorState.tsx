"use client";

import { useTranslations } from "next-intl";
import { cn } from "@/shared/lib/cn";
import { StateActionControl, type StateAction } from "./StateAction";
import { RetryButton } from "./RetryButton";

type ErrorStateProps = {
  title: string;
  description: string;
  errorCode?: string;
  retryAction?: { label?: string; onRetry: () => void; pending?: boolean };
  secondaryAction?: StateAction;
  developmentDetails?: string;
  compact?: boolean;
  className?: string;
};

export function ErrorState({
  title,
  description,
  errorCode,
  retryAction,
  secondaryAction,
  developmentDetails,
  compact = false,
  className
}: ErrorStateProps) {
  const t = useTranslations("errors");

  return (
    <section
      className={cn(
        "rounded-lg border border-red-200 bg-red-50 text-red-950",
        compact ? "p-4" : "p-5",
        className
      )}
      role="alert"
    >
      <h2 className={cn("font-semibold", compact ? "text-sm" : "text-base")}>{title}</h2>
      <p className={cn("leading-6 text-red-800", compact ? "mt-1 text-xs" : "mt-2 text-sm")}>
        {description}
      </p>
      {errorCode ? <p className="mt-2 text-xs text-red-700">{t("reference", { code: errorCode })}</p> : null}
      {retryAction || secondaryAction ? (
        <div className="mt-4 flex flex-wrap items-start gap-2">
          {retryAction ? (
            <RetryButton
              label={retryAction.label}
              onRetry={retryAction.onRetry}
              pending={retryAction.pending}
            />
          ) : null}
          {secondaryAction ? <StateActionControl action={secondaryAction} variant="ghost" /> : null}
        </div>
      ) : null}
      {process.env.NODE_ENV === "development" && developmentDetails ? (
        <details className="mt-4 text-xs text-red-800">
          <summary className="cursor-pointer font-medium">{t("technicalDetails")}</summary>
          <pre className="mt-2 max-h-40 overflow-auto whitespace-pre-wrap rounded bg-white/70 p-3">
            {developmentDetails}
          </pre>
        </details>
      ) : null}
    </section>
  );
}
