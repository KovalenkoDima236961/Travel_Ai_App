import { expect, test } from "../fixtures/test";

test.describe("alpha-auth", () => {
  test("authenticated session opens trips and settings", async ({ page }) => {
    await page.goto("/trips");
    await expect(page.getByRole("heading", { name: "Your trips" })).toBeVisible();

    await page.goto("/settings");
    await expect(page.getByRole("heading", { name: /settings/i })).toBeVisible();
    await expect(page.locator("body")).not.toContainText("stack trace");
  });
});
