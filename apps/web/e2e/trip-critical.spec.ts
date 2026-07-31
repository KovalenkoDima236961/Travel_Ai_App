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

  const tripId = new URL(page.url()).pathname.split("/")[2];
  const workspaceNavigation = page.getByRole("navigation", { name: "Trip workspace" });
  await expect(workspaceNavigation).toBeVisible();
  await workspaceNavigation.getByRole("link", { name: "Plan", exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`/trips/${tripId}/plan`));
  await expect(workspaceNavigation.getByRole("link", { name: "Plan", exact: true })).toHaveAttribute(
    "aria-current",
    "page"
  );

  await page.goto(`/trips/${tripId}?tab=expenses`);
  await expect(workspaceNavigation.getByRole("link", { name: "Money", exact: true })).toHaveAttribute(
    "aria-current",
    "page"
  );
  await workspaceNavigation.getByRole("link", { name: "Prepare", exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`/trips/${tripId}/prepare`));
});
