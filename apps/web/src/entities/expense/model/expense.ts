import type { ExpenseReceiptSummary, ReceiptStatus } from "@/entities/receipt/model";

export type MoneyAmount = {
  amount: number;
  currency: string;
};

export type ExpenseCategory =
  | "transport"
  | "accommodation"
  | "food"
  | "tickets"
  | "activities"
  | "shopping"
  | "fuel"
  | "parking"
  | "tolls"
  | "camping"
  | "groceries"
  | "health_safety"
  | "other";

export type ExpenseSplitType =
  | "equal"
  | "selected_equal"
  | "custom_amounts"
  | "custom_percentages"
  | "payer_only";

export type SettlementStatus = "pending" | "paid" | "cancelled";
export type SettlementSource = "calculated" | "manual";

export type ExpenseParticipant = {
  userId: string;
  displayName: string;
  shareAmount: MoneyAmount;
  sharePercentage?: number | null;
};

export type LinkedItineraryRef = {
  dayNumber: number;
  itemIndex: number;
  itemId?: string | null;
};

export type TripExpense = {
  id: string;
  tripId: string;
  title: string;
  description?: string | null;
  amount: MoneyAmount;
  category: ExpenseCategory;
  expenseDate: string;
  paidByUserId: string;
  paidByDisplayName: string;
  splitType: ExpenseSplitType;
  participants: ExpenseParticipant[];
  linkedItinerary?: LinkedItineraryRef | null;
  linkedRouteLegId?: string | null;
  linkedAccommodation: boolean;
  notes?: string | null;
  metadata: Record<string, unknown>;
  receiptCount: number;
  hasReceipt: boolean;
  latestReceiptStatus?: ReceiptStatus | null;
  receipts: ExpenseReceiptSummary[];
  createdByUserId: string;
  createdAt: string;
  updatedAt: string;
};

export type TripExpensesResponse = {
  items: TripExpense[];
  nextOffset?: number | null;
};

export type ExpenseCustomAmount = {
  userId: string;
  amount: number;
  currency: string;
};

export type ExpenseCustomPercentage = {
  userId: string;
  percentage: number;
};

export type CreateExpenseInput = {
  title: string;
  description?: string | null;
  amount: MoneyAmount;
  category: ExpenseCategory;
  expenseDate: string;
  paidByUserId: string;
  splitType: ExpenseSplitType;
  participantUserIds?: string[];
  customShares?: ExpenseCustomAmount[];
  customPercentages?: ExpenseCustomPercentage[];
  linkedItinerary?: LinkedItineraryRef | null;
  linkedRouteLegId?: string | null;
  linkedAccommodation?: boolean;
  notes?: string | null;
  metadata?: Record<string, unknown>;
};

export type UpdateExpenseInput = Partial<CreateExpenseInput>;

export type ListExpensesFilters = {
  category?: ExpenseCategory | null;
  paidByUserId?: string | null;
  fromDate?: string | null;
  toDate?: string | null;
  linkedOnly?: boolean;
  limit?: number;
  offset?: number;
};

export type ExpenseCategoryTotal = {
  category: ExpenseCategory;
  amount: MoneyAmount;
};

export type ExpensePayerTotal = {
  userId: string;
  displayName: string;
  paid: MoneyAmount;
};

export type ExpenseBalance = {
  userId: string;
  displayName: string;
  paid: MoneyAmount;
  share: MoneyAmount;
  net: MoneyAmount;
  netBeforeSettlements: MoneyAmount;
  settledAmount: MoneyAmount;
  netOutstanding: MoneyAmount;
  status: "owes" | "gets_back" | "settled" | string;
};

export type PlannedVsActual = {
  difference: MoneyAmount;
  percentUsed: number;
};

export type SettlementSummary = {
  pendingCount: number;
  paidCount: number;
  totalPending: MoneyAmount;
};

export type ExpenseSummary = {
  tripId: string;
  expenseCount?: number;
  currency: string;
  actualTotal: MoneyAmount;
  estimatedTotal?: MoneyAmount | null;
  plannedVsActual?: PlannedVsActual | null;
  originalCurrencyTotals: MoneyAmount[];
  byCategory: ExpenseCategoryTotal[];
  byPayer: ExpensePayerTotal[];
  balances: ExpenseBalance[];
  conversionWarnings: string[];
  settlementSummary: SettlementSummary;
};

export type SettlementSuggestion = {
  id: string;
  fromUserId: string;
  fromDisplayName: string;
  toUserId: string;
  toDisplayName: string;
  amount: MoneyAmount;
  status: SettlementStatus;
  source: SettlementSource;
  calculationHash?: string;
};

export type TripSettlement = {
  id: string;
  tripId: string;
  fromUserId: string;
  fromDisplayName: string;
  toUserId: string;
  toDisplayName: string;
  amount: MoneyAmount;
  status: SettlementStatus;
  source: SettlementSource;
  paidAt?: string | null;
  paidByUserId?: string | null;
  cancelledAt?: string | null;
  cancelledByUserId?: string | null;
  notes?: string | null;
  createdAt: string;
  updatedAt: string;
};

export type SettlementsResponse = {
  currency: string;
  suggestions: SettlementSuggestion[];
  paidSettlements: TripSettlement[];
  warnings: string[];
};

export type MarkSettlementPaidInput = {
  notes?: string | null;
};

export const EXPENSE_CATEGORIES: ExpenseCategory[] = [
  "transport",
  "accommodation",
  "food",
  "tickets",
  "activities",
  "shopping",
  "fuel",
  "parking",
  "tolls",
  "camping",
  "groceries",
  "health_safety",
  "other"
];

export const EXPENSE_SPLIT_TYPES: ExpenseSplitType[] = [
  "equal",
  "selected_equal",
  "custom_amounts",
  "custom_percentages",
  "payer_only"
];
