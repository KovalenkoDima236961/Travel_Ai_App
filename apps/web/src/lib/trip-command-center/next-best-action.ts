import type {
  NextBestAction,
  TripCommandCenterAccess,
  TripCommandCenterInput
} from "@/types/trip-command-center";
import type { TripHealthCategory, TripHealthIssue } from "@/types/trip-health";
import type { TripRouteLeg } from "@/entities/route/model";

type Capability = "view" | "collaborate" | "edit";

type ActionCandidate = NextBestAction & {
  capability: Capability;
  priority: number;
};

const SEVERITY_BY_STATUS = {
  blocking: "critical",
  warning: "warning",
  info: "info"
} as const;

export function selectNextBestAction(input: TripCommandCenterInput): NextBestAction {
  const candidates = buildActionCandidates(input).sort((left, right) => left.priority - right.priority);
  const allowed = candidates.find((candidate) => canPerform(candidate.capability, input.userAccess));
  const selected = allowed ?? candidates[0] ?? readyAction(input.trip.id);
  if (selected.capability === "edit" && !input.userAccess.canEdit) {
    return {
      ...toPublicAction(selected),
      actionLabel: "View details",
      viewOnly: true
    };
  }
  return toPublicAction(selected);
}

export function buildTopFixActions(input: TripCommandCenterInput, limit = 5): NextBestAction[] {
  const issuesById = new Map((input.health?.issues ?? []).map((issue) => [issue.id, issue]));
  return (input.health?.topFixes ?? [])
    .slice(0, limit)
    .map((fix, index) => {
      const issue = issuesById.get(fix.issueId);
      return toPublicAction({
        id: fix.issueId,
        title: fix.label,
        description: issue?.description ?? "Review this trip readiness issue.",
        reason: issue?.recommendation ?? "Top Trip Health fix",
        severity: issue?.severity ?? "warning",
        category: issue?.category ?? "other",
        actionLabel: issue?.action?.label ?? "Open issue",
        href: fix.href,
        source: "trip_health",
        capability: inferCapability(issue?.category, fix.href),
        priority: index
      });
    });
}

