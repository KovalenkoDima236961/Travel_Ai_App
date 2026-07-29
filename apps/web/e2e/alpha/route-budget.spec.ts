import { expect, test } from "../fixtures/test";
import {
  createGenerationJobViaAPI,
  createTripViaAPI,
  getBudgetSummaryViaAPI,
  getTripViaAPI,
  waitForGenerationJob
} from "../utils/api";

test.describe("alpha-route-budget", () => {
  test("keeps route basics and budget summary readable after generation", async ({ auth, page, request }) => {
    const trip = await createTripViaAPI(request, auth.accessToken, "Alpha Budget");
    const job = await createGenerationJobViaAPI(request, auth.accessToken, trip.id, {
      expectedItineraryRevision: trip.itineraryRevision
    });
    await waitForGenerationJob(request, auth.accessToken, trip.id, job.id);

    const generatedTrip = await getTripViaAPI(request, auth.accessToken, trip.id);
    expect(generatedTrip.itinerary).toBeTruthy();

    const summary = await getBudgetSummaryViaAPI(request, auth.accessToken, trip.id);
    expect(summary.estimatedItemCount).toBeGreaterThanOrEqual(0);

    await page.goto(`/trips/${trip.id}`);
    await expect(page.getByText(/budget/i).first()).toBeVisible();
    await expect(page.getByText(/route|distance|transport/i).first()).toBeVisible();
  });
});
