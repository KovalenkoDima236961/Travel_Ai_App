import { formatCurrencyAmount, formatPercent } from "./format";
import { buildNavigationGroups } from "./navigation";
import { buildTopFixActions, selectNextBestAction } from "./next-best-action";
import type {
  NavigationGroup,
  ReadinessCard,
  ReadinessCardStatus,
  TripCommandCenterData,
  TripCommandCenterInput
} from "@/types/trip-command-center";
import type { TripHealthIssueSeverity } from "@/types/trip-health";

export function buildTripCommandCenterData(input: TripCommandCenterInput): TripCommandCenterData {
  return {
    nextBestAction: selectNextBestAction(input),
    topFixes: buildTopFixActions(input),
    cards: buildReadinessCards(input),
    navigationGroups: buildCommandCenterNavigation(input),
    recentActivity: (input.activity ?? []).slice(0, 5)
  };
}

export function buildReadinessCards(input: TripCommandCenterInput): ReadinessCard[] {
  return [
    buildHealthCard(input),
    buildRouteReadinessCard(input),
    buildBudgetReadinessCard(input),
    buildGroupReadinessCard(input),
    buildChecklistReminderCard(input),
    buildExpenseSettlementCard(input),
    buildApprovalPolicyCard(input),
    buildActivityCard(input),
    buildOfflineStatusCard(input)
  ].filter((card) => card.status !== "unavailable" || card.id === "offline");
}

export function buildCommandCenterNavigation(input: TripCommandCenterInput): NavigationGroup[] {
  const criticalHealth = countHealth(input, ["critical", "high"]);
  const routeIssues = countHealthCategories(input, ["route", "transport"]);
  const checklistIssues = input.checklist?.summary?.highPriorityUnchecked ?? 0;
  const overdueReminders = input.reminders?.summary.overdue ?? 0;
  const pendingSettlements = input.expenseSummary?.settlementSummary.pendingCount ?? 0;
  const blockingPolicy = input.policyEvaluation?.summary.blockingCount ?? 0;
  const approvalBadge =
    input.approval?.status === "changes_requested" || input.approval?.status === "pending_approval"
      ? 1
      : 0;

  return buildNavigationGroups({
    tripId: input.trip.id,
    badges: {
      health: criticalHealth,
      route: routeIssues,
      dates: input.availability?.summary.missingCount ?? 0,
      polls: (input.polls ?? []).filter((poll) => poll.status === "open").length,
      checklist: checklistIssues,
      reminders: overdueReminders,
      expenses: input.expenseSummary?.conversionWarnings.length ?? 0,
      settlements: pendingSettlements,
      offline: input.offlineStatus.failedCount || input.offlineStatus.pendingCount,
      budget: budgetNavigationBadge(input),
      policy: blockingPolicy,
      approval: approvalBadge
    }
  });
}

export function buildHealthCard(input: TripCommandCenterInput): ReadinessCard {
  const health = input.health;
  if (!health) {
    return {
      id: "health",
      title: "Trip Health",
      status: "unavailable",
      summary: "Trip Health has not loaded yet.",
      metrics: [],
      primaryAction: { label: "Open Health", href: "#health" }
    };
  }
  const critical = countHealth(input, ["critical"]);
  const high = countHealth(input, ["high"]);
  const warnings = countHealth(input, ["warning"]);
  return {
    id: "health",
    title: "Trip Health",
    status: health.level === "not_ready" ? "blocked" : health.level,
    score: health.score,
    summary: health.summary,
    detail: `${critical} critical, ${high} high, ${warnings} warning issue(s).`,
    metrics: [
      { label: "Score", value: String(health.score) },
      { label: "Critical", value: String(critical) },
      { label: "High", value: String(high) }
    ],
    primaryAction: { label: "Open Health", href: "#health" }
  };
}

