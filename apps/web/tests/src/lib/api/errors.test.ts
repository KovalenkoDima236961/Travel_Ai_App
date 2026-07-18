import { describe, expect, it } from "vitest";
import { AppApiError, normalizeApiErrorPayload } from "@/lib/api/errors";
import { ApiError } from "@/shared/api/client";

describe("API error normalization", () => {
  it("normalizes structured validation errors and preserves field/request data", () => {
    const normalized = normalizeApiErrorPayload(
      {
        error: {
          code: "validation_error",
          message: "Invalid request.",
          details: [{ field: "startDate", message: "Start date is required." }],
          requestId: "request-42"
        }
      },
      422
    );

    expect(normalized).toMatchObject({
      code: "validation_error",
      status: 422,
      requestId: "request-42",
      fieldErrors: { startDate: "Start date is required." }
    });
  });

  it("maps legacy string errors by HTTP status", () => {
    const error = new AppApiError(normalizeApiErrorPayload({ error: "forbidden" }, 403));

    expect(error.code).toBe("forbidden");
    expect(error.status).toBe(403);
    expect(error.message).toBe("forbidden");
  });

  it("keeps the ApiError legacy constructor compatible with offline callers", () => {
    const error = new ApiError("Conflict", 409, undefined, "itinerary_conflict", 8);

    expect(error).toMatchObject({
      code: "itinerary_conflict",
      status: 409,
      currentItineraryRevision: 8
    });
  });
});
