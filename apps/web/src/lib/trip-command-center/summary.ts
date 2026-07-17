import { formatCurrencyAmount } from "./format";
import { buildNavigationGroups } from "./navigation";
import type { CommandCenterSummary } from "@/lib/api/command-center";
import type {
  NextBestAction,
  OfflineCommandCenterStatus,
  ReadinessCard,
  ReadinessCardStatus,
  TripCommandCenterData
} from "@/types/trip-command-center";
import type { TripHealth } from "@/types/trip-health";

export function healthFromCommandCenterSummary(summary: CommandCenterSummary): TripHealth | null {
  const health = summary.health;
  if (!health) {
    return null;
  }
  return {
    tripId: summary.tripId,
    score: health.score,
    level: health.level,
    summary: health.summary,
    generatedAt: summary.computedAt,
    categories: [],
    issues: health.topFixes.map((fix) => ({
      id: fix.id,
      category: fix.category,
      severity: fix.severity || "warning",
      status: "open",
      title: fix.title,
      description: fix.description,
      recommendation: fix.recommendation,
      action: { type: "navigate", label: fix.label, href: fix.href }
    })),
    topFixes: health.topFixes.map((fix) => ({
      issueId: fix.id,
      label: fix.label,
      href: fix.href
    })),
    computedFrom: { itineraryRevision: summary.trip.itineraryRevision }
  };
}

export function buildTripCommandCenterDataFromSummary(
  summary: CommandCenterSummary,
  offline: OfflineCommandCenterStatus
): TripCommandCenterData {
  const topFixes = summary.health?.topFixes.map(toAction) ?? [];
  const fallbackAction: NextBestAction = {
    id: "review-overview",
    title: "Review your trip plan",
    description: "Open the itinerary and confirm the remaining trip details.",
    reason: "Keep the plan current before departure.",
    severity: "info",
    category: "other",
    actionLabel: "Open itinerary",
    href: "#itinerary",
    source: "trip",
    viewOnly: !summary.trip.canEdit
  };
  const cards = buildCards(summary, offline);
  return {
    nextBestAction: topFixes[0] ?? fallbackAction,
    topFixes,
    cards,
    navigationGroups: buildNavigationGroups({
      tripId: summary.tripId,
      badges: {
        health: (summary.health?.criticalIssueCount ?? 0) + (summary.health?.highIssueCount ?? 0),
        route: summary.route.missingTransportCount,
        checklist: summary.checklist?.highPriorityCount ?? 0,
        reminders: summary.reminders?.overdueCount ?? 0,
        expenses: summary.expenses?.expenseCount ?? 0,
        settlements: summary.expenses?.pendingSettlementCount ?? 0,
        budget: summary.budget?.budgetExceeded ? "over" : summary.budget?.missingEstimateCount || null,
        offline: offline.failedCount || offline.pendingCount
      }
    }),
    recentActivity: summary.activity?.items ?? []
  };
}

