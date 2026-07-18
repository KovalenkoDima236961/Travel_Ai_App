import type { TripAccommodation } from "@/entities/accommodation/model";
import type { BudgetSummary } from "@/entities/budget/model";
import type { ChecklistItemPayload, ChecklistViewResponse } from "@/entities/checklist/model";
import type {
  CreateExpenseInput,
  ExpenseSummary,
  SettlementsResponse,
  TripExpense,
  TripExpensesResponse,
  UpdateExpenseInput
} from "@/entities/expense/model";
import type {
  CreateReminderInput,
  ReminderSummary,
  ReminderViewResponse,
  TripReminder
} from "@/entities/trip-reminder/model";
import type { Itinerary, Trip } from "@/entities/trip/model";
import type { TravelDaySummary } from "@/types/travel-day";

export type OfflineMutationStatus =
  | "pending"
  | "syncing"
  | "conflict"
  | "failed"
  | "synced"
  | "cancelled"
  | "discarded";

export type OfflineCompanionMutationType =
  | "checklist_item_check"
  | "checklist_item_uncheck"
  | "checklist_item_create"
  | "checklist_item_update"
  | "checklist_item_delete_local"
  | "reminder_complete"
  | "reminder_reopen"
  | "reminder_create"
  | "reminder_disable"
  | "expense_create"
  | "expense_update_local"
  | "expense_delete_local"
  | "receipt_upload";

export type OfflineMutationType = "update_itinerary" | OfflineCompanionMutationType;

export type OfflineMutationEntity =
  | "itinerary"
  | "checklist"
  | "reminder"
  | "expense"
  | "receipt";

export type CachedTripRecord = {
  cacheKey: string;
  tripId: string;
  userId: string;
  trip: Trip;
  tripSummary?: Trip;
  budgetSummary?: BudgetSummary | null;
  accommodation?: TripAccommodation | null;
  itineraryRevision: number;
  routeRevision?: number | null;
  cachedAt: string;
  lastOpenedAt?: string;
  offlineEnabled?: boolean;
};

export type PendingItineraryMutation = {
  mutationId: string;
  id?: string;
  type: "update_itinerary";
  entity?: "itinerary";
  tripId: string;
  userId: string;
  baseRevision: number;
  baseItinerary: Itinerary;
  draftItinerary: Itinerary;
  status: OfflineMutationStatus;
  createdAt: string;
  updatedAt: string;
  lastAttemptAt?: string | null;
  attemptCount?: number;
  errorCode?: string | null;
  errorMessage?: string | null;
  error?: string | null;
  clientMutationId?: string;
  createdOfflineAt?: string;
  dependsOn?: string[];
};

export type OfflineMutationPayloadByType = {
  checklist_item_check: { itemId: string; checkedAt: string };
  checklist_item_uncheck: { itemId: string; uncheckedAt: string };
  checklist_item_create: {
    localEntityId: string;
    input: ChecklistItemPayload;
  };
  checklist_item_update: {
    localEntityId: string;
    input: ChecklistItemPayload;
  };
  checklist_item_delete_local: {
    localEntityId: string;
  };
  reminder_complete: { reminderId: string; completedAt: string };
  reminder_reopen: { reminderId: string; reopenedAt: string };
  reminder_create: {
    localEntityId: string;
    input: CreateReminderInput;
  };
  reminder_disable: { reminderId: string; disabledAt: string };
  expense_create: {
    localEntityId: string;
    input: CreateExpenseInput;
  };
  expense_update_local: {
    localEntityId: string;
    input: UpdateExpenseInput;
  };
  expense_delete_local: {
    localEntityId: string;
  };
  receipt_upload: {
    receiptDraftId: string;
    linkedExpenseLocalId?: string | null;
    linkedExpenseId?: string | null;
  };
};

type PendingCompanionMutationFields<TType extends OfflineCompanionMutationType> = {
  mutationId: string;
  id: string;
  tripId: string;
  userId: string;
  type: TType;
  entity: OfflineMutationEntity;
  status: OfflineMutationStatus;
  payload: OfflineMutationPayloadByType[TType];
  clientMutationId: string;
  createdOfflineAt: string;
  createdAt: string;
  updatedAt: string;
  lastAttemptAt?: string | null;
  attemptCount: number;
  error?: string | null;
  errorCode?: string | null;
  errorMessage?: string | null;
  dependsOn: string[];
  localEntityId?: string | null;
  entityId?: string | null;
  requestHash?: string | null;
};