function buildActionCandidates(input: TripCommandCenterInput): ActionCandidate[] {
  const candidates: ActionCandidate[] = [];
  const trip = input.trip;
  const tripId = trip.id;
  const criticalHealthIssue = findHealthIssue(input.health?.issues ?? [], ["critical"]);
  const highHealthIssue = findHealthIssue(input.health?.issues ?? [], ["high"]);

  if (criticalHealthIssue) {
    candidates.push(actionFromHealthIssue(criticalHealthIssue, 1));
  }

  const blockingPolicy = input.policyEvaluation?.results.find(
    (result) => result.status === "violation" && result.severity === "blocking"
  );
  if (blockingPolicy) {
    candidates.push({
      id: `policy_blocking:${blockingPolicy.ruleKey}`,
      title: blockingPolicy.title,
      description: blockingPolicy.message,
      reason: "Blocking workspace policy violation",
      severity: "critical",
      category: "policy",
      actionLabel: "Open policy",
      href: `/trips/${tripId}?tab=policy`,
      source: "policy",
      capability: "edit",
      priority: 2
    });
  }

  if (input.approval?.status === "changes_requested") {
    candidates.push({
      id: "approval_changes_requested",
      title: "Approval changes requested",
      description: input.approval.decisionNote ?? "Review the requested changes before resubmitting.",
      reason: "Workspace approval is blocked",
      severity: "high",
      category: "approval",
      actionLabel: "Open approval",
      href: `/trips/${tripId}?tab=approval`,
      source: "approval",
      capability: "edit",
      priority: 3
    });
  }

  const groupAction = input.groupReadiness?.topActions[0] ?? null;
  if (groupAction && input.groupReadiness && input.groupReadiness.level !== "ready") {
    candidates.push({
      id: `group_readiness:${groupAction.id}`,
      title: groupAction.label,
      description: groupAction.description,
      reason: "Group readiness needs attention",
      severity: input.groupReadiness.level === "not_ready" ? "high" : "warning",
      category: "group",
      actionLabel: groupAction.label,
      href: groupAction.href,
      source: "group",
      capability: groupAction.targetUserId ? "edit" : "collaborate",
      priority: 4
    });
  }

  if (
    !trip.startDate &&
    (input.availability?.summary.totalCollaborators ?? 0) > 1
  ) {
    candidates.push({
      id: "group_dates_missing",
      title: "Choose dates for this group trip",
      description: "Collaborator availability exists, but the trip still needs selected dates.",
      reason: "Missing selected dates for group trip",
      severity: "high",
      category: "availability",
      actionLabel: "Open dates",
      href: `/trips/${tripId}?tab=dates`,
      source: "group",
      capability: "edit",
      priority: 5
    });
  }

  if (!trip.itinerary || trip.status === "DRAFT") {
    candidates.push({
      id: "itinerary_missing",
      title: "Generate the itinerary",
      description: "The trip has no day-by-day itinerary yet.",
      reason: "Missing itinerary",
      severity: "high",
      category: "itinerary",
      actionLabel: "Generate itinerary",
      href: `/trips/${tripId}?tab=itinerary`,
      source: "trip",
      capability: "edit",
      priority: 6
    });
  }

  const failedGenerationJob = (input.generationJobs ?? []).find((job) => job.status === "failed");
  if (trip.status === "FAILED" || failedGenerationJob || highHealthIssue?.category === "data_quality") {
    candidates.push({
      id: failedGenerationJob ? `generation_failed:${failedGenerationJob.id}` : "generation_failed",
      title: "Review generation quality",
      description:
        failedGenerationJob?.errorMessage ??
        highHealthIssue?.description ??
        "The AI-generated plan needs review before the trip is ready.",
      reason: "AI generation failed or quality issue",
      severity: "high",
      category: "data_quality",
      actionLabel: "Open health",
      href: `/trips/${tripId}?tab=health`,
      source: "trip_health",
      capability: "edit",
      priority: 7
    });
  }

  if (
    trip.tripType === "multi_destination" &&
    (!trip.route || (trip.route.stops?.length ?? 0) < 2)
  ) {
    candidates.push({
      id: "route_missing",
      title: "Add the multi-destination route",
      description: "This multi-destination trip needs route stops before transport can be checked.",
      reason: "Missing route for multi-destination trip",
      severity: "high",
      category: "route",
      actionLabel: "Open route",
      href: `/trips/${tripId}?tab=route`,
      source: "route",
      capability: "edit",
      priority: 8
    });
  }

  const missingTransportLeg = firstMissingTransportLeg(trip.route?.legs ?? []);
  if (missingTransportLeg) {
    candidates.push({
      id: `transport_missing_option:${missingTransportLeg.id}`,
      title: `Find transport for ${legTitle(missingTransportLeg)}`,
      description:
        "This route leg has no selected transport option, so itinerary timing and budget may be inaccurate.",
      reason: "Missing selected transport option",
      severity: "high",
      category: "transport",
      actionLabel: "Find transport",
      href: `/trips/${tripId}?tab=route&legId=${encodeURIComponent(missingTransportLeg.id)}`,
      source: "route",
      capability: "edit",
      priority: 9
    });
  }

  const budgetConfidenceIssue = topBudgetConfidenceIssue(input);
  if (budgetConfidenceIssue) {
    candidates.push({
      id: `budget_confidence:${budgetConfidenceIssue.id}`,
      title: budgetConfidenceIssue.title,
      description: budgetConfidenceIssue.description,
      reason: budgetConfidenceIssue.recommendation || "Budget confidence needs review",
      severity:
        budgetConfidenceIssue.severity === "critical"
          ? "critical"
          : budgetConfidenceIssue.severity === "info"
            ? "info"
            : budgetConfidenceIssue.severity,
      category: "budget",
      actionLabel: budgetConfidenceIssue.action?.label ?? "Review budget confidence",
      href: `/trips/${tripId}?tab=budget`,
      source: "budget",
      capability: "edit",
      priority: 10
    });
  }

  if ((input.budgetSummary?.overBudgetBy ?? 0) > 0) {
    candidates.push({
      id: "budget_exceeded",
      title: "Review budget overrun",
      description: `Estimated costs exceed the trip budget by ${input.budgetSummary?.overBudgetBy ?? 0} ${input.budgetSummary?.currency ?? trip.budgetCurrency}.`,
      reason: "Budget exceeded",
      severity: "high",
      category: "budget",
      actionLabel: "Review budget",
      href: `/trips/${tripId}?tab=budget`,
      source: "budget",
      capability: "edit",
      priority: 10
    });
  }

  if (trip.days > 1 && !trip.accommodation) {
    candidates.push({
      id: "accommodation_missing",
      title: "Add accommodation details",
      description: "Overnight trips need a stay estimate to keep timing, reminders, and budget accurate.",
      reason: "Missing accommodation for overnight trip",
      severity: "warning",
      category: "accommodation",
      actionLabel: "Open trip tools",
      href: `/trips/${tripId}?tab=budget`,
      source: "trip",
      capability: "edit",
      priority: 10
    });
  }

  if ((input.checklist?.summary?.highPriorityUnchecked ?? 0) > 0) {
    candidates.push({
      id: "checklist_high_priority_incomplete",
      title: "Complete high-priority checklist items",
      description: `${input.checklist?.summary?.highPriorityUnchecked ?? 0} important checklist item(s) still need attention.`,
      reason: "High-priority checklist item incomplete",
      severity: "warning",
      category: "checklist",
      actionLabel: "Open checklist",
      href: `/trips/${tripId}?tab=checklist`,
      source: "checklist",
      capability: "collaborate",
      priority: 11
    });
  }

  if ((input.reminders?.summary.overdue ?? 0) > 0) {
    candidates.push({
      id: "reminders_overdue",
      title: "Clear overdue reminders",
      description: `${input.reminders?.summary.overdue ?? 0} reminder(s) are overdue.`,
      reason: "Reminder overdue",
      severity: "warning",
      category: "reminders",
      actionLabel: "Open reminders",
      href: `/trips/${tripId}?tab=reminders`,
      source: "reminders",
      capability: "collaborate",
      priority: 12
    });
  }

  if ((input.availability?.summary.missingCount ?? 0) > 0) {
    candidates.push({
      id: "availability_missing",
      title: "Request missing availability",
      description: `${input.availability?.summary.missingCount ?? 0} collaborator(s) have not submitted availability.`,
      reason: "Collaborator availability missing",
      severity: "warning",
      category: "availability",
      actionLabel: input.userAccess.canEdit ? "Request availability" : "Submit availability",
      href: `/trips/${tripId}?tab=dates`,
      source: "group",
      capability: "collaborate",
      priority: 13
    });
  }

  const openPoll = (input.polls ?? []).find((poll) => poll.status === "open");
  if (openPoll) {
    candidates.push({
      id: `poll_open:${openPoll.id}`,
      title: openPoll.canVote ? "Vote on the open poll" : "Review open group decision",
      description: openPoll.title,
      reason: "Pending group decision",
      severity: "info",
      category: "collaboration",
      actionLabel: "Open polls",
      href: `/trips/${tripId}?tab=polls`,
      source: "group",
      capability: openPoll.canVote ? "collaborate" : "view",
      priority: 14
    });
  }

  if (isPastTrip(trip) && (input.expenseSummary?.settlementSummary.pendingCount ?? 0) > 0) {
    candidates.push({
      id: "settlements_pending",
      title: "Settle outstanding expenses",
      description: `${input.expenseSummary?.settlementSummary.pendingCount ?? 0} settlement(s) are still pending.`,
      reason: "Expenses unsettled after trip",
      severity: "warning",
      category: "expenses",
      actionLabel: "Open expenses",
      href: `/trips/${tripId}?tab=expenses`,
      source: "expenses",
      capability: "collaborate",
      priority: 15
    });
  }

  if (input.offlineStatus.failedCount > 0 || input.offlineStatus.pendingCount > 0) {
    candidates.push({
      id: input.offlineStatus.failedCount > 0 ? "offline_sync_failed" : "offline_sync_pending",
      title: input.offlineStatus.failedCount > 0 ? "Resolve failed offline sync" : "Sync offline changes",
      description:
        input.offlineStatus.failedCount > 0
          ? `${input.offlineStatus.failedCount} offline change(s) failed to sync.`
          : `${input.offlineStatus.pendingCount} offline change(s) are waiting to sync.`,
      reason: "Offline sync needs attention",
      severity: input.offlineStatus.failedCount > 0 ? "high" : "info",
      category: "offline",
      actionLabel: "Open offline status",
      href: `/trips/${tripId}?tab=offline`,
      source: "offline",
      capability: "collaborate",
      priority: 16
    });
  }

  candidates.push({
    ...readyAction(tripId),
    capability: "view",
    priority: 17
  });

  return candidates;
}

