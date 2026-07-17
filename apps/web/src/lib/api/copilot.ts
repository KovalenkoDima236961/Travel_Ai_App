import { apiFetch } from "@/shared/api/client";
import type { CopilotRequest, CopilotResponse } from "@/types/copilot";

export const copilotKeys = {
  all: ["trip-copilot"] as const,
  conversation: (tripId: string) => [...copilotKeys.all, tripId] as const
};

export function sendTripCopilotMessage(tripId: string, input: CopilotRequest) {
  return apiFetch<CopilotResponse>(`/trips/${tripId}/copilot/chat`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}