export function buildRouteReadinessCard(input: TripCommandCenterInput): ReadinessCard {
  const route = input.trip.route;
  const stops = route?.stops?.length ?? 0;
  const legs = route?.legs ?? [];
  const selectedLegs = legs.filter((leg) => Boolean(leg.selectedTransportOption)).length;
  const weakLegs = legs.filter(
    (leg) =>
      !leg.selectedTransportOption ||
      leg.selectedTransportOption.provider === "mock" ||
      leg.selectedTransportOption.confidence === "low"
  ).length;
  const routeIssues = countHealthCategories(input, ["route", "transport"]);
  const criticalRouteIssues = (input.health?.issues ?? []).filter(
    (issue) =>
      issue.status === "open" &&
      issue.severity === "critical" &&
      (issue.category === "route" || issue.category === "transport")
  ).length;

  if (input.trip.tripType !== "multi_destination" && legs.length === 0) {
    return {
      id: "route_transport",
      title: "Route & Transport",
      status: "ready",
      score: 100,
      summary: "Single-destination trip.",
      detail: "Route transport is optional for this trip.",
      metrics: [{ label: "Stops", value: String(Math.max(stops, 1)) }],
      primaryAction: { label: "Open itinerary", href: "#itinerary" }
    };
  }

  const status: ReadinessCardStatus =
    criticalRouteIssues > 0
      ? "blocked"
      : weakLegs > 0 || routeIssues > 0
        ? "needs_attention"
        : legs.length > 0
          ? "ready"
          : "empty";
  return {
    id: "route_transport",
    title: "Route & Transport",
    status,
    score: legs.length > 0 ? Math.round((selectedLegs / legs.length) * 100) : null,
    summary:
      legs.length > 0
        ? `${selectedLegs} of ${legs.length} route leg(s) have selected transport.`
        : "No route legs are defined yet.",
    detail: weakLegs > 0 ? `${weakLegs} leg(s) need real or higher-confidence transport.` : null,
    metrics: [
      { label: "Stops", value: String(stops) },
      { label: "Ready legs", value: legs.length > 0 ? `${selectedLegs}/${legs.length}` : "0/0" },
      { label: "Transport issues", value: String(routeIssues) }
    ],
    primaryAction: {
      label: weakLegs > 0 ? "Fix transport" : "Open Route",
      href: "#route"
    }
  };
}

export function buildBudgetReadinessCard(input: TripCommandCenterInput): ReadinessCard {
  const confidence = input.budgetConfidence;
  const summary = input.budgetSummary;
  if (confidence) {
    const topIssue = confidence.issues[0] ?? null;
    const status: ReadinessCardStatus =
      confidence.riskLevel === "critical"
        ? "blocked"
        : confidence.riskLevel === "high"
          ? "needs_attention"
          : confidence.riskLevel === "medium" || confidence.level === "medium"
            ? "almost_ready"
            : confidence.level === "low" || confidence.level === "very_low"
              ? "needs_attention"
              : "ready";
    return {
      id: "budget",
      title: "Budget",
      status,
      score: confidence.score,
      summary: confidence.summary,
      detail: topIssue ? `${topIssue.title}: ${topIssue.recommendation}` : null,
      metrics: [
        { label: "Confidence", value: `${confidence.score}/100` },
        { label: "Coverage", value: `${confidence.coverage.overall}%` },
        { label: "Risk", value: confidence.riskLevel.replaceAll("_", " ") }
      ],
      primaryAction: {
        label: status === "ready" ? "Open Budget" : "Review budget confidence",
        href: "#budget"
      }
    };
  }
  if (!summary) {
    return {
      id: "budget",
      title: "Budget",
      status: "unavailable",
      summary: "Budget summary is not available yet.",
      metrics: [],
      primaryAction: { label: "Open Budget", href: "#budget" }
    };
  }
  const utilization =
    summary.tripBudget && summary.tripBudget > 0
      ? (summary.estimatedTotal / summary.tripBudget) * 100
      : null;
  const status: ReadinessCardStatus =
    (summary.overBudgetBy ?? 0) > 0
      ? "needs_attention"
      : summary.missingEstimateCount > 0 || (summary.conversionWarnings?.length ?? 0) > 0
        ? "almost_ready"
        : "ready";
  return {
    id: "budget",
    title: "Budget",
    status,
    score: utilization == null ? null : Math.max(0, Math.min(100, Math.round(100 - Math.max(utilization - 100, 0)))),
    summary:
      (summary.overBudgetBy ?? 0) > 0
        ? `Estimated costs are over budget by ${formatCurrencyAmount(summary.overBudgetBy, summary.currency)}.`
        : `Estimated total is ${formatCurrencyAmount(summary.estimatedTotal, summary.currency)}.`,
    detail:
      summary.missingEstimateCount > 0
        ? `${summary.missingEstimateCount} item(s) still need estimates.`
        : null,
    metrics: [
      { label: "Estimated", value: formatCurrencyAmount(summary.estimatedTotal, summary.currency) },
      { label: "Budget used", value: formatPercent(utilization) },
      { label: "Missing", value: String(summary.missingEstimateCount) }
    ],
    primaryAction: {
      label: (summary.overBudgetBy ?? 0) > 0 ? "Review budget" : "Open Budget",
      href: "#budget"
    }
  };
}

