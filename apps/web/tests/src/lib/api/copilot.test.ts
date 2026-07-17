import { describe, expect, it, vi } from "vitest";

const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }));

vi.mock("@/shared/api/client", () => ({ apiFetch }));

import { copilotKeys, sendTripCopilotMessage } from "@/lib/api/copilot";

describe("Trip Copilot API", () => {
  it("posts only the visible message and client context to the private trip endpoint", async () => {
    apiFetch.mockResolvedValue({ conversationId: "conversation-1" });

    await sendTripCopilotMessage("trip-123", {
      conversationId: "conversation-1",
      message: "What should I fix first?",
      clientContext: { currentTab: "overview", selectedDayNumber: 2 }
    });

    expect(apiFetch).toHaveBeenCalledWith("/trips/trip-123/copilot/chat", {
      method: "POST",
      body: JSON.stringify({
        conversationId: "conversation-1",
        message: "What should I fix first?",
        clientContext: { currentTab: "overview", selectedDayNumber: 2 }
      })
    });
  });

  it("uses a trip-scoped cache key", () => {
    expect(copilotKeys.conversation("trip-123")).toEqual(["trip-copilot", "trip-123"]);
  });
});
