import { test, expect } from "./fixtures/test";
import { createPublicShareViaAPI, createTripViaAPI } from "./utils/api";

test("shows an anonymous sanitized read-only public trip", async ({ auth, browser, request }) => {
  const trip = await createTripViaAPI(request, auth.accessToken, "Public Vienna");
  const share = await createPublicShareViaAPI(request, auth.accessToken, trip.id);
  expect(share.enabled).toBeTruthy();

  const anonymousContext = await browser.newContext();
  const anonymousPage = await anonymousContext.newPage();
  await anonymousPage.goto(`/share/${share.shareToken}`);

  await expect(anonymousPage.getByText("Read-only shared trip")).toBeVisible();
  await expect(anonymousPage.getByText("Public Vienna", { exact: true }).first()).toBeVisible();
  await expect(anonymousPage.getByText("Shared expenses")).toHaveCount(0);
  await expect(anonymousPage.getByText("Comments", { exact: true })).toHaveCount(0);
  await expect(anonymousPage.getByRole("button", { name: /edit/i })).toHaveCount(0);
  await anonymousContext.close();
});
