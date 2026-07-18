import type { BudgetSummary } from "@/entities/budget/model";

export const budgetSummaryFixture: BudgetSummary = {
  currency: "EUR",
  tripBudget: 600,
  estimatedTotal: 58,
  remaining: 542,
  overBudgetBy: 0,
  missingEstimateCount: 0,
  estimatedItemCount: 4,
  convertedItemCount: 0,
  unconvertedItemCount: 0,
  byDay: [
    { dayNumber: 1, estimatedTotal: 40, missingEstimateCount: 0 },
    { dayNumber: 2, estimatedTotal: 18, missingEstimateCount: 0 }
  ],
  byCategory: [
    { category: "food", estimatedTotal: 24, itemCount: 1 },
    { category: "ticket", estimatedTotal: 18, itemCount: 1 },
    { category: "transport", estimatedTotal: 16, itemCount: 1 }
  ]
};

export const expensesFixture = [
  {
    id: "expense-1",
    tripId: "20000000-0000-4000-8000-000000000001",
    description: "Naschmarkt lunch",
    amount: 24,
    currency: "EUR",
    category: "food",
    paidAt: "2026-04-10T12:00:00Z",
    receiptId: "receipt-1"
  }
] as const;
