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
  team: "sharing",
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
  const params = new URLSearchParams(window.location.search);
  const target = getDeepLinkTarget(tab, params);
  const targetId = target?.targetId ?? anchor;
  const sectionId = target?.sectionId ?? anchor;
  const delays = [0, 250, 750, 1500];
  const timers: number[] = [];
  let resolved = false;

  for (const [index, delay] of delays.entries()) {
    const timer = window.setTimeout(() => {
      if (resolved) {
        return;
      }
      const element = document.getElementById(targetId);
      if (element) {
        resolved = true;
        element.scrollIntoView({ behavior: "smooth", block: "center" });
        if (target?.targetId) {
          highlightDeepLinkTarget(element);
        }
        return;
      }
      if (index === delays.length - 1) {
        document.getElementById(sectionId)?.scrollIntoView({ behavior: "smooth", block: "start" });
        window.dispatchEvent(
          new CustomEvent("travel-ai:deep-link-missing", { detail: { tab, targetId } })
        );
      }
    }, delay);
    timers.push(timer);
  }
  return () => timers.forEach((timer) => window.clearTimeout(timer));
}

export type DeepLinkTarget = {
  sectionId: string;
  targetId?: string;
};

export function getDeepLinkTarget(
  tab: string,
  params: Pick<URLSearchParams, "get">
): DeepLinkTarget | null {
  const sectionId = TAB_TO_ANCHOR[tab];
  if (!sectionId) {
    return null;
  }
  if (tab === "route" || tab === "transport") {
    const legId = params.get("legId");
    const stopId = params.get("stopId");
    return {
      sectionId,
      targetId: legId
        ? `route-leg-${legId}`
        : stopId
          ? `route-stop-${stopId}`
        : undefined
    };
  }
  if (tab === "itinerary") {
    const itemId = params.get("itemId");
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
  const targetByTab: Record<string, [string, string]> = {
    budget: ["category", "budget-category-"],
    health: ["issueId", "trip-health-issue-"],
    expenses: ["expenseId", "expense-"],
    receipts: ["receiptId", "receipt-"],
    checklist: ["itemId", "checklist-item-"],
    reminders: ["reminderId", "reminder-"],
    polls: ["pollId", "poll-"],
    decisions: ["pollId", "poll-"],
    activity: ["eventId", "activity-event-"],
    comments: ["commentId", "comment-"]
  };
  const targetConfig = targetByTab[tab];
  if (!targetConfig) {
    return { sectionId };
  }
  const value = params.get(targetConfig[0]);
  return { sectionId, targetId: value ? `${targetConfig[1]}${value}` : undefined };
}

function highlightDeepLinkTarget(element: HTMLElement) {
  element.dataset.deepLinkHighlighted = "true";
  element.classList.add("ring-2", "ring-primary-600", "ring-offset-2");
  if (!element.hasAttribute("tabindex")) {
    element.setAttribute("tabindex", "-1");
  }
  element.focus({ preventScroll: true });
  window.setTimeout(() => {
    element.classList.remove("ring-2", "ring-primary-600", "ring-offset-2");
    delete element.dataset.deepLinkHighlighted;
  }, 2400);
}
