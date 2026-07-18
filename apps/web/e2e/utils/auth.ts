import type { Page } from "@playwright/test";
import type { AuthResponse } from "./api";

export async function installAuthState(page: Page, auth: AuthResponse) {
  await page.addInitScript(
    ({ accessToken, refreshToken }) => {
      window.localStorage.setItem("travel_ai_access_token", accessToken);
      window.localStorage.setItem("travel_ai_refresh_token", refreshToken);
    },
    { accessToken: auth.accessToken, refreshToken: auth.refreshToken }
  );
}
