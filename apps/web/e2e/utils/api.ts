import { expect, type APIRequestContext } from "@playwright/test";

export type TestCredentials = { email: string; password: string };
export type AuthResponse = {
  user: { id: string; email: string; createdAt: string };
  accessToken: string;
  refreshToken: string;
};

export const TEST_PASSWORD = "TestPassword1";

export function testEmail(scope: string, workerIndex: number) {
  const runId = (process.env.E2E_RUN_ID ?? process.env.GITHUB_RUN_ID ?? "local")
    .toLowerCase()
    .replace(/[^a-z0-9-]/g, "-");
  return `${scope}-${runId}-${workerIndex}@example.test`;
}

export async function registerOrLogin(
  request: APIRequestContext,
  credentials: TestCredentials
): Promise<AuthResponse> {
  const authURL = process.env.E2E_AUTH_URL ?? "http://127.0.0.1:8082";
  const registerResponse = await request.post(`${authURL}/auth/register`, { data: credentials });

  if (registerResponse.ok()) {
    return registerResponse.json();
  }
  if (registerResponse.status() !== 409) {
    throw new Error(`Could not register E2E user: HTTP ${registerResponse.status()} ${await registerResponse.text()}`);
  }

  const loginResponse = await request.post(`${authURL}/auth/login`, { data: credentials });
  expect(loginResponse.ok(), await loginResponse.text()).toBeTruthy();
  return loginResponse.json();
}

export async function createTripViaAPI(
  request: APIRequestContext,
  accessToken: string,
  destination = "Vienna"
) {
  const tripURL = process.env.E2E_TRIP_URL ?? "http://127.0.0.1:8080";
  const response = await request.post(`${tripURL}/trips`, {
    headers: { Authorization: `Bearer ${accessToken}` },
    data: {
      destination,
      tripType: "single_destination",
      startDate: "2027-04-10",
      days: 2,
      budgetAmount: 600,
      budgetCurrency: "EUR",
      travelers: 2,
      interests: ["food", "culture"],
      pace: "balanced"
    }
  });
  expect(response.ok(), await response.text()).toBeTruthy();
  return response.json() as Promise<{ id: string; destination: string; itineraryRevision: number }>;
}

export async function createPublicShareViaAPI(
  request: APIRequestContext,
  accessToken: string,
  tripId: string
) {
  const tripURL = process.env.E2E_TRIP_URL ?? "http://127.0.0.1:8080";
  const response = await request.post(`${tripURL}/trips/${tripId}/share`, {
    headers: { Authorization: `Bearer ${accessToken}` }
  });
  expect(response.ok(), await response.text()).toBeTruthy();
  return response.json() as Promise<{ shareToken: string; shareUrl: string; enabled: boolean }>;
}
