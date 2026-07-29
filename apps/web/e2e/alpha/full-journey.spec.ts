import { expect, test } from "../fixtures/test";
import {
  cloneAndEditFirstItem,
  createGenerationJobViaAPI,
  createPublicShareViaAPI,
  createTripViaAPI,
  getBudgetSummaryViaAPI,
  getPublicTripViaAPI,
  getTripViaAPI,
  getUnreadNotificationCountViaAPI,
  listItineraryVersionsViaAPI,
  markAllNotificationsReadViaAPI,
  updateItineraryViaAPI,
  waitForGenerationJob
} from "../utils/api";

test.describe("alpha-full-journey", () => {
  test("registers, generates, reviews, edits, shares, and reads notifications", async ({ auth, page, request }) => {
    const trip = await createTripViaAPI(request, auth.accessToken, "Alpha Vienna");
    const job = await createGenerationJobViaAPI(request, auth.accessToken, trip.id, {
      jobType: "full_generation",
      expectedItineraryRevision: trip.itineraryRevision
    });
    await waitForGenerationJob(request, auth.accessToken, trip.id, job.id, "completed");

    const generatedTrip = await getTripViaAPI(request, auth.accessToken, trip.id);
    expect(generatedTrip.itinerary?.days.length ?? 0).toBeGreaterThan(0);
    expect(generatedTrip.itinerary?.days[0].items.length ?? 0).toBeGreaterThan(0);

    await page.goto(`/trips/${trip.id}`);
    await expect(page.getByText("Alpha Vienna", { exact: true }).first()).toBeVisible();
    await expect(page.getByText(/generation|quality|review/i).first()).toBeVisible();

    const edited = cloneAndEditFirstItem(generatedTrip.itinerary!, "Alpha edited activity");
    const updated = await updateItineraryViaAPI(
      request,
      auth.accessToken,
      trip.id,
      edited,
      generatedTrip.itineraryRevision
    );
    expect(updated.itineraryRevision).toBe(generatedTrip.itineraryRevision + 1);

    const versions = await listItineraryVersionsViaAPI(request, auth.accessToken, trip.id);
    expect(versions.items.length).toBeGreaterThanOrEqual(2);

    const budget = await getBudgetSummaryViaAPI(request, auth.accessToken, trip.id);
    expect(budget.currency).toBe("EUR");
    expect(budget.estimatedTotal).toBeGreaterThanOrEqual(0);

    const share = await createPublicShareViaAPI(request, auth.accessToken, trip.id);
    expect(share.enabled).toBeTruthy();
    const publicTrip = await getPublicTripViaAPI(request, share.shareToken);
    const publicBodyText = await publicTrip.text();
    expect(publicTrip.ok(), publicBodyText).toBeTruthy();
    expect(publicBodyText).toContain("Alpha Vienna");
    expect(publicBodyText).not.toMatch(/budgetAmount|budgetCurrency|collaborators|providerPlaceId|traceId|accessToken/i);

    const unreadBefore = await getUnreadNotificationCountViaAPI(request, auth.accessToken);
    expect(unreadBefore).toBeGreaterThanOrEqual(0);
    await markAllNotificationsReadViaAPI(request, auth.accessToken);
    await expect.poll(() => getUnreadNotificationCountViaAPI(request, auth.accessToken)).toBe(0);
  });
});
