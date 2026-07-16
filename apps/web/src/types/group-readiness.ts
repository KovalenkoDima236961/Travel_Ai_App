export type ReadinessLevel = "ready" | "almost_ready" | "needs_attention" | "not_ready";

export type ReadinessCategory =
  | "availability"
  | "calendar"
  | "polls"
  | "checklist"
  | "reminders"
  | "expenses"
  | "settlements"
  | "comments"
  | "activity"
  | "approval"
  | "offline_sync"
  | "profile"
  | "other";

export type ReadinessItemStatus =
  | "complete"
  | "incomplete"
  | "missing"
  | "overdue"
  | "pending"
  | "not_applicable"
  | "unknown";

export type ReadinessIssueSeverity = "info" | "warning" | "high" | "critical";

export type ReadinessAction = {
  type: string;
  label: string;
  href: string;
};

export type ReadinessItem = {
  id: string;
  category: ReadinessCategory;
  status: ReadinessItemStatus;
  severity: ReadinessIssueSeverity;
  title: string;
  description: string;
  action?: ReadinessAction | null;
};

export type CompletedReadinessItem = {
  category: ReadinessCategory;
  title: string;
};

export type CollaboratorReadiness = {
  userId: string;
  displayName: string;
  role: string;
  status: string;
  score: number;
  level: ReadinessLevel;
  isCurrentUser: boolean;
  items: ReadinessItem[];
  completedItems: CompletedReadinessItem[];
  nextAction?: ReadinessAction | null;
};

export type GroupReadinessCategorySummary = {
  category: ReadinessCategory;
  readyCount: number;
  totalCount: number;
  openIssueCount: number;
  highestSeverity?: ReadinessIssueSeverity | "";
};

export type GroupReadinessTopAction = {
  id: string;
  label: string;
  description: string;
  href: string;
  actionType: string;
  targetUserId?: string | null;
};

export type GroupReadiness = {
  tripId: string;
  score: number;
  level: ReadinessLevel;
  summary: string;
  generatedAt: string;
  members: CollaboratorReadiness[];
  categorySummary: GroupReadinessCategorySummary[];
  topActions: GroupReadinessTopAction[];
  debug?: Record<string, unknown>;
};

export type NudgeRequest = {
  targetUserIds: string[];
  categories: ReadinessCategory[];
  message?: string;
  dedupeWindowHours?: number;
};

export type NudgeResponse = {
  sentCount: number;
  skippedCount: number;
  dedupedCount: number;
  targetUserIds: string[];
  categories: ReadinessCategory[];
  dedupeWindowHours: number;
};

