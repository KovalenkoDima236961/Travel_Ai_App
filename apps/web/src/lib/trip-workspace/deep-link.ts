import {
  legacyTabForWorkspaceLocation,
  resolveTripWorkspaceLocation,
  type TripWorkspaceSectionId
} from "./navigation";

export type TripDeepLinkResolutionState =
  | "resolving"
  | "resolved"
  | "not_found"
  | "forbidden"
  | "feature_disabled"
  | "offline_unavailable";

export type TripWorkspaceDeepLink = {
  state: TripDeepLinkResolutionState;
  section: TripWorkspaceSectionId;
  view: string;
  legacyTab: string;
  sectionId: string;
  targetId?: string;
};

export function resolveTripWorkspaceDeepLink({
  pathname,
  searchParams,
  featureEnabled = true,
  hasTripAccess = true,
  offline = false,
  targetAvailableOffline = true
}: {
  pathname: string | null | undefined;
  searchParams: URLSearchParams;
  featureEnabled?: boolean;
  hasTripAccess?: boolean;
  offline?: boolean;
  targetAvailableOffline?: boolean;
}): TripWorkspaceDeepLink {
  const location = resolveTripWorkspaceLocation(pathname, searchParams);
  const legacyTab = location.legacyTab ?? legacyTabForWorkspaceLocation(location.section, location.view);
  const target = getDeepLinkDomTarget(legacyTab, searchParams);
  const state: TripDeepLinkResolutionState = !hasTripAccess
    ? "forbidden"
    : !featureEnabled
      ? "feature_disabled"
      : offline && !targetAvailableOffline
        ? "offline_unavailable"
        : "resolving";
  return {
    state,
    section: location.section,
    view: location.view,
    legacyTab,
    sectionId: target?.sectionId ?? sectionAnchor(location.section, location.view),
    targetId: target?.targetId
  };
}

export type DeepLinkDomTarget = {
  sectionId: string;
  targetId?: string;
};

/** Maps both canonical v2 entity parameters and legacy parameter names. */
export function getDeepLinkDomTarget(
  tab: string,
  params: Pick<URLSearchParams, "get">
): DeepLinkDomTarget | null {
  const sectionId = legacySectionAnchor(tab);
  if (!sectionId) {
    return null;
  }
  if (tab === "route" || tab === "transport") {
    const legId = params.get("leg") ?? params.get("legId");
    const stopId = params.get("stop") ?? params.get("stopId");
    return {
      sectionId,
      targetId: legId
        ? `route-leg-${legId}`
        : stopId
          ? `route-stop-${stopId}`
          : undefined
    };
  }
  if (tab === "itinerary" || tab === "agenda" || tab === "timeline" || tab === "calendar") {
    const itemId = params.get("item") ?? params.get("itemId");
    const day = params.get("day");
    const itemIndex = params.get("itemIndex");
    return {
      sectionId,
      targetId: itemId
        ? `itinerary-item-${itemId}`
        : day && itemIndex
          ? `day-${day}-item-${itemIndex}`
          : undefined
    };
  }
  const targetByTab: Record<string, [readonly string[], string]> = {
    budget: [["category"], "budget-category-"],
    health: [["issue", "issueId"], "trip-health-issue-"],
    expenses: [["expense", "expenseId"], "expense-"],
    receipts: [["receipt", "receiptId"], "receipt-"],
    checklist: [["item", "itemId"], "checklist-item-"],
    reminders: [["reminder", "reminderId"], "reminder-"],
    polls: [["decision", "pollId"], "poll-"],
    decisions: [["decision", "pollId"], "poll-"],
    collaborators: [["member", "collaboratorId", "memberId"], "collaborator-"],
    activity: [["event", "eventId"], "activity-event-"],
    comments: [["comment", "commentId"], "comment-"],
    versions: [["version", "versionId"], "itinerary-version-"]
  };
  const targetConfig = targetByTab[tab];
  if (!targetConfig) {
    return { sectionId };
  }
  const value = firstParam(params, targetConfig[0]);
  return { sectionId, targetId: value ? `${targetConfig[1]}${value}` : undefined };
}

function firstParam(params: Pick<URLSearchParams, "get">, keys: readonly string[]) {
  for (const key of keys) {
    const value = params.get(key);
    if (value) {
      return value;
    }
  }
  return null;
}

function sectionAnchor(section: TripWorkspaceSectionId, view: string) {
  return legacySectionAnchor(legacyTabForWorkspaceLocation(section, view)) ?? "overview";
}

function legacySectionAnchor(tab: string): string | null {
  const anchors: Record<string, string> = {
    overview: "overview",
    command_center: "overview",
    commandCenter: "overview",
    itinerary: "itinerary",
    agenda: "itinerary",
    timeline: "itinerary",
    calendar: "itinerary",
    route: "route",
    transport: "route",
    accommodation: "accommodation",
    map: "map",
    dates: "dates",
    availability: "dates",
    polls: "decisions",
    decisions: "decisions",
    budget: "budget",
    expenses: "expenses",
    settlements: "expenses",
    receipts: "expenses",
    checklist: "checklist",
    reminders: "reminders",
    offline: "offline",
    collaborators: "sharing",
    team: "sharing",
    sharing: "sharing",
    activity: "activity",
    comments: "itinerary",
    health: "health",
    verification: "verification",
    approval: "approval",
    policy: "workspace-policy",
    "workspace-policy": "workspace-policy",
    versions: "itinerary"
  };
  return anchors[tab] ?? null;
}
