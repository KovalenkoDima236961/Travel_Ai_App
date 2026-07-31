export const TRIP_WORKSPACE_SECTION_IDS = [
  "overview",
  "plan",
  "money",
  "group",
  "prepare",
  "more"
] as const;

export type TripWorkspaceSectionId = (typeof TRIP_WORKSPACE_SECTION_IDS)[number];

export type TripWorkspaceSection = {
  id: TripWorkspaceSectionId;
  labelKey: TripWorkspaceSectionId;
  defaultView: string;
  views: readonly string[];
  mobilePrimary: boolean;
};

export const TRIP_WORKSPACE_SECTIONS: readonly TripWorkspaceSection[] = [
  {
    id: "overview",
    labelKey: "overview",
    defaultView: "summary",
    views: ["summary", "attention", "readiness", "activity"],
    mobilePrimary: true
  },
  {
    id: "plan",
    labelKey: "plan",
    defaultView: "itinerary",
    views: ["itinerary", "agenda", "timeline", "calendar", "route", "stay", "map", "verification"],
    mobilePrimary: true
  },
  {
    id: "money",
    labelKey: "money",
    defaultView: "overview",
    views: ["overview", "budget", "expenses", "receipts", "settlements", "splits"],
    mobilePrimary: true
  },
  {
    id: "group",
    labelKey: "group",
    defaultView: "people",
    views: ["people", "discussion", "decisions", "availability", "approvals", "activity"],
    mobilePrimary: false
  },
  {
    id: "prepare",
    labelKey: "prepare",
    defaultView: "checklist",
    views: ["checklist", "reminders", "offline"],
    mobilePrimary: true
  },
  {
    id: "more",
    labelKey: "more",
    defaultView: "tools",
    views: ["tools", "sharing", "exports", "versions", "health", "policy", "settings"],
    mobilePrimary: true
  }
] as const;

type LegacyDestination = {
  section: TripWorkspaceSectionId;
  view: string;
};

/**
 * Compatibility map for links emitted by historical notifications, search
 * results, emails, health actions, and browser bookmarks.
 */
export const LEGACY_TRIP_TAB_MAP: Readonly<Record<string, LegacyDestination>> = {
  overview: { section: "overview", view: "summary" },
  command_center: { section: "overview", view: "summary" },
  commandCenter: { section: "overview", view: "summary" },
  itinerary: { section: "plan", view: "itinerary" },
  agenda: { section: "plan", view: "agenda" },
  timeline: { section: "plan", view: "timeline" },
  calendar: { section: "plan", view: "calendar" },
  route: { section: "plan", view: "route" },
  transport: { section: "plan", view: "route" },
  accommodation: { section: "plan", view: "stay" },
  map: { section: "plan", view: "map" },
  verification: { section: "plan", view: "verification" },
  budget: { section: "money", view: "budget" },
  expenses: { section: "money", view: "expenses" },
  receipts: { section: "money", view: "receipts" },
  settlements: { section: "money", view: "settlements" },
  cost_split: { section: "money", view: "splits" },
  "cost-split": { section: "money", view: "splits" },
  collaborators: { section: "group", view: "people" },
  team: { section: "group", view: "people" },
  sharing: { section: "group", view: "people" },
  comments: { section: "group", view: "discussion" },
  polls: { section: "group", view: "decisions" },
  decisions: { section: "group", view: "decisions" },
  dates: { section: "group", view: "availability" },
  availability: { section: "group", view: "availability" },
  "group-readiness": { section: "group", view: "activity" },
  activity: { section: "group", view: "activity" },
  approval: { section: "group", view: "approvals" },
  checklist: { section: "prepare", view: "checklist" },
  reminders: { section: "prepare", view: "reminders" },
  offline: { section: "prepare", view: "offline" },
  health: { section: "more", view: "health" },
  policy: { section: "more", view: "policy" },
  "workspace-policy": { section: "more", view: "policy" },
  versions: { section: "more", view: "versions" },
  export: { section: "more", view: "exports" },
  settings: { section: "more", view: "settings" },
  recap: { section: "more", view: "tools" }
};

export type TripWorkspaceLocation = {
  tripId: string | null;
  section: TripWorkspaceSectionId;
  view: string;
  legacyTab: string | null;
  isLegacy: boolean;
};

export function isTripWorkspaceSection(value: string | null | undefined): value is TripWorkspaceSectionId {
  return TRIP_WORKSPACE_SECTION_IDS.includes(value as TripWorkspaceSectionId);
}

export function getTripWorkspaceSection(section: TripWorkspaceSectionId) {
  return TRIP_WORKSPACE_SECTIONS.find((candidate) => candidate.id === section) ?? TRIP_WORKSPACE_SECTIONS[0];
}

