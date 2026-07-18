export type RecapStatus = "draft" | "generated" | "edited" | "finalized" | "archived";

export type RecapMoney = { amount: number; currency: string };

export type LearningCandidate = {
  feedbackType: string;
  label: string;
  entityType?: string;
  entityId?: string;
  value?: string;
  metadata?: Record<string, unknown>;
  approved: boolean;
};

export type TripRecapContent = {
  schemaVersion: "trip_recap_v1" | string;
  title: string;
  summary: string;
  highlights: Array<{ title: string; description?: string; dayNumber?: number; itemId?: string }>;
  plannedVsActual: {
    plannedItemCount: number;
    doneItemCount: number;
    skippedItemCount: number;
    delayedItemCount: number;
    unknownItemCount: number;
    completionRate: number;
    notes?: string;
    skippedItems: string[];
    delayedItems: string[];
  };
  budget: {
    plannedTotal?: RecapMoney;
    actualTotal?: RecapMoney;
    varianceAmount?: RecapMoney;
    variancePercent?: number;
    receiptCoveragePercent: number;
    topCategories: Array<{ category: string; total: RecapMoney }>;
    notes?: string;
  };
  routeAndTransport: { summary?: string; issues: string[]; successfulModes: string[]; problemModes: string[] };
  verification: { summary?: string; issues: string[] };
  checklistAndReminders: {
    completedChecklistItems: number;
    totalChecklistItems: number;
    completedReminders: number;
    totalReminders: number;
    notes?: string;
  };
  lessonsLearned: string[];
  futurePreferences: LearningCandidate[];
  templateSuggestion: { recommended: boolean; title?: string; reason?: string };
  userEditableNotes: string;
};

export type TripRecap = {
  id: string;
  tripId: string;
  status: RecapStatus;
  recap: TripRecapContent;
  finalizedAt?: string;
  createdAt: string;
  updatedAt: string;
};

export type TripRecapPermissions = {
  canEdit: boolean;
  canFinalize: boolean;
  canCreateTemplate: boolean;
  canApplyLearning: boolean;
};

export type TripRecapFeedback = {
  id: string;
  feedbackType: string;
  entityType?: string;
  entityId?: string;
  label: string;
  value?: string;
  approvedForPersonalization: boolean;
  metadata: Record<string, unknown>;
  createdAt: string;
};

export type TripRecapStatusResponse = {
  eligible: boolean;
  reason: string;
  hasRecap: boolean;
  recapId?: string;
  tripEndedAt?: string;
  canGenerate: boolean;
  canEdit: boolean;
};

export type TripRecapResponse = {
  recap: TripRecap;
  permissions: TripRecapPermissions;
  feedback: TripRecapFeedback[];
};
