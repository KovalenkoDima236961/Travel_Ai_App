import { afterEach, describe, expect, it, vi } from "vitest";
import {
  cancelGenerationJob,
  createGenerationJob,
  getGenerationJob,
  listGenerationJobs
} from "@/lib/api/generation-jobs";
import type { GenerationJob } from "@/types/generation-jobs";

const job: GenerationJob = {
  id: "job-1",
  tripId: "trip-1",
  requestedByUserId: "user-1",
  jobType: "full_generation",
  status: "queued",
  expectedItineraryRevision: 3,
  createdAt: "2026-06-25T00:00:00Z",
  updatedAt: "2026-06-25T00:00:00Z"
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

describe("generation job API", () => {
  it("creates a generation job with expectedItineraryRevision", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ job }, { ok: true, status: 202 }));
    vi.stubGlobal("fetch", fetchMock);

    const result = await createGenerationJob("trip-1", {
      jobType: "day_regeneration",
      expectedItineraryRevision: 3,
      instruction: " less walking ",
      dayNumber: 2
    });

    expect(result).toEqual(job);
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/trips/trip-1/generation-jobs"),
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          jobType: "day_regeneration",
          expectedItineraryRevision: 3,
          instruction: "less walking",
          dayNumber: 2
        })
      })
    );
  });

  it("unwraps get/list/cancel responses", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ job }, { ok: true, status: 200 }))
      .mockResolvedValueOnce(jsonResponse({ items: [job], limit: 20 }, { ok: true, status: 200 }))
      .mockResolvedValueOnce(jsonResponse({ job }, { ok: true, status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(getGenerationJob("trip-1", "job-1")).resolves.toEqual(job);
    await expect(listGenerationJobs("trip-1")).resolves.toEqual([job]);
    await expect(cancelGenerationJob("trip-1", "job-1")).resolves.toEqual(job);
  });
});