function topBudgetConfidenceIssue(input: TripCommandCenterInput) {
  const confidence = input.budgetConfidence;
  if (!confidence) {
    return null;
  }
  const severe = confidence.issues.find(
    (issue) => issue.severity === "critical" || issue.severity === "high"
  );
  if (severe) {
    return severe;
  }
  if (
    confidence.riskLevel === "critical" ||
    confidence.riskLevel === "high" ||
    confidence.level === "low" ||
    confidence.level === "very_low"
  ) {
    return confidence.issues[0] ?? {
      id: "low_score",
      title: "Budget confidence is low",
      description: confidence.summary,
      recommendation: "Review budget confidence before approval or booking.",
      severity: confidence.riskLevel === "critical" ? "critical" : "high",
      category: "budget"
    };
  }
  return null;
}

function actionFromHealthIssue(issue: TripHealthIssue, priority: number): ActionCandidate {
  return {
    id: issue.id,
    title: issue.title,
    description: issue.description,
    reason: issue.recommendation ?? "High-priority Trip Health issue",
    severity: issue.severity,
    category: issue.category,
    actionLabel: issue.action?.label ?? "View all issues",
    href: issue.action?.href ?? `?tab=health`,
    source: "trip_health",
    capability: inferCapability(issue.category, issue.action?.href),
    priority
  };
}

