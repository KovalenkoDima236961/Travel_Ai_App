import type {
  ChecklistItemPayload,
  ChecklistSummary,
  ChecklistViewResponse,
  TripChecklist,
  TripChecklistItem
} from "@/entities/checklist/model";
import type {
  CreateExpenseInput,
  ExpenseParticipant,
  ExpenseSummary,
  MoneyAmount,
  TripExpense
} from "@/entities/expense/model";
import type {
  CreateReminderInput,
  ReminderSummary,
  TripReminder
} from "@/entities/trip-reminder/model";
import {
  createOfflineId,
  getCachedChecklist,
  getCachedExpenses,
  getCachedReminders,
  putCachedChecklist,
  putCachedExpenses,
  putCachedReminders,
  updateOfflineReceiptDraft
} from "@/lib/offline/trip-cache";
import type { PendingCompanionMutation } from "@/lib/offline/types";

export const OFFLINE_METADATA_KEY = "offline";

export type OfflineMutationMetadata = {
  pendingSync?: boolean;
  localEntityId?: string;
  clientMutationId?: string;
  operation?: string;
  requestHash?: string;
};

export async function applyOfflineChecklistChecked(input: {
  tripId: string;
  userId: string;
  itemId: string;
  checked: boolean;
  currentUserId?: string | null;
  clientMutationId: string;
}): Promise<ChecklistViewResponse | null> {
  const cached = await getCachedChecklist(input.tripId, input.userId);
  if (!cached?.checklist.checklist) {
    return cached?.checklist ?? null;
  }

  const now = new Date().toISOString();
  const checklist: TripChecklist = {
    ...cached.checklist.checklist,
    items: cached.checklist.checklist.items.map((item) =>
      item.id === input.itemId
        ? {
            ...item,
            checked: input.checked,
            checkedAt: input.checked ? now : null,
            checkedByUserId: input.checked ? input.currentUserId ?? null : null,
            updatedAt: now,
            metadata: mergeOfflineMetadata(item.metadata, {
              pendingSync: true,
              clientMutationId: input.clientMutationId,
              operation: input.checked ? "check" : "uncheck"
            })
          }
        : item
    ),
    updatedAt: now
  };
  const response = {
    ...cached.checklist,
    checklist,
    summary: buildChecklistSummary(checklist.items, input.currentUserId)
  };
  await putCachedChecklist({ tripId: input.tripId, userId: input.userId, checklist: response });
  return response;
}

export async function applyOfflineChecklistCreate(input: {
  tripId: string;
  userId: string;
  payload: ChecklistItemPayload;
  currentUserId?: string | null;
  clientMutationId: string;
  localEntityId?: string;
}): Promise<{ response: ChecklistViewResponse; localEntityId: string }> {
  const cached = await getCachedChecklist(input.tripId, input.userId);
  const now = new Date().toISOString();
  const localEntityId = input.localEntityId ?? createOfflineId("checklist-item");
  const checklist =
    cached?.checklist.checklist ??
    ({
      id: createOfflineId("checklist"),
      tripId: input.tripId,
      status: "active",
      title: "Offline checklist",
      summary: null,
      createdByUserId: input.currentUserId ?? input.userId,
      updatedAt: now,
      items: [],
      metadata: {},
      createdAt: now
    } satisfies TripChecklist);
  const item: TripChecklistItem = {
    id: localEntityId,
    checklistId: checklist.id,
    title: input.payload.title,
    description: input.payload.description ?? null,
    category: input.payload.category,
    itemType: input.payload.itemType ?? "packing",
    priority: input.payload.priority ?? "medium",
    quantity: input.payload.quantity ?? null,
    assignedToUserId: input.payload.assignedToUserId ?? null,
    assignedToDisplayName: null,
    dueDate: input.payload.dueDate ?? null,
    checked: false,
    checkedAt: null,
    checkedByUserId: null,
    source: "manual",
    reason: input.payload.reason ?? null,
    relatedDayNumber: input.payload.relatedDayNumber ?? null,
    relatedItemIndex: input.payload.relatedItemIndex ?? null,
    relatedItemId: input.payload.relatedItemId ?? null,
    sortOrder: nextSortOrder(checklist.items),
    metadata: mergeOfflineMetadata(input.payload.metadata, {
      pendingSync: true,
      localEntityId,
      clientMutationId: input.clientMutationId,
      operation: "create"
    }),
    createdAt: now,
    updatedAt: now
  };
  const nextChecklist = {
    ...checklist,
    items: [...checklist.items, item],
    updatedAt: now
  };
  const response: ChecklistViewResponse = {
    checklist: nextChecklist,
    summary: buildChecklistSummary(nextChecklist.items, input.currentUserId),
    canGenerate: cached?.checklist.canGenerate ?? false
  };
  await putCachedChecklist({ tripId: input.tripId, userId: input.userId, checklist: response });
  return { response, localEntityId };
}

