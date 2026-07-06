import type { GenerationJob, GenerationJobStatus } from "@/types/generation-jobs";

export type TemplateAdaptationJobStatus = GenerationJobStatus;

/** The adaptation job is a `template_adaptation` generation job. Its `tripId` is
 * the draft trip created up front (available immediately), and its
 * `resultPayload` carries the adaptation summary once completed. */
export type TemplateAdaptationJob = GenerationJob & {
  resultPayload?: TemplateAdaptationSummary | null;
};

export type TemplateAdaptationTarget = {
  destination: string;
  startDate: string;
  durationDays: number;
  budget?: { amount: number; currency: string } | null;
  travelers?: number | null;
  pace?: string | null;
  interests?: string[];
  avoid?: string[];
};

export type TemplateAdaptationInput = TemplateAdaptationTarget & {
  title: string;
  workspaceId?: string | null;
  specialInstructions?: string | null;
  fallbackToDeterministic?: boolean;
};

export type TemplateAdaptationSummary = {
  sourceDurationDays: number;
  targetDurationDays: number;
  preservedStructure: boolean;
  changedDestination: boolean;
  fallbackUsed: boolean;
  fallbackReason?: string;
  majorChanges: string[];
  warnings: string[];
};