function buildCards(
  summary: CommandCenterSummary,
  offline: OfflineCommandCenterStatus
): ReadinessCard[] {
  const cards: ReadinessCard[] = [];
  if (summary.health) {
    cards.push({
      id: "health",
      title: "Trip Health",
      status: levelStatus(summary.health.level),
      score: summary.health.score,
      summary: summary.health.summary,
      detail: `${summary.health.criticalIssueCount} critical, ${summary.health.highIssueCount} high, ${summary.health.warningIssueCount} warning issue(s).`,
      metrics: [
        { label: "Score", value: String(summary.health.score) },
        { label: "Critical", value: String(summary.health.criticalIssueCount) },
        { label: "High", value: String(summary.health.highIssueCount) }
      ],
      primaryAction: { label: "Open Health", href: "#health" }
    });
  } else if (hasSectionError(summary, "health")) {
    cards.push(unavailableCard("health", "Trip Health", "#health"));
  }
  cards.push({
    id: "route_transport",
    title: "Route & Transport",
    status:
      summary.route.legCount === 0
        ? "empty"
        : summary.route.missingTransportCount > 0
          ? "needs_attention"
          : "ready",
    score: summary.route.legCount > 0 ? summary.route.selectedTransportCoverage : null,
    summary:
      summary.route.legCount > 0
        ? `${summary.route.legCount - summary.route.missingTransportCount} of ${summary.route.legCount} route leg(s) have selected transport.`
        : "No route legs are defined yet.",
    metrics: [
      { label: "Stops", value: String(summary.route.stopCount) },
      { label: "Coverage", value: `${summary.route.selectedTransportCoverage}%` },
      { label: "Missing", value: String(summary.route.missingTransportCount) }
    ],
    primaryAction: { label: "Open Route", href: "#route" }
  });
  if (summary.budget) {
    cards.push({
      id: "budget",
      title: "Budget",
      status: budgetStatus(summary.budget.riskLevel, summary.budget.confidenceLevel),
      score: summary.budget.confidenceScore,
      summary: summary.budget.summary,
      detail: summary.budget.missingEstimateCount
        ? `${summary.budget.missingEstimateCount} item(s) still need estimates.`
        : null,
      metrics: [
        { label: "Confidence", value: `${summary.budget.confidenceScore}/100` },
        { label: "Coverage", value: `${summary.budget.coverage}%` },
        {
          label: "Estimated",
          value: formatCurrencyAmount(summary.budget.estimatedTotal.amount, summary.budget.currency)
        }
      ],
      primaryAction: { label: "Open Budget", href: "#budget" }
    });
  } else if (hasSectionError(summary, "budget")) {
    cards.push(unavailableCard("budget", "Budget", "#budget"));
  }
  if (summary.groupReadiness) {
    cards.push({
      id: "group",
      title: "Group Readiness",
      status: levelStatus(summary.groupReadiness.level),
      score: summary.groupReadiness.score,
      summary: summary.groupReadiness.summary,
      metrics: [
        { label: "Members", value: String(summary.groupReadiness.memberCount) },
        { label: "Attention", value: String(summary.groupReadiness.membersNeedingAttention) }
      ],
      primaryAction: {
        label: summary.groupReadiness.topActionLabel || "Open Group Readiness",
        href: summary.groupReadiness.topActionHref || "#group-readiness"
      }
    });
  } else if (hasSectionError(summary, "groupReadiness")) {
    cards.push(unavailableCard("group", "Group Readiness", "#group-readiness"));
  }
  if (summary.checklist || summary.reminders) {
    const outstanding =
      (summary.checklist?.totalCount ?? 0) - (summary.checklist?.completedCount ?? 0);
    cards.push({
      id: "checklist_reminders",
      title: "Checklist & Reminders",
      status:
        (summary.checklist?.overdueCount ?? 0) + (summary.reminders?.overdueCount ?? 0) > 0
          ? "needs_attention"
          : outstanding > 0
            ? "almost_ready"
            : "ready",
      summary: `${outstanding} checklist item(s) and ${summary.reminders?.overdueCount ?? 0} overdue reminder(s) remain.`,
      metrics: [
        { label: "Done", value: `${summary.checklist?.completedCount ?? 0}/${summary.checklist?.totalCount ?? 0}` },
        { label: "Overdue", value: String((summary.checklist?.overdueCount ?? 0) + (summary.reminders?.overdueCount ?? 0)) }
      ],
      primaryAction: { label: "Open Checklist", href: "#checklist" },
      secondaryAction: { label: "Open Reminders", href: "#reminders" }
    });
  } else if (
    hasSectionError(summary, "checklist") ||
    hasSectionError(summary, "reminders")
  ) {
    cards.push(
      unavailableCard("checklist_reminders", "Checklist & Reminders", "#checklist")
    );
  }
  if (summary.expenses) {
    cards.push({
      id: "expenses_settlements",
      title: "Expenses & Settlements",
      status: summary.expenses.pendingSettlementCount > 0 ? "needs_attention" : "ready",
      summary: `${summary.expenses.expenseCount} expense(s) recorded; ${summary.expenses.pendingSettlementCount} settlement(s) pending.`,
      metrics: [
        {
          label: "Actual",
          value: formatCurrencyAmount(summary.expenses.actualTotal.amount, summary.expenses.actualTotal.currency)
        },
        { label: "Expenses", value: String(summary.expenses.expenseCount) },
        { label: "Pending", value: String(summary.expenses.pendingSettlementCount) }
      ],
      primaryAction: { label: "Open Expenses", href: "#expenses" }
    });
  } else if (hasSectionError(summary, "expenses")) {
    cards.push(unavailableCard("expenses_settlements", "Expenses & Settlements", "#expenses"));
  }
  if (summary.activity) {
    cards.push({
      id: "activity",
      title: "Recent Activity",
      status: "ready",
      summary: `${summary.activity.recentCount} recent update(s).`,
      metrics: [{ label: "Recent", value: String(summary.activity.recentCount) }],
      primaryAction: { label: "Open Activity", href: "#activity" }
    });
  } else if (hasSectionError(summary, "activity")) {
    cards.push(unavailableCard("activity", "Recent Activity", "#activity"));
  }
  cards.push({
    id: "offline",
    title: "Offline",
    status: offline.failedCount || offline.conflictCount ? "needs_attention" : "ready",
    summary: offline.online
      ? offline.availableOffline
        ? "An offline copy is available on this device."
        : "Online; save this trip for offline use when needed."
      : "You are viewing the available offline trip data.",
    metrics: [
      { label: "Pending", value: String(offline.pendingCount) },
      { label: "Failed", value: String(offline.failedCount) }
    ],
    primaryAction: { label: "Offline status", href: "#offline" }
  });
  return cards;
}

function hasSectionError(summary: CommandCenterSummary, section: string) {
  return summary.sectionErrors.some((error) => error.section === section);
}

function unavailableCard(
  id: ReadinessCard["id"],
  title: string,
  href: string
): ReadinessCard {
  return {
    id,
    title,
    status: "unavailable",
    summary: `${title} is temporarily unavailable. Other trip data is still usable.`,
    metrics: [],
    primaryAction: { label: `Open ${title}`, href }
  };
}

function toAction(fix: NonNullable<CommandCenterSummary["health"]>["topFixes"][number]): NextBestAction {
  return {
    id: fix.id,
    title: fix.title,
    description: fix.description,
    reason: fix.recommendation || fix.description,
    severity: fix.severity || "warning",
    category: fix.category || "other",
    actionLabel: fix.label,
    href: fix.href,
    source: "trip_health"
  };
}

function levelStatus(level: string): ReadinessCardStatus {
  return level === "not_ready" ? "blocked" : (level as ReadinessCardStatus);
}

function budgetStatus(risk: string, confidence: string): ReadinessCardStatus {
  if (risk === "critical") return "blocked";
  if (risk === "high" || confidence === "low" || confidence === "very_low") return "needs_attention";
  if (risk === "medium" || confidence === "medium") return "almost_ready";
  return "ready";
}
