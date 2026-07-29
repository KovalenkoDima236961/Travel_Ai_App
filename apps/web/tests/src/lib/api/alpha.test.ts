import { afterEach, describe, expect, it, vi } from "vitest";
import {
  createAlphaInvite,
  inviteFromWaitlist,
  joinAlphaWaitlist,
  recordAnalyticsEvent
} from "@/lib/api/alpha";

function jsonResponse(body: unknown, init: { ok: boolean; status: number }): Response {
  return {
    ok: init.ok,
    status: init.status,
    text: async () => JSON.stringify(body),
    json: async () => body
  } as unknown as Response;
}

afterEach(() => {
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
});

describe("alpha API client", () => {
  it("submits public waitlist registration without auth-only fields", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ waitlist: { id: "wait-1" } }, { ok: true, status: 201 }));
    vi.stubGlobal("fetch", fetchMock);

    await joinAlphaWaitlist("tester@example.com");

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/alpha/waitlist"),
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ email: "tester@example.com", source: "web" })
      })
    );
  });

  it("normalizes date-only invite expirations before posting", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ invite: { id: "invite-1" } }, { ok: true, status: 201 }))
      .mockResolvedValueOnce(
        jsonResponse({ invite: { id: "invite-2" }, waitlist: { id: "wait-1" } }, { ok: true, status: 201 })
      );
    vi.stubGlobal("fetch", fetchMock);

    await createAlphaInvite({
      testerGroup: "external",
      maxActivations: 2,
      expiresAt: "2026-08-05",
      notes: "Batch 1"
    });
    await inviteFromWaitlist({
      waitlistId: "wait-1",
      testerGroup: "qa",
      expiresAt: "2026-08-06"
    });

    const bodies = fetchMock.mock.calls.map(([, init]) => JSON.parse(init?.body as string));
    expect(bodies[0]).toMatchObject({
      testerGroup: "external",
      maxActivations: 2,
      expiresAt: "2026-08-05T23:59:59.000Z"
    });
    expect(bodies[1]).toMatchObject({
      waitlistId: "wait-1",
      testerGroup: "qa",
      expiresAt: "2026-08-06T23:59:59.000Z"
    });
  });

  it("adds alpha event source, version, and session metadata", async () => {
    vi.stubEnv("NEXT_PUBLIC_APP_VERSION", "alpha-test");
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ event: { id: "event-1" } }, { ok: true, status: 201 }));
    vi.stubGlobal("fetch", fetchMock);

    await recordAnalyticsEvent({
      eventName: "trip_created",
      feature: "trips",
      entityType: "trip",
      entityId: "trip-1",
      metadata: { days: 3 }
    });

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/analytics/events"),
      expect.objectContaining({ method: "POST" })
    );
    const [, init] = fetchMock.mock.calls[0];
    expect(JSON.parse(init?.body as string)).toMatchObject({
      eventName: "trip_created",
      feature: "trips",
      entityType: "trip",
      entityId: "trip-1",
      metadata: { days: 3 },
      sessionId: "server",
      source: "web",
      appVersion: "alpha-test"
    });
  });
});
