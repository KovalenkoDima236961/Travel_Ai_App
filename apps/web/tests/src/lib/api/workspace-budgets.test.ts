import { afterEach, describe, expect, it, vi } from "vitest";
import {
  archiveWorkspaceBudget,
  createWorkspaceBudget,
  getWorkspaceBudgetSummary,
  listWorkspaceBudgets,
  makeWorkspaceBudgetPrimary,
  updateWorkspaceBudget
} from "@/lib/api/workspace-budgets";

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

describe("workspace budgets API", () => {
  it("lists budgets with an optional status", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ budgets: [] }, { ok: true, status: 200 })
    );
    vi.stubGlobal("fetch", fetchMock);

    await listWorkspaceBudgets("workspace-1", "active");

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/workspaces/workspace-1/budgets?status=active"),
      expect.any(Object)
    );
  });

  it("creates and normalizes payload fields", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ budget: { id: "budget-1" } }, { ok: true, status: 201 })
    );
    vi.stubGlobal("fetch", fetchMock);

    await createWorkspaceBudget("workspace-1", {
      name: " Japan budget ",
      description: " Shared ",
      amount: 5000,
      currency: " eur ",
      periodStart: "2026-09-01",
      periodEnd: "2026-09-30",
      isPrimary: true
    });

    const [, init] = fetchMock.mock.calls[0];
    expect(init?.method).toBe("POST");
    expect(JSON.parse(init?.body as string)).toMatchObject({
      name: "Japan budget",
      description: "Shared",
      amount: 5000,
      currency: "EUR",
      isPrimary: true
    });
  });

  it("updates, archives, makes primary, and fetches summary endpoints", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ budget: { id: "budget-1" } }, { ok: true, status: 200 })
    );
    vi.stubGlobal("fetch", fetchMock);

    await updateWorkspaceBudget("workspace-1", "budget-1", { amount: 4500 });
    await archiveWorkspaceBudget("workspace-1", "budget-1", "Replaced");
    await makeWorkspaceBudgetPrimary("workspace-1", "budget-1");
    await getWorkspaceBudgetSummary("workspace-1", "budget-1");

    expect(String(fetchMock.mock.calls[0][0])).toContain(
      "/workspaces/workspace-1/budgets/budget-1"
    );
    expect(fetchMock.mock.calls[0][1]?.method).toBe("PATCH");
    expect(String(fetchMock.mock.calls[1][0])).toContain(
      "/workspaces/workspace-1/budgets/budget-1/archive"
    );
    expect(String(fetchMock.mock.calls[2][0])).toContain(
      "/workspaces/workspace-1/budgets/budget-1/make-primary"
    );
    expect(String(fetchMock.mock.calls[3][0])).toContain(
      "/workspaces/workspace-1/budgets/budget-1/summary"
    );
  });
});
