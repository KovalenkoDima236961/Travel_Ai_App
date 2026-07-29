import { expect, type APIRequestContext } from "@playwright/test";

export type TestCredentials = { email: string; password: string };
export type AuthResponse = {
  user: { id: string; email: string; createdAt: string };
  accessToken: string;
  refreshToken: string;
};
export type TestItinerary = {
  days: Array<{
    day: number;
    title: string;
    items: Array<Record<string, unknown> & { name: string; time?: string; type?: string }>;
  }>;
};
export type TestTrip = {
  id: string;
  destination: string;
  itineraryRevision: number;
  itinerary?: TestItinerary | null;
  workspaceId?: string | null;
};
export type GenerationJob = {
  id: string;
  tripId: string;
  jobType?: string;
  status: "queued" | "running" | "completed" | "failed" | "cancelled";
  expectedItineraryRevision?: number;
  resultItineraryRevision?: number | null;
  errorCode?: string | null;
  errorMessageSafe?: string | null;
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
  return unwrapTrip(await response.json());
}

export async function getTripViaAPI(
  request: APIRequestContext,
  accessToken: string,
  tripId: string
): Promise<TestTrip> {
  const tripURL = process.env.E2E_TRIP_URL ?? "http://127.0.0.1:8080";
  const response = await request.get(`${tripURL}/trips/${tripId}`, {
    headers: { Authorization: `Bearer ${accessToken}` }
  });
  expect(response.ok(), await response.text()).toBeTruthy();
  return unwrapTrip(await response.json());
}

export async function createGenerationJobViaAPI(
  request: APIRequestContext,
  accessToken: string,
  tripId: string,
  input: {
    jobType?: "full_generation" | "day_regeneration" | "item_regeneration";
    expectedItineraryRevision: number;
    instruction?: string;
    dayNumber?: number;
    itemIndex?: number;
  }
): Promise<GenerationJob> {
  const tripURL = process.env.E2E_TRIP_URL ?? "http://127.0.0.1:8080";
  const response = await request.post(`${tripURL}/trips/${tripId}/generation-jobs`, {
    headers: { Authorization: `Bearer ${accessToken}` },
    data: {
      jobType: input.jobType ?? "full_generation",
      expectedItineraryRevision: input.expectedItineraryRevision,
      ...(input.instruction ? { instruction: input.instruction } : {}),
      ...(input.dayNumber != null ? { dayNumber: input.dayNumber } : {}),
      ...(input.itemIndex != null ? { itemIndex: input.itemIndex } : {})
    }
  });
  expect(response.ok(), await response.text()).toBeTruthy();
  return (await response.json()).job as GenerationJob;
}

export async function getGenerationJobViaAPI(
  request: APIRequestContext,
  accessToken: string,
  tripId: string,
  jobId: string
): Promise<GenerationJob> {
  const tripURL = process.env.E2E_TRIP_URL ?? "http://127.0.0.1:8080";
  const response = await request.get(`${tripURL}/trips/${tripId}/generation-jobs/${jobId}`, {
    headers: { Authorization: `Bearer ${accessToken}` }
  });
  expect(response.ok(), await response.text()).toBeTruthy();
  return (await response.json()).job as GenerationJob;
}

export async function waitForGenerationJob(
  request: APIRequestContext,
  accessToken: string,
  tripId: string,
  jobId: string,
  expectedStatus: GenerationJob["status"] = "completed"
): Promise<GenerationJob> {
  const attempts = Number(process.env.ALPHA_GENERATION_POLL_ATTEMPTS ?? 90);
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    const job = await getGenerationJobViaAPI(request, accessToken, tripId, jobId);
    if (job.status === expectedStatus) return job;
    if (!["queued", "running"].includes(job.status)) {
      throw new Error(`Generation job ${jobId} finished with ${job.status}; expected ${expectedStatus}. ${job.errorCode ?? ""}`);
    }
    await new Promise((resolve) => setTimeout(resolve, 2_000));
  }
  throw new Error(`Generation job ${jobId} did not reach ${expectedStatus}.`);
}

export async function updateItineraryViaAPI(
  request: APIRequestContext,
  accessToken: string,
  tripId: string,
  itinerary: TestItinerary,
  expectedItineraryRevision: number
): Promise<TestTrip> {
  const tripURL = process.env.E2E_TRIP_URL ?? "http://127.0.0.1:8080";
  const response = await request.put(`${tripURL}/trips/${tripId}/itinerary`, {
    headers: { Authorization: `Bearer ${accessToken}` },
    data: { itinerary, expectedItineraryRevision }
  });
  expect(response.ok(), await response.text()).toBeTruthy();
  return unwrapTrip(await response.json());
}

export async function expectItineraryConflictViaAPI(
  request: APIRequestContext,
  accessToken: string,
  tripId: string,
  itinerary: TestItinerary,
  staleRevision: number
) {
  const tripURL = process.env.E2E_TRIP_URL ?? "http://127.0.0.1:8080";
  const response = await request.put(`${tripURL}/trips/${tripId}/itinerary`, {
    headers: { Authorization: `Bearer ${accessToken}` },
    data: { itinerary, expectedItineraryRevision: staleRevision }
  });
  expect(response.status(), await response.text()).toBe(409);
  const body = await response.json();
  expect(body.error).toBe("itinerary_conflict");
}

