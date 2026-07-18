import { expect, test } from "@playwright/test";
import { registerOrLogin, TEST_PASSWORD, testEmail } from "./utils/api";

test.describe("critical authentication", () => {
  test("registers and logs out through the UI", async ({ page }, testInfo) => {
    const email = testEmail(`ui-register-${testInfo.retry}`, testInfo.workerIndex);
    await page.goto("/register");
    await page.getByLabel("Email").fill(email);
    await page.getByLabel("Password", { exact: true }).fill(TEST_PASSWORD);
    await page.getByLabel("Confirm password").fill(TEST_PASSWORD);
    await page.getByRole("button", { name: "Create account" }).click();

    await expect(page).toHaveURL(/\/trips(?:\?|$)/);
    await page.getByRole("button", { name: "Account" }).click();
    await page.getByRole("menuitem", { name: "Log out" }).click();
    await expect(page).toHaveURL(/\/login(?:\?next=%2Ftrips)?$/);
  });

  test("logs in an API-created user through the UI", async ({ page, request }, testInfo) => {
    const credentials = {
      email: testEmail(`ui-login-${testInfo.retry}`, testInfo.workerIndex),
      password: TEST_PASSWORD
    };
    await registerOrLogin(request, credentials);
    await page.goto("/login");
    await page.getByLabel("Email").fill(credentials.email);
    await page.getByLabel("Password").fill(credentials.password);
    await page.getByRole("button", { name: "Log in" }).click();

    await expect(page).toHaveURL(/\/trips(?:\?|$)/);
    await expect(page.getByRole("heading", { name: "Your trips" })).toBeVisible();
  });

  test("shows invalid-login feedback and redirects protected routes", async ({ page }) => {
    await page.goto("/trips");
    await expect(page).toHaveURL(/\/login\?next=%2Ftrips$/);

    await page.getByLabel("Email").fill("missing@example.test");
    await page.getByLabel("Password").fill("WrongPassword1");
    await page.getByRole("button", { name: "Log in" }).click();
    await expect(
      page.getByRole("alert").filter({ hasText: "We could not log you in." }),
    ).toBeVisible();
  });
});
