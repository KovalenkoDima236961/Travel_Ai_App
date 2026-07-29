import { apiFetch, apiFetchPublic } from "@/shared/api/client";

export type AlphaInvite = {
  id: string;
  code?: string;
  codeDisplay: string;
  expiresAt?: string | null;
  maxActivations: number;
  currentActivations: number;
  creatorUserId: string;
  notes: string;
  testerGroup: "internal" | "external" | "qa" | "design_reviewer";
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
};

export type AlphaWaitlistEntry = {
  id: string;
  email: string;
  emailDomain: string;
  status: "registered" | "invited" | "accepted" | "declined" | "removed";
  invitedInviteId?: string | null;
  source: string;
  notes: string;
  createdAt: string;
  updatedAt: string;
  invitedAt?: string | null;
  acceptedAt?: string | null;
  declinedAt?: string | null;
  removedAt?: string | null;
};

export type AlphaParticipant = {
  userId: string;
  inviteId?: string | null;
  alphaParticipant: boolean;
  testerGroup: string;
  invitationDate?: string | null;
  firstLoginAt?: string | null;
  firstTripAt?: string | null;
  firstAiGenerationAt?: string | null;
  lastActivityAt?: string | null;
  active: boolean;
};

export type AlphaFeedback = {
  id: string;
  userId: string;
  category: AlphaFeedbackCategory;
  title: string;
  descriptionSanitized: string;
  status: "open" | "triaged" | "in_progress" | "resolved" | "closed" | "duplicate";
  priority: "low" | "normal" | "high" | "urgent";
  ownerUserId?: string | null;
  internalNotes?: string;
  metadata: Record<string, unknown>;
  appVersion?: string | null;
  browserFamily?: string | null;
  osFamily?: string | null;
  deviceType?: string | null;
  requestId?: string | null;
  correlationId?: string | null;
  provider?: string | null;
  modelAlias?: string | null;
  promptVersion?: string | null;
  featureFlags: Record<string, unknown>;
  attachmentCount: number;
  createdAt: string;
  updatedAt: string;
};

export type AlphaFeedbackCategory =
  | "ai"
  | "ui"
  | "performance"
  | "bug"
  | "security"
  | "accessibility"
  | "feature_request"
  | "other";

export type AlphaDashboard = {
  generatedAt: string;
  users: { invited: number; active: number; inactive: number; retained: number };
  trips: { created: number; completed: number };
  ai: {
    generations: number;
    successRate: number;
    repairRate: number;
    fallbackRate: number;
    averageLatencyMs: number;
    tokenUsage: number;
    regeneratedItineraries: number;
    removedPlaces: number;
    replacedPlaces: number;
    acceptedItineraries: number;
    badPlaceReports: number;
  };
  feedback: {
    total: number;
    bugReports: number;
    aiReports: number;
    featureRequests: number;
    byCategory: Record<string, number>;
    byStatus: Record<string, number>;
  };
  usage: { dau: number; wau: number; mau: number };
  costs: { openaiTokens: number; estimatedOpenAiCost: number };
  health: { failures: number; retries: number; incidents: number };
  funnel: Array<{
    name: string;
    users: number;
    conversion: number;
    dropoffFromPrevious: number;
  }>;
  featureUsage: Array<{
    feature: string;
    usageCount: number;
    uniqueUsers: number;
    firstUse?: string | null;
    repeatUse: number;
    unused: boolean;
  }>;
  alerts: Array<{ type: string; severity: string; message: string; value: number }>;
};

export type WeeklyAlphaReport = {
  id: string;
  weekStart: string;
  weekEnd: string;
  summaryMarkdown: string;
  metrics: AlphaDashboard;
  generatedByUserId?: string | null;
  generatedAt: string;
};

export type AnalyticsEventInput = {
  eventName: string;
  feature?: string;
  entityType?: string;
  entityId?: string;
  metadata?: Record<string, unknown>;
};

export const alphaKeys = {
  participant: ["alpha", "participant"] as const,
  dashboard: ["ops", "alpha", "dashboard"] as const,
  invites: ["ops", "alpha", "invites"] as const,
  waitlist: (status?: string) => ["ops", "alpha", "waitlist", status ?? "all"] as const,
  feedback: (status?: string, category?: string) =>
    ["ops", "alpha", "feedback", status ?? "all", category ?? "all"] as const,
  weeklyReports: ["ops", "alpha", "weekly-reports"] as const
};

export function joinAlphaWaitlist(email: string) {
  return apiFetchPublic<{ waitlist: AlphaWaitlistEntry }>("/alpha/waitlist", {
    method: "POST",
    body: JSON.stringify({ email, source: "web" })
  });
}

export function activateAlphaInvite(code: string) {
  return apiFetch<{ participant: AlphaParticipant }>("/alpha/activate", {
    method: "POST",
    body: JSON.stringify({ code })
  });
}

export function getAlphaParticipant() {
  return apiFetch<{ participant: AlphaParticipant }>("/alpha/me");
}

export function recordAnalyticsEvent(input: AnalyticsEventInput) {
  const navigatorData = browserContext();
  return apiFetch<{ event: unknown }>("/analytics/events", {
    method: "POST",
    body: JSON.stringify({
      ...navigatorData,
      ...input,
      sessionId: getSessionId(),
      source: "web",
      appVersion: process.env.NEXT_PUBLIC_APP_VERSION || "unknown"
    })
  });
}