export async function replaceOfflineChecklistItem(input: {
  tripId: string;
  userId: string;
  localEntityId?: string | null;
  item: TripChecklistItem;
}): Promise<void> {
  const cached = await getCachedChecklist(input.tripId, input.userId);
  if (!cached?.checklist.checklist) {
    return;
  }
  const checklist = cached.checklist.checklist;
  const nextItems = checklist.items.map((item) =>
    item.id === input.localEntityId || item.id === input.item.id ? input.item : item
  );
  await putCachedChecklist({
    tripId: input.tripId,
    userId: input.userId,
    checklist: {
      ...cached.checklist,
      checklist: { ...checklist, items: nextItems, updatedAt: new Date().toISOString() },
      summary: buildChecklistSummary(nextItems, input.userId)
    }
  });
}

export async function applyOfflineReminderStatus(input: {
  tripId: string;
  userId: string;
  reminderId: string;
  status: "completed" | "pending" | "disabled";
  currentUserId?: string | null;
  clientMutationId: string;
}): Promise<TripReminder[] | null> {
  const cached = await getCachedReminders(input.tripId, input.userId);
  if (!cached) {
    return null;
  }
  const now = new Date().toISOString();
  const reminders = cached.reminders.map((reminder) =>
    reminder.id === input.reminderId
      ? {
          ...reminder,
          status: input.status,
          completedAt: input.status === "completed" ? now : null,
          completedByUserId: input.status === "completed" ? input.currentUserId ?? null : null,
          disabledAt: input.status === "disabled" ? now : null,
          disabledByUserId: input.status === "disabled" ? input.currentUserId ?? null : null,
          updatedAt: now,
          metadata: mergeOfflineMetadata(reminder.metadata, {
            pendingSync: true,
            clientMutationId: input.clientMutationId,
            operation: input.status
          })
        }
      : reminder
  );
  await putCachedReminders({
    tripId: input.tripId,
    userId: input.userId,
    reminders,
    summary: buildReminderSummary(reminders, input.currentUserId)
  });
  return reminders;
}

export async function applyOfflineReminderCreate(input: {
  tripId: string;
  userId: string;
  payload: CreateReminderInput;
  currentUserId?: string | null;
  clientMutationId: string;
  localEntityId?: string;
}): Promise<{ reminders: TripReminder[]; localEntityId: string }> {
  const cached = await getCachedReminders(input.tripId, input.userId);
  const now = new Date().toISOString();
  const localEntityId = input.localEntityId ?? createOfflineId("reminder");
  const reminder: TripReminder = {
    id: localEntityId,
    tripId: input.tripId,
    title: input.payload.title,
    description: input.payload.description ?? null,
    category: input.payload.category,
    priority: input.payload.priority ?? "medium",
    source: "manual",
    status: "pending",
    triggerDate: input.payload.triggerDate,
    triggerTime: input.payload.triggerTime ?? null,
    timezone: input.payload.timezone ?? null,
    relativeOffsetDays: input.payload.relativeOffsetDays ?? null,
    assignedToUserId: input.payload.assignedToUserId ?? null,
    assignedToDisplayName: null,
    checklistItemId: input.payload.checklistItemId ?? null,
    relatedDayNumber: input.payload.relatedDayNumber ?? null,
    relatedItemIndex: input.payload.relatedItemIndex ?? null,
    relatedItemId: input.payload.relatedItemId ?? null,
    completedAt: null,
    completedByUserId: null,
    disabledAt: null,
    disabledByUserId: null,
    failureReason: null,
    metadata: mergeOfflineMetadata(input.payload.metadata, {
      pendingSync: true,
      localEntityId,
      clientMutationId: input.clientMutationId,
      operation: "create"
    }),
    createdByUserId: input.currentUserId ?? input.userId,
    updatedByUserId: input.currentUserId ?? input.userId,
    createdAt: now,
    updatedAt: now
  };
  const reminders = [...(cached?.reminders ?? []), reminder];
  await putCachedReminders({
    tripId: input.tripId,
    userId: input.userId,
    reminders,
    summary: buildReminderSummary(reminders, input.currentUserId)
  });
  return { reminders, localEntityId };
}

