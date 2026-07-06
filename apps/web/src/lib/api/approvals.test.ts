import { afterEach, describe, expect, it, vi } from "vitest";

import {
  approveTrip,
  cancelTripApproval,
  getTripApproval,
  listTripApprovalEvents,
  listWorkspaceApprovals,
  requestTripChanges,
  submitTripApproval
} from "@/lib/api/approvals";

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

describe("approvals API", () => {
  it("gets trip approval from the right endpoint", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ status: "draft" }, { ok: true, status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await getTripApproval("trip-1");

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/trips/trip-1/approval"),
      expect.any(Object)
    );
  });

  it("submits with note and acknowledged warnings", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ status: "pending_approval" }, { ok: true, status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await submitTripApproval("trip-1", { note: " ready ", acknowledgedWarnings: ["availability_checked"] });

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toContain("/trips/trip-1/approval/submit");
    expect(init.method).toBe("POST");
    const body = JSON.parse(init.body as string);
    expect(body.note).toBe("ready");
    expect(body.acknowledgedWarnings).toEqual(["availability_checked"]);
  });

  it("approves and requests changes on their endpoints", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ status: "approved" }, { ok: true, status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await approveTrip("trip-1", { decisionNote: "ok" });
    await requestTripChanges("trip-1", { decisionNote: "fix budget" });

    expect(fetchMock.mock.calls[0][0]).toContain("/trips/trip-1/approval/approve");
    expect(fetchMock.mock.calls[1][0]).toContain("/trips/trip-1/approval/request-changes");
  });

  it("cancels and lists events", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ events: [] }, { ok: true, status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await cancelTripApproval("trip-1", {});
    await listTripApprovalEvents("trip-1");

    expect(fetchMock.mock.calls[0][0]).toContain("/trips/trip-1/approval/cancel");
    expect(fetchMock.mock.calls[1][0]).toContain("/trips/trip-1/approval/events");
  });

  it("lists workspace approvals with a status filter", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ approvals: [], counts: {}, nextCursor: null }, { ok: true, status: 200 })
    );
    vi.stubGlobal("fetch", fetchMock);

    await listWorkspaceApprovals("workspace-1", { status: "pending_approval", limit: 25 });

    const url = fetchMock.mock.calls[0][0] as string;
    expect(url).toContain("/workspaces/workspace-1/approvals");
    expect(url).toContain("status=pending_approval");
    expect(url).toContain("limit=25");
  });
});
