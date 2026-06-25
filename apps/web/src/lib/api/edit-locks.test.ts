import { afterEach, describe, expect, it, vi } from "vitest";

import {
  acquireTripEditLock,
  getTripEditLock,
  releaseTripEditLock
} from "@/lib/api/edit-locks";

const lock = {
  locked: true,
  scope: "itinerary",
  tripId: "trip-1",
  lockedByUserId: "user-1",
  lockedByDisplayName: "Anna",
  lockedByRole: "editor",
  lockedByCurrentUser: false,
  createdAt: "2026-06-25T12:00:00Z",
  expiresAt: "2026-06-25T12:03:00Z",
  ttlSeconds: 180
};

function jsonResponse(body: unknown, init: { ok: boolean; status: number }): Response {
  return {
    ok: init.ok,
    status: init.status,
    text: async () => JSON.stringify(body),
    json: async () => body
  } as unknown as Response;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("edit lock API", () => {
  it("gets the current trip edit lock", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(lock, { ok: true, status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    const result = await getTripEditLock("trip-1");

    expect(result).toEqual(lock);
    expect(fetchMock.mock.calls[0]?.[0]).toBe("http://localhost:8080/trips/trip-1/edit-lock");
    expect(fetchMock.mock.calls[0]?.[1]?.method).toBeUndefined();
  });

  it("parses acquire success", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ acquired: true, lock: { ...lock, lockedByCurrentUser: true } }, { ok: true, status: 200 })
    );
    vi.stubGlobal("fetch", fetchMock);

    const result = await acquireTripEditLock("trip-1");

    expect(result.acquired).toBe(true);
    expect(result.lock?.lockedByCurrentUser).toBe(true);
    expect(fetchMock.mock.calls[0]?.[1]?.method).toBe("POST");
    expect(JSON.parse(fetchMock.mock.calls[0]?.[1]?.body as string)).toEqual({
      scope: "itinerary"
    });
  });

  it("returns 409 lock conflicts as typed acquire responses", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse(
        {
          error: "edit_lock_conflict",
          message: "Another user is already editing this itinerary.",
          acquired: false,
          reason: "locked_by_other_user",
          lock
        },
        { ok: false, status: 409 }
      )
    );
    vi.stubGlobal("fetch", fetchMock);

    const result = await acquireTripEditLock("trip-1");

    expect(result).toEqual({
      acquired: false,
      reason: "locked_by_other_user",
      lock
    });
  });

  it("releases the current user's lock", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ released: true }, { ok: true, status: 200 })
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(releaseTripEditLock("trip-1")).resolves.toEqual({ released: true });
    expect(fetchMock.mock.calls[0]?.[1]?.method).toBe("DELETE");
    expect(JSON.parse(fetchMock.mock.calls[0]?.[1]?.body as string)).toEqual({
      scope: "itinerary"
    });
  });
});
