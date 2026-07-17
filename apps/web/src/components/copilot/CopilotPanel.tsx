"use client";

import { useEffect } from "react";
import { useTranslations } from "next-intl";
import { CopilotDisclaimer } from "./CopilotDisclaimer";
import { CopilotErrorState } from "./CopilotErrorState";
import { CopilotInput } from "./CopilotInput";
import { CopilotLoadingState } from "./CopilotLoadingState";
import { CopilotMessageList } from "./CopilotMessageList";
import { CopilotSuggestedPrompts } from "./CopilotSuggestedPrompts";
import { useTripCopilot } from "@/hooks/useTripCopilot";

type CopilotPanelProps = {
  tripId: string;
  open: boolean;
  currentTab?: string;
  currentPath?: string;
  onClose: () => void;
};

export function CopilotPanel({ tripId, open, currentTab, currentPath, onClose }: CopilotPanelProps) {
  const t = useTranslations("copilot");
  const copilot = useTripCopilot(tripId, { currentTab, currentPath });

  useEffect(() => {
    if (!open) {
      return;
    }
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose();
      }
    };
    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [onClose, open]);

  if (!open) {
    return null;
  }
  return (
    <div className="fixed inset-0 z-50 bg-cocoa-900/25 sm:bg-transparent" role="dialog" aria-label={t("title")} aria-modal="true">
      <aside className="absolute inset-0 flex h-full flex-col bg-white shadow-2xl sm:bottom-4 sm:left-auto sm:right-4 sm:top-4 sm:w-[min(430px,calc(100vw-2rem))] sm:rounded-2xl sm:border sm:border-sand-300">
        <header className="flex items-start justify-between border-b border-sand-300 px-4 py-4">
          <div>
            <h2 className="font-newsreader text-xl font-semibold text-cocoa-900">{t("title")}</h2>
            <p className="mt-1 text-xs text-cocoa-500">{t("subtitle")}</p>
          </div>
          <button aria-label={t("close")} className="rounded-full p-2 text-cocoa-500 hover:bg-sand-100 hover:text-cocoa-900" onClick={onClose} type="button">
            <svg aria-hidden="true" className="h-5 w-5" fill="none" viewBox="0 0 24 24"><path d="m6 6 12 12M18 6 6 18" stroke="currentColor" strokeLinecap="round" strokeWidth="2" /></svg>
          </button>
        </header>
        <div className="min-h-0 flex-1 space-y-4 overflow-y-auto bg-sand-100 p-4">
          {copilot.conversation.messages.length === 0 ? <CopilotSuggestedPrompts prompts={copilot.suggestedPrompts} onSelect={(prompt) => void copilot.sendMessage(prompt)} /> : null}
          <CopilotMessageList messages={copilot.conversation.messages} onNavigate={onClose} />
          {copilot.isLoading ? <CopilotLoadingState /> : null}
          {copilot.error && !copilot.isLoading ? <CopilotErrorState onRetry={copilot.retry} tripId={tripId} /> : null}
        </div>
        <CopilotDisclaimer />
        <CopilotInput disabled={copilot.isLoading} onSend={(message) => void copilot.sendMessage(message)} />
      </aside>
    </div>
  );
}
