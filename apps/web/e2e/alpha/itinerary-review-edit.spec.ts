import { expect, test } from "../fixtures/test";
import {
  cloneAndEditFirstItem,
  createGenerationJobViaAPI,
  createTripViaAPI,
  expectItineraryConflictViaAPI,
  getTripViaAPI,
  updateItineraryViaAPI,
  waitForGenerationJob
} from "../utils/api";

test.describe("alpha-itinerary-review-edit", () => {
  test("prevents stale revision overwrite", async ({ auth, request }) => {
    const trip = await createTripViaAPI(request, auth.accessToken, "Alpha Conflict");
    const job = await createGenerationJobViaAPI(request, auth.accessToken, trip.id, {
      expectedItineraryRevision: trip.itineraryRevision
    });
    await waitForGenerationJob(request, auth.accessToken, trip.id, job.id);

    const generatedTrip = await getTripViaAPI(request, auth.accessToken, trip.id);
    const firstEdit = cloneAndEditFirstItem(generatedTrip.itinerary!, "First alpha edit");
    await updateItineraryViaAPI(request, auth.accessToken, trip.id, firstEdit, generatedTrip.itineraryRevision);

    const staleEdit = cloneAndEditFirstItem(generatedTrip.itinerary!, "Stale alpha edit");
    await expectItineraryConflictViaAPI(request, auth.accessToken, trip.id, staleEdit, generatedTrip.itineraryRevision);

    const afterConflict = await getTripViaAPI(request, auth.accessToken, trip.id);
    expect(afterConflict.itineraryRevision).toBe(generatedTrip.itineraryRevision + 1);
  });
});