export async function replaceOfflineReminder(input: {
  tripId: string;
  userId: string;
  localEntityId?: string | null;
  reminder: TripReminder;
}) {
  const cached = await getCachedReminders(input.tripId, input.userId);
  if (!cached) {
    return;
  }
  const reminders = cached.reminders.map((reminder) =>
    reminder.id === input.localEntityId || reminder.id === input.reminder.id
      ? input.reminder
      : reminder
  );
  await putCachedReminders({
    tripId: input.tripId,
    userId: input.userId,
    reminders,
    summary: buildReminderSummary(reminders, input.userId)
  });
}

export async function applyOfflineExpenseCreate(input: {
  tripId: string;
  userId: string;
  payload: CreateExpenseInput;
  users: Array<{ id: string; name: string }>;
  currentUserId?: string | null;
  clientMutationId: string;
  localEntityId?: string;
}): Promise<{ expenses: TripExpense[]; localEntityId: string }> {
  const cached = await getCachedExpenses(input.tripId, input.userId);
  const now = new Date().toISOString();
  const localEntityId = input.localEntityId ?? createOfflineId("expense");
  const participants = buildExpenseParticipants(input.payload, input.users);
  const expense: TripExpense = {
    id: localEntityId,
    tripId: input.tripId,
    title: input.payload.title,
    description: input.payload.description ?? null,
    amount: input.payload.amount,
    category: input.payload.category,
    expenseDate: input.payload.expenseDate,
    paidByUserId: input.payload.paidByUserId,
    paidByDisplayName:
      input.users.find((user) => user.id === input.payload.paidByUserId)?.name ?? "You",
    splitType: input.payload.splitType,
    participants,
    linkedItinerary: input.payload.linkedItinerary ?? null,
    linkedRouteLegId: input.payload.linkedRouteLegId ?? null,
    linkedAccommodation: input.payload.linkedAccommodation ?? false,
    notes: input.payload.notes ?? null,
    metadata: mergeOfflineMetadata(input.payload.metadata, {
      pendingSync: true,
      localEntityId,
      clientMutationId: input.clientMutationId,
      operation: "create"
    }),
    receiptCount: 0,
    hasReceipt: false,
    latestReceiptStatus: null,
    receipts: [],
    createdByUserId: input.currentUserId ?? input.userId,
    createdAt: now,
    updatedAt: now
  };
  const expenses = [...(cached?.expenses ?? []), expense];
  await putCachedExpenses({ tripId: input.tripId, userId: input.userId, expenses });
  return { expenses, localEntityId };
}

export async function replaceOfflineExpense(input: {
  tripId: string;
  userId: string;
  localEntityId?: string | null;
  expense: TripExpense;
}) {
  const cached = await getCachedExpenses(input.tripId, input.userId);
  if (!cached) {
    return;
  }
  const expenses = cached.expenses.map((expense) =>
    expense.id === input.localEntityId || expense.id === input.expense.id ? input.expense : expense
  );
  await putCachedExpenses({ tripId: input.tripId, userId: input.userId, expenses });
}

export async function rollbackOfflineCompanionMutation(
  mutation: PendingCompanionMutation
): Promise<void> {
  switch (mutation.type) {
    case "checklist_item_create":
      await removeCachedChecklistItem(mutation, mutation.payload.localEntityId);
      return;
    case "checklist_item_check":
      await rollbackCachedChecklistCheck(mutation, mutation.payload.itemId, false);
      return;
    case "checklist_item_uncheck":
      await rollbackCachedChecklistCheck(mutation, mutation.payload.itemId, true);
      return;
    case "reminder_create":
      await removeCachedReminder(mutation, mutation.payload.localEntityId);
      return;
    case "reminder_complete":
      await rollbackCachedReminderStatus(mutation, mutation.payload.reminderId, "pending");
      return;
    case "reminder_reopen":
      await rollbackCachedReminderStatus(mutation, mutation.payload.reminderId, "completed");
      return;
    case "reminder_disable":
      await rollbackCachedReminderStatus(mutation, mutation.payload.reminderId, "pending");
      return;
    case "expense_create":
      await removeCachedExpense(mutation, mutation.payload.localEntityId);
      return;
    case "receipt_upload":
      await updateOfflineReceiptDraft(mutation.payload.receiptDraftId, {
        status: "cancelled",
        error: null
      });
      return;
    case "checklist_item_update":
    case "checklist_item_delete_local":
    case "expense_update_local":
    case "expense_delete_local":
      return;
  }
}

