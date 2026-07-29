import { expect, test } from "../fixtures/test";
import {
  createGenerationJobViaAPI,
  createTripViaAPI,
  getGenerationJobViaAPI,
  getTripViaAPI,
  waitForGenerationJob
} from "../utils/api";

test.describe("alpha-failure-recovery", () => {
  test("does not create a generation job with a stale itinerary revision", async ({ auth, request }) => {
    const trip = await createTripViaAPI(request, auth.accessToken, "Alpha Stale Job");
    const responseTrip = await getTripViaAPI(request, auth.accessToken, trip.id);
    const tripURL = process.env.E2E_TRIP_URL ?? "http://127.0.0.1:8080";
    const response = await request.post(`${tripURL}/trips/${trip.id}/generation-jobs`, {
      headers: { Authorization: `Bearer ${auth.accessToken}` },
      data: { jobType: "full_generation", expectedItineraryRevision: responseTrip.itineraryRevision + 10 }
    });
    expect([400, 409, 422]).toContain(response.status());
  });

  test("optional transient provider fixture recovers without corrupting the trip", async ({ auth, request }) => {
    test.skip(!process.env.ALPHA_TRANSIENT_PROVIDER_INSTRUCTION, "Set ALPHA_TRANSIENT_PROVIDER_INSTRUCTION to run this destructive fixture.");
    const trip = await createTripViaAPI(request, auth.accessToken, "Alpha Transient Fixture");
    const job = await createGenerationJobViaAPI(request, auth.accessToken, trip.id, {
      expectedItineraryRevision: trip.itineraryRevision,
      instruction: process.env.ALPHA_TRANSIENT_PROVIDER_INSTRUCTION
    });
    await waitForGenerationJob(request, auth.accessToken, trip.id, job.id, "completed");
    const recovered = await getTripViaAPI(request, auth.accessToken, trip.id);
    expect(recovered.itinerary).toBeTruthy();
  });

  test("optional permanent provider fixture fails cleanly", async ({ auth, request }) => {
    test.skip(!process.env.ALPHA_PERMANENT_PROVIDER_INSTRUCTION, "Set ALPHA_PERMANENT_PROVIDER_INSTRUCTION to run this destructive fixture.");
    const trip = await createTripViaAPI(request, auth.accessToken, "Alpha Permanent Fixture");
    const job = await createGenerationJobViaAPI(request, auth.accessToken, trip.id, {
      expectedItineraryRevision: trip.itineraryRevision,
      instruction: process.env.ALPHA_PERMANENT_PROVIDER_INSTRUCTION
    });
    const failed = await waitForGenerationJob(request, auth.accessToken, trip.id, job.id, "failed");
    expect(failed.errorCode ?? failed.errorMessageSafe ?? "").not.toContain("OPENAI_API_KEY");
    const clean = await getGenerationJobViaAPI(request, auth.accessToken, trip.id, job.id);
    expect(clean.status).toBe("failed");
  });
});