export async function listItineraryVersionsViaAPI(
  request: APIRequestContext,
  accessToken: string,
  tripId: string
) {
  const tripURL = process.env.E2E_TRIP_URL ?? "http://127.0.0.1:8080";
  const response = await request.get(`${tripURL}/trips/${tripId}/itinerary/versions`, {
    headers: { Authorization: `Bearer ${accessToken}` }
  });
  expect(response.ok(), await response.text()).toBeTruthy();
  return response.json() as Promise<{ items: Array<{ id: string; versionNumber: number; source: string }> }>;
}

export async function getBudgetSummaryViaAPI(
  request: APIRequestContext,
  accessToken: string,
  tripId: string
) {
  const tripURL = process.env.E2E_TRIP_URL ?? "http://127.0.0.1:8080";
  const response = await request.get(`${tripURL}/trips/${tripId}/budget-summary`, {
    headers: { Authorization: `Bearer ${accessToken}` }
  });
  expect(response.ok(), await response.text()).toBeTruthy();
  return response.json() as Promise<{ currency: string; estimatedTotal: number; estimatedItemCount: number }>;
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

export async function createProtectedPublicShareViaAPI(
  request: APIRequestContext,
  accessToken: string,
  tripId: string,
  password = "alpha-share-pass"
) {
  const tripURL = process.env.E2E_TRIP_URL ?? "http://127.0.0.1:8080";
  const expiresAt = new Date(Date.now() + 60 * 60 * 1000).toISOString();
  const response = await request.post(`${tripURL}/trips/${tripId}/share`, {
    headers: { Authorization: `Bearer ${accessToken}` },
    data: { password, expiresAt }
  });
  expect(response.ok(), await response.text()).toBeTruthy();
  return response.json() as Promise<{ shareToken: string; enabled: boolean; passwordRequired: boolean; expiresAt: string }>;
}

export async function disablePublicShareViaAPI(
  request: APIRequestContext,
  accessToken: string,
  tripId: string
) {
  const tripURL = process.env.E2E_TRIP_URL ?? "http://127.0.0.1:8080";
  const response = await request.delete(`${tripURL}/trips/${tripId}/share`, {
    headers: { Authorization: `Bearer ${accessToken}` }
  });
  expect(response.ok(), await response.text()).toBeTruthy();
}

export async function getPublicShareStatusViaAPI(request: APIRequestContext, shareToken: string) {
  const tripURL = process.env.E2E_TRIP_URL ?? "http://127.0.0.1:8080";
  const response = await request.get(`${tripURL}/public/trips/${shareToken}/status`);
  return response;
}

export async function unlockPublicShareViaAPI(
  request: APIRequestContext,
  shareToken: string,
  password: string
) {
  const tripURL = process.env.E2E_TRIP_URL ?? "http://127.0.0.1:8080";
  return request.post(`${tripURL}/public/trips/${shareToken}/unlock`, { data: { password } });
}

export async function getPublicTripViaAPI(
  request: APIRequestContext,
  shareToken: string,
  accessToken?: string
) {
  const tripURL = process.env.E2E_TRIP_URL ?? "http://127.0.0.1:8080";
  return request.get(`${tripURL}/public/trips/${shareToken}`, {
    headers: accessToken ? { Authorization: `Bearer ${accessToken}` } : undefined
  });
}

export async function getUnreadNotificationCountViaAPI(
  request: APIRequestContext,
  accessToken: string
) {
  const notificationURL = process.env.E2E_NOTIFICATION_URL ?? "http://127.0.0.1:8086";
  const response = await request.get(`${notificationURL}/notifications/unread-count`, {
    headers: { Authorization: `Bearer ${accessToken}` }
  });
  expect(response.ok(), await response.text()).toBeTruthy();
  return ((await response.json()) as { count?: number }).count ?? 0;
}

export async function markAllNotificationsReadViaAPI(
  request: APIRequestContext,
  accessToken: string
) {
  const notificationURL = process.env.E2E_NOTIFICATION_URL ?? "http://127.0.0.1:8086";
  const response = await request.patch(`${notificationURL}/notifications/read-all`, {
    headers: { Authorization: `Bearer ${accessToken}` }
  });
  expect(response.ok(), await response.text()).toBeTruthy();
}

export function cloneAndEditFirstItem(itinerary: TestItinerary, name: string): TestItinerary {
  const copy = JSON.parse(JSON.stringify(itinerary)) as TestItinerary;
  copy.days[0].items[0] = {
    ...copy.days[0].items[0],
    name,
    note: "Edited during alpha readiness E2E",
    estimatedCost: 12
  };
  return copy;
}

function unwrapTrip(raw: unknown): TestTrip {
  const body = raw as { trip?: TestTrip } & TestTrip;
  return body.trip ?? body;
}