function budgetNavigationBadge(input: TripCommandCenterInput) {
  const confidence = input.budgetConfidence;
  if (confidence?.riskLevel === "critical" || confidence?.riskLevel === "high") {
    return "risk";
  }
  if (confidence?.level === "low" || confidence?.level === "very_low") {
    return "low";
  }
  if (input.budgetSummary?.overBudgetBy) {
    return "over";
  }
  return null;
}

export function buildGroupReadinessCard(input: TripCommandCenterInput): ReadinessCard {
  if (input.groupReadiness) {
    const readiness = input.groupReadiness;
    const readyMembers = readiness.members.filter((member) => member.level === "ready").length;
    const attentionMembers = readiness.members.length - readyMembers;
    const topAction = readiness.topActions[0] ?? null;
    return {
      id: "group",
      title: "Group Readiness",
      status: readiness.level === "not_ready" ? "blocked" : readiness.level,
      score: readiness.score,
      summary: readiness.summary,
      detail:
        topAction?.description ??
        (attentionMembers > 0 ? `${attentionMembers} collaborator(s) need attention.` : null),
      metrics: [
        { label: "Ready", value: `${readyMembers}/${readiness.members.length}` },
        { label: "Attention", value: String(attentionMembers) },
        { label: "Actions", value: String(readiness.topActions.length) }
      ],
      primaryAction: {
        label: topAction?.label ?? "Open Group Readiness",
        href: topAction?.href ?? "#group-readiness"
      },
      secondaryAction: { label: "Open Group Readiness", href: "#group-readiness" }
    };
  }
  const collaborators = input.availability?.summary.totalCollaborators ?? input.trip.travelers;
  if (collaborators <= 1 && !input.trip.workspaceId) {
    return {
      id: "group",
      title: "Group Readiness",
      status: "unavailable",
      summary: "Personal trip with no collaborators.",
      metrics: [],
      primaryAction: { label: "Open Sharing", href: "#sharing" }
    };
  }
  const missingAvailability = input.availability?.summary.missingCount ?? 0;
  const openPolls = (input.polls ?? []).filter((poll) => poll.status === "open").length;
  const assignedIncomplete =
    input.checklist?.checklist?.items.filter(
      (item) => !item.checked && Boolean(item.assignedToUserId)
    ).length ?? 0;
  const status: ReadinessCardStatus =
    missingAvailability > 0 || openPolls > 0
      ? "needs_attention"
      : assignedIncomplete > 0
        ? "almost_ready"
        : "ready";
  return {
    id: "group",
    title: "Group Readiness",
    status,
    summary:
      missingAvailability > 0
        ? `${missingAvailability} collaborator(s) still need to submit availability.`
        : openPolls > 0
          ? `${openPolls} open poll(s) need a decision.`
          : "Group inputs look ready.",
    detail: assignedIncomplete > 0 ? `${assignedIncomplete} assigned checklist item(s) remain.` : null,
    metrics: [
      { label: "Travelers", value: String(collaborators) },
      { label: "Missing availability", value: String(missingAvailability) },
      { label: "Open polls", value: String(openPolls) }
    ],
    primaryAction: {
      label: missingAvailability > 0 ? "Open Dates" : "Open Polls",
      href: missingAvailability > 0 ? "#dates" : "#decisions"
    }
  };
}