export function trackAlphaEvent(input: AnalyticsEventInput) {
  void recordAnalyticsEvent(input).catch(() => undefined);
}

export function submitAlphaFeedback(input: {
  category: AlphaFeedbackCategory;
  title: string;
  description: string;
  metadata?: Record<string, unknown>;
}) {
  return apiFetch<{ feedback: AlphaFeedback }>("/feedback", {
    method: "POST",
    body: JSON.stringify({
      ...browserContext(),
      ...input,
      appVersion: process.env.NEXT_PUBLIC_APP_VERSION || "unknown",
      featureFlags: {}
    })
  });
}

export function getAlphaDashboard() {
  return apiFetch<AlphaDashboard>("/ops/alpha/dashboard");
}

export function listAlphaInvites() {
  return apiFetch<{ invites: AlphaInvite[] }>("/ops/alpha/invites");
}

export function createAlphaInvite(input: {
  testerGroup: AlphaInvite["testerGroup"];
  maxActivations: number;
  expiresAt?: string | null;
  notes?: string;
}) {
  return apiFetch<{ invite: AlphaInvite }>("/ops/alpha/invites", {
    method: "POST",
    body: JSON.stringify({ ...input, expiresAt: normalizeDateInput(input.expiresAt) })
  });
}

export function updateAlphaInvite(
  inviteId: string,
  input: Partial<Pick<AlphaInvite, "enabled" | "notes" | "testerGroup" | "maxActivations">> & {
    expiresAt?: string | null;
  }
) {
  return apiFetch<{ invite: AlphaInvite }>(`/ops/alpha/invites/${encodeURIComponent(inviteId)}`, {
    method: "PATCH",
    body: JSON.stringify({ ...input, expiresAt: normalizeDateInput(input.expiresAt) })
  });
}

export function listAlphaWaitlist(status?: string) {
  const query = status ? `?status=${encodeURIComponent(status)}` : "";
  return apiFetch<{ waitlist: AlphaWaitlistEntry[] }>(`/ops/alpha/waitlist${query}`);
}

export function inviteFromWaitlist(input: {
  waitlistId: string;
  testerGroup: AlphaInvite["testerGroup"];
  notes?: string;
  expiresAt?: string | null;
}) {
  return apiFetch<{ invite: AlphaInvite; waitlist: AlphaWaitlistEntry }>(
    "/ops/alpha/invite-from-waitlist",
    { method: "POST", body: JSON.stringify({ ...input, expiresAt: normalizeDateInput(input.expiresAt) }) }
  );
}

export function listAlphaFeedback(status?: string, category?: string) {
  const params = new URLSearchParams();
  if (status) params.set("status", status);
  if (category) params.set("category", category);
  const query = params.toString();
  return apiFetch<{ feedback: AlphaFeedback[] }>(
    `/ops/alpha/feedback${query ? `?${query}` : ""}`
  );
}

export function updateAlphaFeedback(
  feedbackId: string,
  input: Partial<Pick<AlphaFeedback, "status" | "priority" | "ownerUserId" | "internalNotes">>
) {
  return apiFetch<{ feedback: AlphaFeedback }>(
    `/ops/alpha/feedback/${encodeURIComponent(feedbackId)}`,
    { method: "PATCH", body: JSON.stringify(input) }
  );
}

export function listWeeklyAlphaReports() {
  return apiFetch<{ reports: WeeklyAlphaReport[] }>("/ops/alpha/reports/weekly");
}

export function generateWeeklyAlphaReport(weekStart?: string) {
  return apiFetch<{ report: WeeklyAlphaReport }>("/ops/alpha/reports/weekly/generate", {
    method: "POST",
    body: JSON.stringify({ weekStart })
  });
}

function browserContext() {
  if (typeof window === "undefined") {
    return {};
  }
  const ua = window.navigator.userAgent;
  return {
    browserFamily: browserFamily(ua),
    osFamily: osFamily(ua),
    deviceType: /Mobi|Android|iPhone/i.test(ua) ? "mobile" : "desktop"
  };
}

function browserFamily(ua: string) {
  if (/Edg\//.test(ua)) return "edge";
  if (/Chrome\//.test(ua)) return "chrome";
  if (/Safari\//.test(ua)) return "safari";
  if (/Firefox\//.test(ua)) return "firefox";
  return "other";
}

function osFamily(ua: string) {
  if (/Windows/i.test(ua)) return "windows";
  if (/Mac OS X/i.test(ua)) return "macos";
  if (/Android/i.test(ua)) return "android";
  if (/iPhone|iPad/i.test(ua)) return "ios";
  if (/Linux/i.test(ua)) return "linux";
  return "other";
}

function getSessionId() {
  if (typeof window === "undefined") {
    return "server";
  }
  const key = "travel_ai_alpha_session";
  let value = window.sessionStorage.getItem(key);
  if (!value) {
    value =
      window.crypto?.randomUUID?.() ??
      `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
    window.sessionStorage.setItem(key, value);
  }
  return value;
}

function normalizeDateInput(value: string | null | undefined) {
  if (value == null || value === "") {
    return value ?? null;
  }
  if (/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    return new Date(`${value}T23:59:59.000Z`).toISOString();
  }
  return value;
}