function findHealthIssue(issues: TripHealthIssue[], severities: TripHealthIssue["severity"][]) {
  return issues.find((issue) => issue.status === "open" && severities.includes(issue.severity));
}

function inferCapability(category?: TripHealthCategory, href?: string | null): Capability {
  if (!category && !href) {
    return "view";
  }
  if (category === "availability" || category === "checklist" || category === "reminders" || category === "expenses" || category === "collaboration" || category === "offline") {
    return "collaborate";
  }
  if (category === "route" || category === "transport" || category === "budget" || category === "accommodation" || category === "policy" || category === "approval" || category === "itinerary" || category === "data_quality") {
    return "edit";
  }
  if (href?.includes("tab=route") || href?.includes("tab=budget") || href?.includes("tab=approval") || href?.includes("tab=policy")) {
    return "edit";
  }
  return "view";
}

function canPerform(capability: Capability, access: TripCommandCenterAccess): boolean {
  if (capability === "view") {
    return access.canView;
  }
  if (capability === "collaborate") {
    return access.canCollaborate || access.canEdit;
  }
  return access.canEdit;
}

function firstMissingTransportLeg(legs: TripRouteLeg[]): TripRouteLeg | null {
  return (
    legs.find((leg) => {
      const selected = leg.selectedTransportOption;
      return (
        !selected ||
        selected.provider === "mock" ||
        selected.confidence === "low" ||
        selected.status === "unknown" ||
        selected.status === "unavailable"
      );
    }) ?? null
  );
}

function legTitle(leg: TripRouteLeg): string {
  const from = leg.fromName || "origin";
  const to = leg.toName || "destination";
  return `${from} -> ${to}`;
}

function readyAction(tripId: string): ActionCandidate {
  return {
    id: "trip_ready",
    title: "Trip looks ready",
    description: "No critical issues found. Review the itinerary or share the trip with collaborators.",
    reason: "No urgent actions",
    severity: "info",
    category: "itinerary",
    actionLabel: "Open itinerary",
    href: `/trips/${tripId}?tab=itinerary`,
    source: "trip_health",
    capability: "view",
    priority: 17
  };
}

function isPastTrip(input: TripCommandCenterInput["trip"]): boolean {
  if (!input.startDate) {
    return false;
  }
  const start = new Date(`${input.startDate}T00:00:00`);
  if (Number.isNaN(start.getTime())) {
    return false;
  }
  const end = new Date(start);
  end.setDate(start.getDate() + Math.max(input.days, 1));
  return end.getTime() < Date.now();
}

function toPublicAction(candidate: ActionCandidate): NextBestAction {
  const { capability: _capability, priority: _priority, ...action } = candidate;
  return action;
}

export const tripCommandCenterTestInternals = {
  buildActionCandidates
};
