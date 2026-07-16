"use client";

import { useEffect } from "react";
import { useTranslations } from "next-intl";
import type { RouteImpact } from "@/lib/route-builder/route-draft";
import { RouteImpactSummary } from "./RouteImpactSummary";

type RouteImpactPreviewDialogProps = {
  open: boolean;
  impact: RouteImpact;
  pending?: boolean;
  error?: string | null;
  onCancel: () => void;
  onConfirm: () => void;
};

export function RouteImpactPreviewDialog({
  open,
  impact,
  pending = false,
  error,
  onCancel,
  onConfirm
}: RouteImpactPreviewDialogProps) {
  const t = useTranslations("route");
  useEffect(() => {
    if (!open) {
      return;
    }
    const handleKey = (event: KeyboardEvent) => {
      if (event.key === "Escape" && !pending) {
        onCancel();
      }
    };
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [onCancel, open, pending]);

  if (!open) {
    return null;
  }
  return (
    <div className="fixed inset-0 z-[80] flex items-end justify-center bg-cocoa-900/45 p-0 sm:items-center sm:p-5">
      <section
        aria-labelledby="route-impact-title"
        aria-modal="true"
        className="max-h-[92vh] w-full overflow-y-auto rounded-t-[22px] bg-white p-5 shadow-2xl sm:max-w-[560px] sm:rounded-[22px] sm:p-6"
        role="dialog"
      >
        <h2 id="route-impact-title" className="font-newsreader text-[24px] font-semibold text-cocoa-900">
          {t("impactPreview")}
        </h2>
        <p className="mt-2 text-[14px] leading-6 text-cocoa-500">{t("impactPreviewDescription")}</p>
        <div className="mt-4"><RouteImpactSummary impact={impact} /></div>
        {error ? <p role="alert" className="mt-3 text-[13px] font-medium text-red-700">{error}</p> : null}
        <div className="mt-5 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <button
            className="h-11 rounded-full border border-sand-400 px-5 text-[14px] font-semibold text-cocoa-700 disabled:opacity-50"
            disabled={pending}
            onClick={onCancel}
            type="button"
          >
            {t("keepEditing")}
          </button>
          <button
            className="h-11 rounded-full bg-cocoa-900 px-5 text-[14px] font-semibold text-white transition hover:bg-cocoa-700 disabled:opacity-50"
            disabled={pending}
            onClick={onConfirm}
            type="button"
          >
            {pending ? t("saving") : t("saveRouteChanges")}
          </button>
        </div>
      </section>
    </div>
  );
}