export function buildChecklistSummary(
  items: TripChecklistItem[],
  currentUserId?: string | null
): ChecklistSummary {
  const checkedItems = items.filter((item) => item.checked).length;
  const categories = Array.from(new Set(items.map((item) => item.category))).map((category) => {
    const categoryItems = items.filter((item) => item.category === category);
    return {
      category,
      total: categoryItems.length,
      checked: categoryItems.filter((item) => item.checked).length
    };
  });
  return {
    totalItems: items.length,
    checkedItems,
    uncheckedItems: items.length - checkedItems,
    highPriorityUnchecked: items.filter(
      (item) => !item.checked && (item.priority === "high" || item.priority === "critical")
    ).length,
    assignedToMe: items.filter((item) => currentUserId && item.assignedToUserId === currentUserId)
      .length,
    categories
  };
}

export function buildReminderSummary(
  reminders: TripReminder[],
  currentUserId?: string | null
): ReminderSummary {
  const today = new Date().toISOString().slice(0, 10);
  return {
    total: reminders.length,
    pending: reminders.filter((reminder) => reminder.status === "pending").length,
    completed: reminders.filter((reminder) => reminder.status === "completed").length,
    overdue: reminders.filter(
      (reminder) => reminder.status === "pending" && reminder.triggerDate < today
    ).length,
    dueToday: reminders.filter(
      (reminder) => reminder.status === "pending" && reminder.triggerDate === today
    ).length,
    highPriorityPending: reminders.filter(
      (reminder) =>
        reminder.status === "pending" &&
        (reminder.priority === "high" || reminder.priority === "critical")
    ).length,
    assignedToMe: reminders.filter(
      (reminder) => currentUserId && reminder.assignedToUserId === currentUserId
    ).length,
    stale: true
  };
}

export function expenseSummaryWithPending(
  summary: ExpenseSummary | null,
  expenses: TripExpense[],
  currency: string
): ExpenseSummary | null {
  if (!summary) {
    return null;
  }
  const pendingTotal = expenses
    .filter((expense) => isOfflinePending(expense.metadata))
    .filter((expense) => expense.amount.currency === summary.currency)
    .reduce((total, expense) => total + expense.amount.amount, 0);
  if (pendingTotal <= 0) {
    return summary;
  }
  return {
    ...summary,
    actualTotal: {
      amount: summary.actualTotal.amount + pendingTotal,
      currency: summary.actualTotal.currency || currency
    },
    settlementSummary: {
      ...summary.settlementSummary,
      totalPending: {
        amount: summary.settlementSummary.totalPending.amount,
        currency: summary.settlementSummary.totalPending.currency || currency
      }
    }
  };
}

export function isOfflinePending(metadata?: Record<string, unknown> | null) {
  const offline = metadata?.[OFFLINE_METADATA_KEY];
  return Boolean(
    offline &&
      typeof offline === "object" &&
      "pendingSync" in offline &&
      (offline as OfflineMutationMetadata).pendingSync
  );
}

export function clearOfflineMetadata<T extends { metadata?: Record<string, unknown> | null }>(
  value: T
): T {
  if (!value.metadata?.[OFFLINE_METADATA_KEY]) {
    return value;
  }
  const metadata = { ...value.metadata };
  delete metadata[OFFLINE_METADATA_KEY];
  return { ...value, metadata };
}

function mergeOfflineMetadata(
  metadata: Record<string, unknown> | null | undefined,
  offline: OfflineMutationMetadata
) {
  return {
    ...(metadata ?? {}),
    [OFFLINE_METADATA_KEY]: {
      ...(((metadata ?? {})[OFFLINE_METADATA_KEY] as Record<string, unknown> | undefined) ?? {}),
      ...offline
    }
  };
}

function nextSortOrder(items: TripChecklistItem[]) {
  return items.reduce((max, item) => Math.max(max, item.sortOrder), 0) + 1;
}

function buildExpenseParticipants(
  input: CreateExpenseInput,
  users: Array<{ id: string; name: string }>
): ExpenseParticipant[] {
  const participantIds =
    input.splitType === "equal"
      ? users.map((user) => user.id)
      : input.splitType === "payer_only"
        ? [input.paidByUserId]
        : input.participantUserIds?.length
          ? input.participantUserIds
          : users.map((user) => user.id);
  const shares = splitAmounts(input.amount, participantIds, input);
  return participantIds.map((userId, index) => ({
    userId,
    displayName: users.find((user) => user.id === userId)?.name ?? userId.slice(0, 8),
    shareAmount: shares[index],
    sharePercentage:
      input.splitType === "custom_percentages"
        ? input.customPercentages?.find((item) => item.userId === userId)?.percentage ?? null
        : null
  }));
}

