import { test as base } from "@playwright/test";
import type { AuthResponse, TestCredentials } from "../utils/api";
import { registerOrLogin, TEST_PASSWORD, testEmail } from "../utils/api";
import { installAuthState } from "../utils/auth";

type AppFixtures = {
  auth: AuthResponse;
  credentials: TestCredentials;
};

export const test = base.extend<AppFixtures>({
  credentials: async ({}, use, testInfo) => {
    await use({
      email: testEmail(`owner-${testInfo.parallelIndex}`, testInfo.workerIndex),
      password: TEST_PASSWORD
    });
  },
  auth: async ({ credentials, request }, use) => {
    await use(await registerOrLogin(request, credentials));
  },
  page: async ({ auth, page }, use) => {
    await installAuthState(page, auth);
    await use(page);
  }
});

export { expect } from "@playwright/test";