export function buildChecklistReminderCard(input: TripCommandCenterInput): ReadinessCard {
  const checklistSummary = input.checklist?.summary;
  const reminderSummary = input.reminders?.summary;
  const totalChecklist = checklistSummary?.totalItems ?? 0;
  const checked = checklistSummary?.checkedItems ?? 0;
  const highPriority = checklistSummary?.highPriorityUnchecked ?? 0;
  const overdue = reminderSummary?.overdue ?? 0;
  const dueToday = reminderSummary?.dueToday ?? 0;

  const status: ReadinessCardStatus =
    highPriority > 0 || overdue > 0
      ? "needs_attention"
      : totalChecklist === 0 && (reminderSummary?.total ?? 0) === 0
        ? "empty"
        : totalChecklist > 0 && checked < totalChecklist
          ? "almost_ready"
          : "ready";

  return {
    id: "checklist_reminders",
    title: "Checklist & Reminders",
    status,
    score: totalChecklist > 0 ? Math.round((checked / totalChecklist) * 100) : null,
    summary:
      status === "empty"
        ? "No checklist or reminders yet."
        : `${checked} of ${totalChecklist} checklist item(s) complete.`,
    detail:
      overdue > 0
        ? `${overdue} overdue reminder(s).`
        : highPriority > 0
          ? `${highPriority} high-priority checklist item(s) remain.`
          : null,
    metrics: [
      { label: "Checklist", value: totalChecklist > 0 ? `${checked}/${totalChecklist}` : "0/0" },
      { label: "High priority", value: String(highPriority) },
      { label: "Due today", value: String(dueToday) }
    ],
    primaryAction: {
      label: highPriority > 0 ? "Complete checklist" : "Open Checklist",
      href: "#checklist"
    },
    secondaryAction: { label: "Open Reminders", href: "#reminders" }
  };
}

export function buildExpenseSettlementCard(input: TripCommandCenterInput): ReadinessCard {
  const summary = input.expenseSummary;
  const pending = summary?.settlementSummary.pendingCount ?? 0;
  const settlementWarnings = input.settlements?.warnings.length ?? 0;
  const status: ReadinessCardStatus =
    pending > 0 && isPastTrip(input)
      ? "needs_attention"
      : pending > 0 || settlementWarnings > 0
        ? "almost_ready"
        : summary
          ? "ready"
          : "empty";
  return {
    id: "expenses_settlements",
    title: "Expenses & Settlements",
    status,
    summary: summary
      ? `Actual spending is ${formatCurrencyAmount(summary.actualTotal.amount, summary.actualTotal.currency)}.`
      : "No actual expenses have been added yet.",
    detail: pending > 0 ? `${pending} pending settlement(s).` : null,
    metrics: [
      {
        label: "Actual",
        value: summary
          ? formatCurrencyAmount(summary.actualTotal.amount, summary.actualTotal.currency)
          : "n/a"
      },
      { label: "Pending settlements", value: String(pending) },
      { label: "Warnings", value: String(settlementWarnings) }
    ],
    primaryAction: { label: "Open Expenses", href: "#expenses" }
  };
}

