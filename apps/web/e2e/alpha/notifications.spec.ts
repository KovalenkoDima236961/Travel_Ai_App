import { expect, test } from "../fixtures/test";
import {
  createGenerationJobViaAPI,
  createTripViaAPI,
  getUnreadNotificationCountViaAPI,
  markAllNotificationsReadViaAPI,
  waitForGenerationJob
} from "../utils/api";

test.describe("alpha-notifications", () => {
  test("notification unread count is available after generation and can be cleared", async ({ auth, request }) => {
    const trip = await createTripViaAPI(request, auth.accessToken, "Alpha Notifications");
    const job = await createGenerationJobViaAPI(request, auth.accessToken, trip.id, {
      expectedItineraryRevision: trip.itineraryRevision
    });
    await waitForGenerationJob(request, auth.accessToken, trip.id, job.id);

    const unread = await getUnreadNotificationCountViaAPI(request, auth.accessToken);
    expect(unread).toBeGreaterThanOrEqual(0);
    await markAllNotificationsReadViaAPI(request, auth.accessToken);
    await expect.poll(() => getUnreadNotificationCountViaAPI(request, auth.accessToken)).toBe(0);
  });
});