function splitAmounts(
  amount: MoneyAmount,
  participantIds: string[],
  input: CreateExpenseInput
): MoneyAmount[] {
  if (input.splitType === "custom_amounts") {
    return participantIds.map((userId) => {
      const share = input.customShares?.find((item) => item.userId === userId);
      return {
        amount: share?.amount ?? 0,
        currency: share?.currency ?? amount.currency
      };
    });
  }
  if (input.splitType === "custom_percentages") {
    return participantIds.map((userId) => {
      const percentage = input.customPercentages?.find((item) => item.userId === userId)?.percentage ?? 0;
      return {
        amount: amount.amount * (percentage / 100),
        currency: amount.currency
      };
    });
  }
  const cents = Math.round(amount.amount * 100);
  const base = participantIds.length ? Math.floor(cents / participantIds.length) : 0;
  const remainder = participantIds.length ? cents % participantIds.length : 0;
  return participantIds.map((_, index) => ({
    amount: (base + (index < remainder ? 1 : 0)) / 100,
    currency: amount.currency
  }));
}

async function removeCachedChecklistItem(
  mutation: PendingCompanionMutation,
  localEntityId: string
) {
  const cached = await getCachedChecklist(mutation.tripId, mutation.userId);
  const checklist = cached?.checklist.checklist;
  if (!cached || !checklist) {
    return;
  }
  const items = checklist.items.filter((item) => item.id !== localEntityId);
  await putCachedChecklist({
    tripId: mutation.tripId,
    userId: mutation.userId,
    checklist: {
      ...cached.checklist,
      checklist: { ...checklist, items, updatedAt: new Date().toISOString() },
      summary: buildChecklistSummary(items, mutation.userId)
    }
  });
}

async function rollbackCachedChecklistCheck(
  mutation: PendingCompanionMutation,
  itemId: string,
  checked: boolean
) {
  const cached = await getCachedChecklist(mutation.tripId, mutation.userId);
  const checklist = cached?.checklist.checklist;
  if (!cached || !checklist) {
    return;
  }
  const now = new Date().toISOString();
  const items = checklist.items.map((item) =>
    item.id === itemId
      ? clearOfflineMetadata({
          ...item,
          checked,
          checkedAt: checked ? item.checkedAt ?? now : null,
          checkedByUserId: checked ? item.checkedByUserId : null,
          updatedAt: now
        })
      : item
  );
  await putCachedChecklist({
    tripId: mutation.tripId,
    userId: mutation.userId,
    checklist: {
      ...cached.checklist,
      checklist: { ...checklist, items, updatedAt: now },
      summary: buildChecklistSummary(items, mutation.userId)
    }
  });
}

async function removeCachedReminder(mutation: PendingCompanionMutation, localEntityId: string) {
  const cached = await getCachedReminders(mutation.tripId, mutation.userId);
  if (!cached) {
    return;
  }
  const reminders = cached.reminders.filter((reminder) => reminder.id !== localEntityId);
  await putCachedReminders({
    tripId: mutation.tripId,
    userId: mutation.userId,
    reminders,
    summary: buildReminderSummary(reminders, mutation.userId)
  });
}

async function rollbackCachedReminderStatus(
  mutation: PendingCompanionMutation,
  reminderId: string,
  status: TripReminder["status"]
) {
  const cached = await getCachedReminders(mutation.tripId, mutation.userId);
  if (!cached) {
    return;
  }
  const now = new Date().toISOString();
  const reminders = cached.reminders.map((reminder) =>
    reminder.id === reminderId
      ? clearOfflineMetadata({
          ...reminder,
          status,
          completedAt: status === "completed" ? reminder.completedAt ?? now : null,
          completedByUserId: status === "completed" ? reminder.completedByUserId : null,
          disabledAt: status === "disabled" ? reminder.disabledAt ?? now : null,
          disabledByUserId: status === "disabled" ? reminder.disabledByUserId : null,
          updatedAt: now
        })
      : reminder
  );
  await putCachedReminders({
    tripId: mutation.tripId,
    userId: mutation.userId,
    reminders,
    summary: buildReminderSummary(reminders, mutation.userId)
  });
}

async function removeCachedExpense(mutation: PendingCompanionMutation, localEntityId: string) {
  const cached = await getCachedExpenses(mutation.tripId, mutation.userId);
  if (!cached) {
    return;
  }
  const expenses = cached.expenses.filter((expense) => expense.id !== localEntityId);
  await putCachedExpenses({ tripId: mutation.tripId, userId: mutation.userId, expenses });
}
