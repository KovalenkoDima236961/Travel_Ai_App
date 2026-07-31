import type { NavigationGroup } from "@/types/trip-command-center";
import {
  getDeepLinkDomTarget,
  type DeepLinkDomTarget
} from "@/lib/trip-workspace/deep-link";

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
  verification: "verification",
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
        { id: "verification", label: "Verification", href: "#verification", badge: badges.verification },
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
  const behavior: ScrollBehavior = window.matchMedia("(prefers-reduced-motion: reduce)").matches
    ? "auto"
    : "smooth";
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
        element.scrollIntoView({ behavior, block: "center" });
        if (target?.targetId) {
          highlightDeepLinkTarget(element);
        }
        window.dispatchEvent(
          new CustomEvent("travel-ai:deep-link-resolved", {
            detail: { tab, targetId: target?.targetId ?? null }
          })
        );
        return;
      }
      if (index === delays.length - 1) {
        document.getElementById(sectionId)?.scrollIntoView({ behavior, block: "start" });
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
  return getDeepLinkDomTarget(tab, params);
}

function highlightDeepLinkTarget(element: HTMLElement) {
  const hadTabIndex = element.hasAttribute("tabindex");
  element.dataset.deepLinkHighlighted = "true";
  element.classList.add("ring-2", "ring-primary-600", "ring-offset-2");
  if (!hadTabIndex) {
    element.setAttribute("tabindex", "-1");
  }
  element.focus({ preventScroll: true });
  const clear = () => {
    element.classList.remove("ring-2", "ring-primary-600", "ring-offset-2");
    delete element.dataset.deepLinkHighlighted;
    if (!hadTabIndex) {
      element.removeAttribute("tabindex");
    }
    window.removeEventListener("pointerdown", clear);
    window.removeEventListener("keydown", clear);
  };
  window.addEventListener("pointerdown", clear, { once: true });
  window.addEventListener("keydown", clear, { once: true });
  window.setTimeout(clear, 2400);
}

export type { DeepLinkDomTarget };
