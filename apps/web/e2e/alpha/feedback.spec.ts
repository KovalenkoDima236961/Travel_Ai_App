import { expect, test } from "../fixtures/test";

test.describe("alpha-feedback", () => {
  test("submits structured feedback from an authenticated page", async ({ page }) => {
    await page.goto("/trips");

    await page.getByRole("button", { name: "Feedback" }).click();
    await expect(page.getByRole("heading", { name: "Send feedback" })).toBeVisible();

    await page.locator("label", { hasText: "Category" }).locator("select").selectOption("feature_request");
    await page.locator("label", { hasText: "Title" }).locator("input").fill("Calendar export for alpha");
    await page.locator("label", { hasText: "Details" }).locator("textarea").fill("It would help to export finalized days to my calendar.");
    await page.getByRole("button", { name: "Send" }).click();

    await expect(page.getByText("Feedback sent.")).toBeVisible();
  });
});
