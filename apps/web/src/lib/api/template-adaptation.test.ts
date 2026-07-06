import { afterEach, describe, expect, it, vi } from "vitest";
import {
  createTemplateAdaptationJob,
  getTemplateAdaptationJob
} from "@/lib/api/template-adaptation";
import type { GenerationJob } from "@/types/generation-jobs";

const job: GenerationJob = {
  id: "job-1",
  tripId: "trip-1",
  requestedByUserId: "user-1",
  jobType: "template_adaptation",
  status: "queued",
  expectedItineraryRevision: 0,
  createdAt: "2026-07-06T00:00:00Z",
  updatedAt: "2026-07-06T00:00:00Z"
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

describe("template adaptation API", () => {
  it("creates an adaptation job and normalizes the payload", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ job }, { ok: true, status: 202 }));
    vi.stubGlobal("fetch", fetchMock);

    const result = await createTemplateAdaptationJob("template-1", {
      title: "  Vienna weekend  ",
      destination: "  Vienna  ",
      startDate: "2026-09-10",
      durationDays: 3,
      budget: { amount: 700, currency: "eur" },
      travelers: 2,
      pace: "balanced",
      interests: ["museums", " food ", "museums"],
      avoid: ["nightclubs"],
      specialInstructions: "  first timers  ",
      fallbackToDeterministic: true
    });

    expect(result).toEqual(job);
    const [url, options] = fetchMock.mock.calls[0];
    expect(String(url)).toContain("/trip-templates/template-1/adaptation-jobs");
    const body = JSON.parse((options as RequestInit).body as string);
    expect(body).toMatchObject({
      title: "Vienna weekend",
      destination: "Vienna",
      startDate: "2026-09-10",
      durationDays: 3,
      budget: { amount: 700, currency: "EUR" },
      travelers: 2,
      pace: "balanced",
      interests: ["museums", "food"],
      avoid: ["nightclubs"],
      specialInstructions: "first timers",
      fallbackToDeterministic: true
    });
  });

  it("defaults fallbackToDeterministic to true", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ job }, { ok: true, status: 202 }));
    vi.stubGlobal("fetch", fetchMock);

    await createTemplateAdaptationJob("template-1", {
      title: "Vienna",
      destination: "Vienna",
      startDate: "2026-09-10",
      durationDays: 3
    });

    const body = JSON.parse((fetchMock.mock.calls[0][1] as RequestInit).body as string);
    expect(body.fallbackToDeterministic).toBe(true);
  });

  it("reuses the per-trip job status endpoint", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ job }, { ok: true, status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(getTemplateAdaptationJob("trip-1", "job-1")).resolves.toEqual(job);
    expect(String(fetchMock.mock.calls[0][0])).toContain("/trips/trip-1/generation-jobs/job-1");
  });
});