export type PendingCompanionMutation<
  TType extends OfflineCompanionMutationType = OfflineCompanionMutationType
> = TType extends OfflineCompanionMutationType
  ? PendingCompanionMutationFields<TType>
  : never;

export type PendingOfflineMutation = PendingItineraryMutation | PendingCompanionMutation;

export type SyncMetadataRecord = {
  key: string;
  userId?: string | null;
  value: unknown;
  updatedAt: string;
};

export type CachedTripDetailsRecord = {
  cacheKey: string;
  tripId: string;
  userId: string;
  trip: Trip;
  itinerary?: Itinerary | null;
  route?: Trip["route"] | null;
  accommodation?: TripAccommodation | null;
  itineraryRevision: number;
  cachedAt: string;
  source: "trip_detail" | "manual";
};

export type CachedTravelDayRecord = {
  cacheKey: string;
  tripId: string;
  userId: string;
  date: string;
  summary: TravelDaySummary;
  itineraryRevision: number;
  cachedAt: string;
};

export type CachedChecklistRecord = {
  cacheKey: string;
  tripId: string;
  userId: string;
  checklist: ChecklistViewResponse;
  cachedAt: string;
  localVersion: number;
};

export type CachedRemindersRecord = {
  cacheKey: string;
  tripId: string;
  userId: string;
  reminders: TripReminder[];
  summary: ReminderSummary;
  cachedAt: string;
  localVersion: number;
};

export type CachedExpensesRecord = {
  cacheKey: string;
  tripId: string;
  userId: string;
  expenses: TripExpense[];
  cachedAt: string;
  localVersion: number;
};

export type CachedExpenseSummaryRecord = {
  cacheKey: string;
  tripId: string;
  userId: string;
  summary: ExpenseSummary;
  cachedAt: string;
};

export type CachedSettlementsRecord = {
  cacheKey: string;
  tripId: string;
  userId: string;
  settlements: SettlementsResponse;
  cachedAt: string;
};

export type OfflineReceiptDraftRecord = {
  id: string;
  tripId: string;
  userId: string;
  fileBlob: Blob;
  filename: string;
  contentType: string;
  sizeBytes: number;
  linkedExpenseLocalId?: string | null;
  linkedExpenseId?: string | null;
  createdOfflineAt: string;
  status: "pending_upload" | "uploading" | "uploaded" | "failed" | "cancelled";
  error?: string | null;
};

export type SyncLogRecord = {
  id: string;
  tripId?: string | null;
  userId: string;
  eventType: string;
  message: string;
  createdAt: string;
  metadata?: Record<string, unknown>;
};

export type OfflineSettingsRecord = {
  userId: string;
  autoCacheOpenedTrips: boolean;
  cacheReceiptsOffline: boolean;
  maxCachedTrips: number;
  lastCleanupAt?: string | null;
};

export type EnqueueItineraryUpdateInput = {
  tripId: string;
  userId: string;
  baseRevision: number;
  baseItinerary: Itinerary;
  draftItinerary: Itinerary;
};

export type SyncResult =
  | {
      status: "synced";
      mutation: PendingOfflineMutation;
      trip?: Trip;
      entity?: unknown;
    }
  | {
      status: "conflict";
      mutation: PendingOfflineMutation;
      currentItineraryRevision?: number | null;
      latestTrip?: Trip | null;
      errorMessage?: string | null;
    }
  | {
      status: "failed";
      mutation: PendingOfflineMutation;
      retryable: boolean;
      errorCode?: string | null;
      errorMessage?: string | null;
    };

export function isPendingItineraryMutation(
  mutation: PendingOfflineMutation | null | undefined
): mutation is PendingItineraryMutation {
  return mutation?.type === "update_itinerary";
}

export function isPendingCompanionMutation(
  mutation: PendingOfflineMutation | null | undefined
): mutation is PendingCompanionMutation {
  return Boolean(mutation && mutation.type !== "update_itinerary");
}

export function toReminderViewResponse(record: CachedRemindersRecord): ReminderViewResponse {
  return {
    reminders: record.reminders,
    summary: record.summary
  };
}

export function toTripExpensesResponse(record: CachedExpensesRecord): TripExpensesResponse {
  return {
    items: record.expenses
  };
}
