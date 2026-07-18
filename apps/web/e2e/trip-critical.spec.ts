import { test, expect } from "./fixtures/test";

test("creates a trip, starts deterministic generation, and opens it from the list", async ({ page }) => {
  await page.goto("/trips/new");

  await page.getByPlaceholder("City, region, or country").fill("Vienna");
  await page.locator("#startDate").fill("2027-04-10");
  await page.getByRole("button", { name: "Continue" }).click();
  await expect(page.getByRole("heading", { name: "Who is going?" })).toBeVisible();

  await page.getByRole("button", { name: "Continue" }).click();
  await expect(page.getByRole("heading", { name: "Budget and style" })).toBeVisible();
  await page.locator("#budgetAmount").fill("600");
  await page.getByRole("button", { name: "Food" }).click();
  await page.getByRole("button", { name: "Continue" }).click();
  await expect(page.getByRole("heading", { name: "Route and transport" })).toBeVisible();

  await page.getByRole("button", { name: "Continue" }).click();
  await expect(page.getByText("Vienna", { exact: true }).first()).toBeVisible();
  await page.getByRole("button", { name: "Create trip and generate itinerary" }).click();

  await expect(page).toHaveURL(/\/trips\/[0-9a-f-]+/);
  await expect(page.getByText("Vienna", { exact: true }).first()).toBeVisible();

  await page.goto("/trips");
  await expect(page.getByText("Vienna", { exact: true }).first()).toBeVisible();
  await page.getByText("Vienna", { exact: true }).first().click();
  await expect(page).toHaveURL(/\/trips\/[0-9a-f-]+/);
});
