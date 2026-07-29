import { expect, test } from "../fixtures/test";
import {
  createProtectedPublicShareViaAPI,
  createPublicShareViaAPI,
  createTripViaAPI,
  disablePublicShareViaAPI,
  getPublicShareStatusViaAPI,
  getPublicTripViaAPI,
  unlockPublicShareViaAPI
} from "../utils/api";

test.describe("alpha-public-share", () => {
  test("disables share and blocks the old token", async ({ auth, request }) => {
    const trip = await createTripViaAPI(request, auth.accessToken, "Alpha Share Disabled");
    const share = await createPublicShareViaAPI(request, auth.accessToken, trip.id);
    await disablePublicShareViaAPI(request, auth.accessToken, trip.id);

    const status = await getPublicShareStatusViaAPI(request, share.shareToken);
    expect([404, 410]).toContain(status.status());
    const publicTrip = await getPublicTripViaAPI(request, share.shareToken);
    expect([404, 410]).toContain(publicTrip.status());
  });

  test("requires password for protected public share", async ({ auth, request }) => {
    const trip = await createTripViaAPI(request, auth.accessToken, "Alpha Protected Share");
    const share = await createProtectedPublicShareViaAPI(request, auth.accessToken, trip.id);
    expect(share.passwordRequired).toBeTruthy();

    const locked = await getPublicTripViaAPI(request, share.shareToken);
    expect(locked.status()).toBe(401);
    const wrong = await unlockPublicShareViaAPI(request, share.shareToken, "wrong-password");
    expect([401, 429]).toContain(wrong.status());

    const unlocked = await unlockPublicShareViaAPI(request, share.shareToken, "alpha-share-pass");
    expect(unlocked.ok(), await unlocked.text()).toBeTruthy();
    const accessToken = ((await unlocked.json()) as { accessToken: string }).accessToken;
    const publicTrip = await getPublicTripViaAPI(request, share.shareToken, accessToken);
    expect(publicTrip.ok(), await publicTrip.text()).toBeTruthy();
  });
});