export function buildApprovalPolicyCard(input: TripCommandCenterInput): ReadinessCard {
  if (!input.trip.workspaceId) {
    return {
      id: "approval_policy",
      title: "Approval & Policy",
      status: "unavailable",
      summary: "No workspace approval applies to this personal trip.",
      metrics: [],
      primaryAction: null
    };
  }
  const approvalStatus = input.approval?.status ?? "draft";
  const blocking = input.policyEvaluation?.summary.blockingCount ?? 0;
  const warnings = input.policyEvaluation?.summary.warningCount ?? 0;
  const risk = input.approvalRisk?.status ?? "unknown";
  const status: ReadinessCardStatus =
    blocking > 0 || approvalStatus === "changes_requested"
      ? "blocked"
      : approvalStatus === "pending_approval" || warnings > 0
        ? "needs_attention"
        : approvalStatus === "approved" || approvalStatus === "not_required"
          ? "ready"
          : "almost_ready";
  return {
    id: "approval_policy",
    title: "Approval & Policy",
    status,
    summary:
      approvalStatus === "changes_requested"
        ? "Approval changes were requested."
        : blocking > 0
          ? `${blocking} blocking policy issue(s).`
          : `Approval status is ${approvalStatus.replaceAll("_", " ")}.`,
    detail: warnings > 0 ? `${warnings} policy warning(s). Risk level: ${risk}.` : `Risk level: ${risk}.`,
    metrics: [
      { label: "Approval", value: approvalStatus.replaceAll("_", " ") },
      { label: "Policy blockers", value: String(blocking) },
      { label: "Risk", value: risk }
    ],
    primaryAction: { label: "Open Approval", href: "#approval" },
    secondaryAction: { label: "Open Policy", href: "#workspace-policy" }
  };
}

export function buildActivityCard(input: TripCommandCenterInput): ReadinessCard {
  const activity = input.activity ?? [];
  return {
    id: "activity",
    title: "Recent Activity",
    status: activity.length > 0 ? "ready" : "empty",
    summary:
      activity.length > 0
        ? `${Math.min(activity.length, 5)} recent update(s) available.`
        : "No recent activity yet.",
    metrics: [{ label: "Events", value: String(activity.length) }],
    primaryAction: { label: "View activity", href: "#activity" }
  };
}

export function buildOfflineStatusCard(input: TripCommandCenterInput): ReadinessCard {
  const offline = input.offlineStatus;
  const status: ReadinessCardStatus =
    offline.failedCount > 0 || offline.conflictCount > 0
      ? "needs_attention"
      : offline.pendingCount > 0
        ? "almost_ready"
        : offline.availableOffline
          ? "ready"
          : "unavailable";
  return {
    id: "offline",
    title: "Offline Status",
    status,
    summary: offline.availableOffline
      ? "This trip has an offline copy on this device."
      : "Offline copy is not enabled on this device.",
    detail:
      offline.pendingCount > 0 || offline.failedCount > 0
        ? `${offline.pendingCount} pending, ${offline.failedCount} failed sync item(s).`
        : offline.cachedAt
          ? `Last saved offline ${new Date(offline.cachedAt).toLocaleString()}.`
          : null,
    metrics: [
      { label: "Network", value: offline.online ? "Online" : "Offline" },
      { label: "Pending", value: String(offline.pendingCount) },
      { label: "Failed", value: String(offline.failedCount) }
    ],
    primaryAction: {
      label: offline.pendingCount > 0 ? "Sync now" : "Open offline panel",
      href: "#offline"
    }
  };
}

function countHealth(input: TripCommandCenterInput, severities: TripHealthIssueSeverity[]) {
  return (input.health?.issues ?? []).filter(
    (issue) => issue.status === "open" && severities.includes(issue.severity)
  ).length;
}

function countHealthCategories(input: TripCommandCenterInput, categories: string[]) {
  return (input.health?.issues ?? []).filter(
    (issue) => issue.status === "open" && categories.includes(issue.category)
  ).length;
}

function isPastTrip(input: TripCommandCenterInput): boolean {
  if (!input.trip.startDate) {
    return false;
  }
  const start = new Date(`${input.trip.startDate}T00:00:00`);
  if (Number.isNaN(start.getTime())) {
    return false;
  }
  const end = new Date(start);
  end.setDate(start.getDate() + Math.max(input.trip.days, 1));
  return end.getTime() < Date.now();
}
