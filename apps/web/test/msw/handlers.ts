import { http, HttpResponse } from "msw";
import { budgetSummaryFixture } from "../fixtures/budget";
import { notificationsFixture } from "../fixtures/notifications";
import { tripFixture, tripsListFixture } from "../fixtures/trips";
import { ownerAuthUser, ownerProfile, ownerPreferences } from "../fixtures/users";

export const handlers = [
  http.get("*/auth/me", () => HttpResponse.json(ownerAuthUser)),
  http.get("*/users/me/profile", () => HttpResponse.json(ownerProfile)),
  http.get("*/users/me/preferences", () => HttpResponse.json(ownerPreferences)),
  http.get("*/trips", () => HttpResponse.json(tripsListFixture)),
  http.get("*/trips/:tripId", () => HttpResponse.json(tripFixture)),
  http.get("*/trips/:tripId/budget", () => HttpResponse.json(budgetSummaryFixture)),
  http.get("*/notifications", () => HttpResponse.json({ items: notificationsFixture })),
  http.get("*/notifications/unread-count", () => HttpResponse.json({ count: 1 }))
];
