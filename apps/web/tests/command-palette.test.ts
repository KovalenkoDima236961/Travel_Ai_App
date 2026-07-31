import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  commandToResult,
  filterCommands,
  getCommandRegistry
} from "@/lib/command-palette/commands";
import {
  getRecentCommandPaletteItems,
  recordRecentCommandPaletteItem
} from "@/lib/command-palette/recent-items";
import { buildOfflineCachedResults } from "@/lib/command-palette/local-search";
import { resetOfflineDbForTests } from "@/lib/offline/db";
import {
  cacheChecklistSnapshot,
  cacheTripSnapshot,
  clearOfflineData
} from "@/lib/offline/trip-cache";
import type { Trip } from "@/entities/trip/model";
import type { SearchResult } from "@/types/search";

const t = (key: string) => key;

describe("command palette helpers", () => {
  beforeEach(() => {
    const store = new Map<string, string>();
    vi.stubGlobal("window", {
      localStorage: {
        getItem: (key: string) => store.get(key) ?? null,
        setItem: (key: string, value: string) => store.set(key, value),
        removeItem: (key: string) => store.delete(key)
      }
    });
    resetOfflineDbForTests();
  });

  it("keeps trip-scoped edit commands permission aware", () => {
    const commands = getCommandRegistry(t);
    const upload = commands.find((command) => command.id === "trip.uploadReceipt");
    expect(upload).toBeDefined();

    const hiddenWithoutTrip = filterCommands(commands, "upload", {
      currentTripId: null,
      canEditCurrentTrip: true
    });
    expect(hiddenWithoutTrip.map((command) => command.id)).not.toContain("trip.uploadReceipt");

    const disabled = commandToResult(upload!, {
      currentTripId: "trip-1",
      canEditCurrentTrip: false
    });
    expect(disabled).toBeNull();

    const enabled = commandToResult(upload!, {
      currentTripId: "trip-1",
      canEditCurrentTrip: true
    });
    expect(enabled?.href).toBe("/trips/trip-1?tab=receipts&action=upload");
  });

  it("stores recent items per user and de-duplicates by id", () => {
    const first = result("trip:1", "/trips/1");
    const second = result("trip:2", "/trips/2");

    recordRecentCommandPaletteItem("user-a", first);
    recordRecentCommandPaletteItem("user-b", second);
    recordRecentCommandPaletteItem("user-a", { ...first, title: "Updated Rome" });

    expect(getRecentCommandPaletteItems("user-a")).toHaveLength(1);
    expect(getRecentCommandPaletteItems("user-a")[0].title).toBe("Updated Rome");
    expect(getRecentCommandPaletteItems("user-b")[0].id).toBe("trip:2");
  });

  it("includes first-run and demo commands", () => {
    const ids = getCommandRegistry(t).map((command) => command.id);
    expect(ids).toContain("onboarding.gettingStarted");
    expect(ids).toContain("onboarding.createFirstTrip");
    expect(ids).toContain("onboarding.discovery");
    expect(ids).toContain("onboarding.demoTrip");
    expect(ids).toContain("onboarding.restart");
  });

  it("searches only the current user's offline cached companion data", async () => {
    await clearOfflineData();
    const trip = offlineTrip("trip-offline", "Vienna");
    await cacheTripSnapshot({ userId: "user-a", trip });
    await cacheChecklistSnapshot({
      tripId: trip.id,
      userId: "user-a",
      checklist: {
        checklist: {
          id: "checklist-1",
          tripId: trip.id,
          status: "active",
          title: "Offline checklist",
          createdByUserId: "user-a",
          updatedAt: trip.updatedAt,
          createdAt: trip.createdAt,
          items: [
            {
              id: "passport-item",
              checklistId: "checklist-1",
              title: "Passport",
              description: "Bring a valid passport",
              category: "documents",
              itemType: "document",
              priority: "critical",
              checked: false,
              source: "manual",
              sortOrder: 1,
              createdAt: trip.createdAt,
              updatedAt: trip.updatedAt
            }
          ]
        },
        canGenerate: false
      }
    });

    const ownerResults = await buildOfflineCachedResults("user-a", "passport");
    const otherResults = await buildOfflineCachedResults("user-b", "passport");

    expect(ownerResults.some((item) => item.type === "checklist_item")).toBe(true);
    expect(ownerResults[0]?.sourceService).toBe("offline-cache");
    expect(ownerResults[0]?.metadata?.offline).toBe(true);
    expect(otherResults).toHaveLength(0);
  });
});

function result(id: string, href: string): SearchResult {
  return {
    id,
    type: "trip",
    title: "Rome",
    href,
    icon: "map",
    category: "Trips",
    score: 1
  };
}

function offlineTrip(id: string, destination: string): Trip {
  return {
    id,
    userId: "user-a",
    destination,
    days: 3,
    budgetCurrency: "EUR",
    travelers: 1,
    interests: [],
    pace: "balanced",
    status: "DRAFT",
    itineraryRevision: 1,
    createdAt: "2026-07-30T12:00:00Z",
    updatedAt: "2026-07-30T12:00:00Z"
  };
}
