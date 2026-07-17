"use client";

import { useCallback, useMemo, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { sendTripCopilotMessage } from "@/lib/api/copilot";
import type {
  CopilotClientContext,
  CopilotConversation,
  CopilotMessage,
  CopilotSuggestedPrompt
} from "@/types/copilot";

function messageId() {
  return typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : `copilot-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

export function useTripCopilot(tripId: string, clientContext: CopilotClientContext = {}) {
  const t = useTranslations("copilot");
  const [conversation, setConversation] = useState<CopilotConversation>({ id: null, messages: [] });
  const mutation = useMutation({
    mutationFn: (message: string) =>
      sendTripCopilotMessage(tripId, {
        conversationId: conversation.id ?? undefined,
        message,
        clientContext
      })
  });

  const sendMessage = useCallback(
    async (value: string) => {
      const message = value.trim();
      if (!message || mutation.isPending) {
        return;
      }
      const userMessage: CopilotMessage = { id: messageId(), role: "user", content: message };
      setConversation((current) => ({ ...current, messages: [...current.messages, userMessage] }));
      try {
        const response = await mutation.mutateAsync(message);
        setConversation((current) => ({
          id: response.conversationId,
          messages: [
            ...current.messages,
            {
              id: response.messageId,
              role: "assistant",
              content: response.answer,
              response
            }
          ]
        }));
      } catch {
        // The mutation error drives the panel's recoverable error state.
      }
    },
    [mutation, tripId]
  );

  const prompts = useMemo<CopilotSuggestedPrompt[]>(
    () => [
      { id: "next", label: t("promptNext") },
      { id: "health", label: t("promptHealth") },
      { id: "budget", label: t("promptBudget") },
      { id: "group", label: t("promptGroup") },
      { id: "route", label: t("promptRoute") },
      { id: "pack", label: t("promptPack") },
      { id: "share", label: t("promptShare") },
      { id: "receipt", label: t("promptReceipt") },
      { id: "approval", label: t("promptApproval") },
      { id: "changes", label: t("promptChanges") }
    ],
    [t]
  );

  return {
    clearConversation: () => {
      mutation.reset();
      setConversation({ id: null, messages: [] });
    },
    conversation,
    error: mutation.error,
    isLoading: mutation.isPending,
    retry: () => mutation.reset(),
    sendMessage,
    suggestedPrompts: prompts
  };
}
