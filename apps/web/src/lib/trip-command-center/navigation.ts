import type { NavigationGroup } from "@/types/trip-command-center";

export const TAB_TO_ANCHOR: Record<string, string> = {
  overview: "overview",
  command_center: "overview",
  commandCenter: "overview",
  itinerary: "itinerary",
  route: "route",
  transport: "route",
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
  sharing: "sharing",
  activity: "activity",
  comments: "itinerary",
  health: "health",
  approval: "approval",
  policy: "workspace-policy",
  versions: "itinerary"
};

export function buildNavigationGroups({
  tripId,
  badges = {}
}: {
  tripId: string;
  badges?: Partial<Record<string, number | string | null>>;
}): NavigationGroup[] {
  return [
    {
      id: "plan",
      label: "Plan",
      items: [
        { id: "overview", label: "Overview", href: "#overview" },
        { id: "itinerary", label: "Itinerary", href: "#itinerary" },
        { id: "route", label: "Route & Transport", href: "#route", badge: badges.route },
        { id: "dates", label: "Dates", href: "#dates", badge: badges.dates },
        { id: "polls", label: "Polls", href: "#decisions", badge: badges.polls }
      ]
    },
    {
      id: "prepare",
      label: "Prepare",
      items: [
        { id: "checklist", label: "Checklist", href: "#checklist", badge: badges.checklist },
        { id: "reminders", label: "Reminders", href: "#reminders", badge: badges.reminders },
        { id: "offline", label: "Offline", href: "#offline", badge: badges.offline }
      ]
    },
    {
      id: "money",
      label: "Money",
      items: [
        { id: "budget", label: "Budget", href: "#budget", badge: badges.budget },
        { id: "expenses", label: "Expenses", href: "#expenses", badge: badges.expenses },
        { id: "settlements", label: "Settlements", href: "#expenses", badge: badges.settlements },
        { id: "receipts", label: "Receipts", href: "#expenses" }
      ]
    },
    {
      id: "team",
      label: "Team",
      items: [
        { id: "collaborators", label: "Collaborators", href: "#sharing" },
        { id: "activity", label: "Activity", href: "#activity" },
        { id: "comments", label: "Comments", href: "#itinerary" }
      ]
    },
    {
      id: "control",
      label: "Control",
      items: [
        { id: "health", label: "Health", href: "#health", badge: badges.health },
        { id: "approval", label: "Approval", href: "#approval", badge: badges.approval },
        { id: "policy", label: "Policy", href: "#workspace-policy", badge: badges.policy },
        { id: "versions", label: "Versions", href: "#itinerary" },
        { id: "overview", label: "Analytics", href: `/trips/${tripId}/analytics` }
      ]
    }
  ];
}

export function scrollToTabAnchor(tab: string | null | undefined) {
  if (!tab || typeof window === "undefined") {
    return;
  }
  const anchor = TAB_TO_ANCHOR[tab];
  if (!anchor) {
    return;
  }
  window.requestAnimationFrame(() => {
    const params = new URLSearchParams(window.location.search);
    const legId = params.get("legId");
    const stopId = params.get("stopId");
    const focusedRouteAnchor =
      tab === "route" || tab === "transport"
        ? legId
          ? `route-leg-${legId}`
          : stopId
            ? `route-stop-${stopId}`
            : null
        : null;
    document.getElementById(focusedRouteAnchor ?? anchor)?.scrollIntoView({ block: "start" });
  });
}