export function resolveTripWorkspaceLocation(
  pathname: string | null | undefined,
  searchParams: Pick<URLSearchParams, "get">
): TripWorkspaceLocation {
  const pathMatch = /^\/trips\/([^/]+)(?:\/([^/]+))?\/?$/.exec(pathname ?? "");
  const tripId = pathMatch?.[1] && pathMatch[1] !== "new" ? decodeURIComponent(pathMatch[1]) : null;
  const pathSection = isTripWorkspaceSection(pathMatch?.[2]) ? pathMatch[2] : null;
  const requestedSection = isTripWorkspaceSection(searchParams.get("section"))
    ? searchParams.get("section") as TripWorkspaceSectionId
    : null;
  const legacyTab = searchParams.get("tab");
  const legacyDestination = legacyTab ? LEGACY_TRIP_TAB_MAP[legacyTab] : null;
  const section = pathSection ?? requestedSection ?? legacyDestination?.section ?? "overview";
  const sectionConfig = getTripWorkspaceSection(section);
  const requestedView = searchParams.get("view") ?? legacyDestination?.view ?? sectionConfig.defaultView;
  const view = sectionConfig.views.includes(requestedView)
    ? requestedView
    : sectionConfig.defaultView;

  return {
    tripId,
    section,
    view,
    legacyTab,
    isLegacy: Boolean(legacyTab || requestedSection) && !pathSection
  };
}

export function buildTripWorkspaceHref(
  tripId: string,
  section: TripWorkspaceSectionId,
  input?: URLSearchParams | Record<string, string | number | boolean | null | undefined>
) {
  const params = input instanceof URLSearchParams
    ? new URLSearchParams(input)
    : new URLSearchParams(
        Object.entries(input ?? {}).flatMap(([key, value]) =>
          value == null || value === "" ? [] : [[key, String(value)]]
        )
      );
  params.delete("tab");
  params.delete("section");
  const sectionConfig = getTripWorkspaceSection(section);
  const view = params.get("view");
  if (view && !sectionConfig.views.includes(view)) {
    params.set("view", sectionConfig.defaultView);
  }
  const query = params.toString();
  return `/trips/${encodeURIComponent(tripId)}/${section}${query ? `?${query}` : ""}`;
}

export function normalizeTripWorkspaceHref(href: string): string {
  if (!href.startsWith("/trips/")) {
    return href;
  }
  const url = new URL(href, "https://travel-ai.invalid");
  const location = resolveTripWorkspaceLocation(url.pathname, url.searchParams);
  if (!location.tripId) {
    return href;
  }
  const params = new URLSearchParams(url.searchParams);
  params.delete("tab");
  params.delete("section");
  if (!params.has("view") && location.view !== getTripWorkspaceSection(location.section).defaultView) {
    params.set("view", location.view);
  }
  normalizeEntityParameters(params, location.section, location.view);
  return buildTripWorkspaceHref(location.tripId, location.section, params);
}

export function legacyTabForWorkspaceLocation(
  section: TripWorkspaceSectionId,
  view: string
): string {
  const exact = Object.entries(LEGACY_TRIP_TAB_MAP).find(
    ([, destination]) => destination.section === section && destination.view === view
  );
  if (exact) {
    return exact[0];
  }
  return {
    overview: "overview",
    plan: "itinerary",
    money: "budget",
    group: "sharing",
    prepare: "checklist",
    more: "health"
  }[section];
}

export function workspaceLoadSections(section: TripWorkspaceSectionId, view: string): string[] {
  const base: Record<TripWorkspaceSectionId, string[]> = {
    overview: ["overview", "health", "verification", "activity"],
    plan: ["itinerary", "route", "map", "verification", "weather"],
    money: ["budget", "expenses", "cost-split"],
    group: ["group-readiness", "dates", "decisions", "sharing", "activity", "itinerary"],
    prepare: ["checklist", "reminders", "offline"],
    more: ["health", "workspace-policy", "approval", "sharing", "itinerary"]
  };
  const viewAnchor = legacyTabForWorkspaceLocation(section, view);
  return Array.from(new Set([...base[section], viewAnchor]));
}

function normalizeEntityParameters(
  params: URLSearchParams,
  section: TripWorkspaceSectionId,
  view: string
) {
  const aliases: Array<[string, string]> = [
    ["legId", "leg"],
    ["stopId", "stop"],
    ["expenseId", "expense"],
    ["receiptId", "receipt"],
    ["commentId", "comment"],
    ["collaboratorId", "member"],
    ["memberId", "member"],
    ["reminderId", "reminder"],
    ["pollId", "decision"],
    ["versionId", "version"],
    ["eventId", "event"],
    ["issueId", "issue"]
  ];
  if (params.has("itemId")) {
    aliases.push(["itemId", "item"]);
  }
  for (const [legacy, canonical] of aliases) {
    const value = params.get(legacy);
    if (value && !params.has(canonical)) {
      params.set(canonical, value);
      params.delete(legacy);
    }
  }
  if (section === "group" && view === "people" && params.has("collaborator")) {
    params.set("member", params.get("collaborator") ?? "");
    params.delete("collaborator");
  }
}
