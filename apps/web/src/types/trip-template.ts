import type { EstimatedCost } from "@/types/budget";
import type { Trip } from "@/types/trip";

export type TripTemplateVisibility = "private" | "workspace";
export type TripTemplateStatus = "active" | "archived";

export type TripTemplateAccess = {
  role: string;
  source: "private" | "workspace" | string;
  canUse: boolean;
  canEdit: boolean;
  canArchive: boolean;
  canDuplicate: boolean;
};

export type TripTemplate = {
  id: string;
  workspaceId?: string | null;
  createdByUserId: string;
  sourceTripId?: string | null;
  title: string;
  description?: string | null;
  destinationHint?: string | null;
  durationDays: number;
  defaultCurrency?: string | null;
  visibility: TripTemplateVisibility;
  tags: string[];
  estimatedTotalAmount?: number | null;
  estimatedTotalCurrency?: string | null;
  status: TripTemplateStatus;
  access: TripTemplateAccess;
  createdAt: string;
  updatedAt: string;
  archivedAt?: string | null;
};

export type TripTemplatePlace = {
  name: string;
  category?: string | null;
  address?: string | null;
  lat?: number | null;
  lng?: number | null;
  provider?: string | null;
  providerPlaceId?: string | null;
};

export type TripTemplateItem = {
  templateItemId: string;
  name: string;
  type: string;
  description?: string | null;
  time?: string | null;
  startTime?: string | null;
  endTime?: string | null;
  durationMinutes?: number | null;
  place?: TripTemplatePlace | null;
  estimatedCost?: EstimatedCost | null;
  notes?: string | null;
};

export type TripTemplateDay = {
  dayOffset: number;
  title: string;
  items: TripTemplateItem[];
};

export type TripTemplateJSON = {
  schemaVersion: 1;
  durationDays: number;
  days: TripTemplateDay[];
  summary?: {
    estimatedTotalAmount?: number | null;
    currency?: string | null;
  };
  metadata?: Record<string, unknown>;
};

export type TripTemplateDetail = TripTemplate & {
  templateJson: TripTemplateJSON;
};

export type ListTripTemplatesParams = {
  visibility?: TripTemplateVisibility | "all";
  workspaceId?: string | null;
  status?: TripTemplateStatus;
  tag?: string;
  q?: string;
  limit?: number;
  offset?: number;
};

export type ListTripTemplatesResponse = {
  templates: TripTemplate[];
  items?: TripTemplate[];
  limit: number;
  offset: number;
  nextCursor?: string | null;
};

export type SaveTripAsTemplateInput = {
  title: string;
  description?: string | null;
  visibility: TripTemplateVisibility;
  workspaceId?: string | null;
  destinationHint?: string | null;
  defaultCurrency?: string | null;
  tags?: string[];
};

export type UpdateTripTemplateInput = {
  title?: string;
  description?: string | null;
  destinationHint?: string | null;
  defaultCurrency?: string | null;
  tags?: string[];
};

export type DuplicateTripTemplateInput = {
  title?: string;
  visibility: TripTemplateVisibility;
  workspaceId?: string | null;
};

export type CreateTripFromTemplateInput = {
  title: string;
  destination: string;
  startDate: string;
  workspaceId?: string | null;
  budget?: {
    amount?: number | null;
    currency: string;
  } | null;
  travelers?: number | null;
  pace?: "relaxed" | "balanced" | "packed" | string;
};

export type CreateTripFromTemplateResult = Trip;
