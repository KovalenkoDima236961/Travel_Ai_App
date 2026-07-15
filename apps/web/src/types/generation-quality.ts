export type GenerationQualityStatus =
  | "not_validated"
  | "validated"
  | "validated_with_warnings"
  | "repaired_and_validated"
  | "repaired_with_warnings"
  | "repair_failed"
  | "schema_invalid"
  | "blocked_by_policy"
  | "blocked_by_critical_issues"
  | "ai_output_invalid";

export type GenerationIssueSeverity =
  | "info"
  | "warning"
  | "high"
  | "critical"
  | "blocking";

export type GenerationValidationIssue = {
  id: string;
  code?: string;
  category: string;
  severity: GenerationIssueSeverity;
  title: string;
  description?: string;
  fixability?: string;
  dayNumber?: number | null;
  itemIndex?: number | null;
  routeLegId?: string;
  ruleKey?: string;
};

export type GenerationRepairScope = {
  type: string;
  dayNumber?: number | null;
  itemIndex?: number | null;
  routeLegId?: string;
};

export type GenerationRepairChange = {
  type: string;
  description?: string;
  dayNumber?: number | null;
  itemIndex?: number | null;
  metadata?: Record<string, unknown>;
};

export type GenerationRepairAttempt = {
  attempt: number;
  repairScope: GenerationRepairScope;
  targetIssueIds: string[];
  issuesFixed: string[];
  issuesRemaining: string[];
  durationMs: number;
  aiProviderMode?: string;
  changesMade?: GenerationRepairChange[];
  warnings?: string[];
};

export type GenerationQuality = {
  status: GenerationQualityStatus;
  validatedAt?: string;
  validatorVersion?: string;
  repairAttempts: number;
  maxRepairAttempts?: number;
  blockingIssueCount: number;
  criticalIssueCount: number;
  highIssueCount: number;
  warningIssueCount: number;
  remainingIssues: GenerationValidationIssue[];
  repairedIssues: GenerationValidationIssue[];
  warnings: string[];
  repairAttemptsLog?: GenerationRepairAttempt[];
};

export type GenerationQualityCarrier = {
  generationQuality?: GenerationQuality | null;
  metadata?: { generationQuality?: GenerationQuality | null } | Record<string, unknown> | null;
  resultPayload?: { generationQuality?: GenerationQuality | null } | Record<string, unknown> | null;
};
