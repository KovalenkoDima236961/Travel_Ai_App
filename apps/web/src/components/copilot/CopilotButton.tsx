"use client";

import { useTranslations } from "next-intl";

type CopilotButtonProps = {
  onClick: () => void;
};

export function CopilotButton({ onClick }: CopilotButtonProps) {
  const t = useTranslations("copilot");
  return (
    <button
      aria-label={t("open")}
      className="fixed bottom-5 right-5 z-40 inline-flex h-12 items-center gap-2 rounded-full bg-cocoa-900 px-4 text-sm font-semibold text-white shadow-soft transition hover:bg-cocoa-700 focus:outline-none focus:ring-2 focus:ring-primary-600 focus:ring-offset-2 sm:bottom-7 sm:right-7"
      onClick={onClick}
      type="button"
    >
      <svg aria-hidden="true" className="h-5 w-5" fill="none" viewBox="0 0 24 24">
        <path d="M12 3a7 7 0 0 0-7 7v4a4 4 0 0 0 4 4h3l3 3v-3a4 4 0 0 0 4-4v-4a7 7 0 0 0-7-7Z" stroke="currentColor" strokeWidth="1.8" />
        <path d="M9 11h.01M12 11h.01M15 11h.01" stroke="currentColor" strokeLinecap="round" strokeWidth="2.5" />
      </svg>
      <span className="hidden sm:inline">{t("title")}</span>
    </button>
  );
}
