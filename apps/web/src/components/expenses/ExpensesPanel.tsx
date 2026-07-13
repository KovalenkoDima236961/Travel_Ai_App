"use client";

import { FormEvent, useMemo, useState } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Textarea } from "@/shared/ui/textarea";
import { formatMoney } from "@/entities/budget/model";
import {
  EXPENSE_CATEGORIES,
  EXPENSE_SPLIT_TYPES,
  type CreateExpenseInput,
  type ExpenseCategory,
  type ExpenseSplitType,
  type SettlementSuggestion,
  type TripExpense
} from "@/entities/expense/model";
import type { Trip } from "@/entities/trip/model";
import type { TripTraveler } from "@/entities/cost-splitting/model";
import {
  useCreateTripExpense,
  useDeleteTripExpense,
  useMarkSettlementPaid,
  useTripExpenseSummary,
  useTripExpenses,
  useTripSettlements
} from "@/hooks/useTripExpenses";
import { getErrorMessage } from "@/lib/utils";

type ExpenseUserOption = {
  id: string;
  name: string;
};

type ExpensesPanelProps = {
  trip: Trip;
  travelers: TripTraveler[];
  canEdit: boolean;
  offline?: boolean;
  currentUserId?: string | null;
};

export function ExpensesPanel({
  trip,
  travelers,
  canEdit,
  offline = false,
  currentUserId
}: ExpensesPanelProps) {
  const t = useTranslations("expenses");
  const settlementsT = useTranslations("settlements");
  const [addOpen, setAddOpen] = useState(false);
  const [payingSuggestion, setPayingSuggestion] = useState<SettlementSuggestion | null>(null);
  const [settlementNotes, setSettlementNotes] = useState("");
  const [panelError, setPanelError] = useState<string | null>(null);
  const currency = trip.budgetCurrency ?? "EUR";
  const enabled = !offline;
  const users = useMemo(
    () => buildExpenseUsers(trip, travelers, currentUserId),
    [trip, travelers, currentUserId]
  );

  const expensesQuery = useTripExpenses({ tripId: trip.id, enabled });
  const summaryQuery = useTripExpenseSummary({ tripId: trip.id, currency, enabled });
  const settlementsQuery = useTripSettlements({ tripId: trip.id, currency, enabled });
  const createMutation = useCreateTripExpense(trip.id);
  const deleteMutation = useDeleteTripExpense(trip.id);
  const markPaidMutation = useMarkSettlementPaid(trip.id);

  const expenses = expensesQuery.data?.items ?? [];
  const summary = summaryQuery.data ?? null;
  const settlements = settlementsQuery.data ?? null;
  const canMutateExpenses = canEdit && !offline && users.length > 0;

  async function markPaid(suggestion: SettlementSuggestion) {
    setPanelError(null);
    try {
      await markPaidMutation.mutateAsync({
        settlementId: suggestion.id,
        input: { notes: settlementNotes.trim() || null }
      });
      setPayingSuggestion(null);
      setSettlementNotes("");
    } catch (error) {
      setPanelError(getErrorMessage(error, settlementsT("markPaidError")));
    }
  }

  async function removeExpense(expense: TripExpense) {
    setPanelError(null);
    try {
      await deleteMutation.mutateAsync(expense.id);
    } catch (error) {
      setPanelError(getErrorMessage(error, t("deleteError")));
    }
  }

  return (
    <section className="space-y-4" id="expenses">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 className="text-xl font-semibold text-slate-950">{t("title")}</h2>
          <p className="mt-1 text-sm leading-6 text-slate-600">{t("subtitle")}</p>
        </div>
        {canMutateExpenses ? (
          <Button
            disabled={createMutation.isPending}
            onClick={() => setAddOpen((open) => !open)}
            size="sm"
            type="button"
            variant="secondary"
          >
            {addOpen ? t("closeAdd") : t("addExpense")}
          </Button>
        ) : null}
      </div>

      {offline ? (
        <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
          {t("offline")}
        </div>
      ) : null}

      {panelError ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
          {panelError}
        </div>
      ) : null}

      {addOpen ? (
        <AddExpenseDialog
          currency={currency}
          isSaving={createMutation.isPending}
          onCancel={() => setAddOpen(false)}
          onSubmit={async (input) => {
            setPanelError(null);
            try {
              await createMutation.mutateAsync(input);
              setAddOpen(false);
            } catch (error) {
              setPanelError(getErrorMessage(error, t("createError")));
            }
          }}
          users={users}
        />
      ) : null}

      <div className="grid gap-4 md:grid-cols-3">
        <MetricCard
          label={t("actualTotal")}
          loading={summaryQuery.isLoading}
          value={formatMoney(summary?.actualTotal.amount, summary?.actualTotal.currency ?? currency)}
        />
        <MetricCard
          label={t("plannedTotal")}
          loading={summaryQuery.isLoading}
          value={
            summary?.estimatedTotal
              ? formatMoney(summary.estimatedTotal.amount, summary.estimatedTotal.currency)
              : t("notAvailable")
          }
        />
        <MetricCard
          label={settlementsT("pending")}
          loading={summaryQuery.isLoading}
          value={formatMoney(
            summary?.settlementSummary.totalPending.amount,
            summary?.settlementSummary.totalPending.currency ?? currency
          )}
        />
      </div>

      {summary?.plannedVsActual ? (
        <Card className="grid gap-3 md:grid-cols-2">
          <SummaryRow
            label={t("plannedDifference")}
            value={formatMoney(
              summary.plannedVsActual.difference.amount,
              summary.plannedVsActual.difference.currency
            )}
          />
          <SummaryRow
            label={t("percentUsed")}
            value={`${Math.round(summary.plannedVsActual.percentUsed)}%`}
          />
        </Card>
      ) : null}

      {summary?.conversionWarnings.length ? (
        <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
          <p className="font-medium">{t("conversionWarnings")}</p>
          <ul className="mt-2 list-disc space-y-1 pl-5">
            {summary.conversionWarnings.map((warning, index) => (
              <li key={`${warning}-${index}`}>{warning}</li>
            ))}
          </ul>
        </div>
      ) : null}

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_24rem]">
        <Card>
          <h3 className="text-base font-semibold text-slate-950">{t("expenses")}</h3>
          {expensesQuery.isLoading ? (
            <p className="mt-4 text-sm text-slate-500">{t("loading")}</p>
          ) : expenses.length === 0 ? (
            <p className="mt-4 text-sm text-slate-500">{t("empty")}</p>
          ) : (
            <div className="mt-4 divide-y divide-slate-100">
              {expenses.map((expense) => (
                <ExpenseRow
                  canDelete={canMutateExpenses}
                  deleting={deleteMutation.isPending}
                  expense={expense}
                  key={expense.id}
                  onDelete={removeExpense}
                />
              ))}
            </div>
          )}
        </Card>

        <Card>
          <h3 className="text-base font-semibold text-slate-950">{settlementsT("title")}</h3>
          <p className="mt-1 text-sm leading-6 text-slate-600">{settlementsT("subtitle")}</p>

          {settlementsQuery.isLoading ? (
            <p className="mt-4 text-sm text-slate-500">{settlementsT("loading")}</p>
          ) : null}

          {summary?.balances.length ? (
            <div className="mt-4 space-y-2">
              {summary.balances.map((balance) => (
                <SummaryRow
                  key={balance.userId}
                  label={balance.displayName}
                  value={formatMoney(
                    balance.netOutstanding.amount,
                    balance.netOutstanding.currency
                  )}
                />
              ))}
            </div>
          ) : null}

          <div className="mt-5 space-y-3">
            {(settlements?.suggestions ?? []).length === 0 ? (
              <p className="text-sm text-slate-500">{settlementsT("settled")}</p>
            ) : (
              settlements?.suggestions.map((suggestion) => (
                <div
                  className="rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm"
                  key={suggestion.id}
                >
                  <div className="flex items-start justify-between gap-3">
                    <p className="leading-6 text-slate-700">
                      <span className="font-medium text-slate-950">
                        {suggestion.fromDisplayName}
                      </span>{" "}
                      {settlementsT("pays")}{" "}
                      <span className="font-medium text-slate-950">
                        {suggestion.toDisplayName}
                      </span>
                    </p>
                    <span className="whitespace-nowrap font-semibold text-slate-950">
                      {formatMoney(suggestion.amount.amount, suggestion.amount.currency)}
                    </span>
                  </div>
                  {canMutateExpenses ? (
                    <Button
                      className="mt-3"
                      disabled={markPaidMutation.isPending}
                      onClick={() => setPayingSuggestion(suggestion)}
                      size="sm"
                      type="button"
                      variant="secondary"
                    >
                      {settlementsT("markPaid")}
                    </Button>
                  ) : null}
                </div>
              ))
            )}
          </div>

          {payingSuggestion ? (
            <div className="mt-4 rounded-lg border border-slate-200 bg-white p-3">
              <p className="text-sm font-medium text-slate-950">
                {settlementsT("confirmPaid")}
              </p>
              <Textarea
                className="mt-3 min-h-20"
                onChange={(event) => setSettlementNotes(event.target.value)}
                placeholder={settlementsT("notesPlaceholder")}
                value={settlementNotes}
              />
              <div className="mt-3 flex flex-wrap gap-2">
                <Button
                  disabled={markPaidMutation.isPending}
                  onClick={() => markPaid(payingSuggestion)}
                  size="sm"
                  type="button"
                >
                  {settlementsT("confirm")}
                </Button>
                <Button
                  onClick={() => {
                    setPayingSuggestion(null);
                    setSettlementNotes("");
                  }}
                  size="sm"
                  type="button"
                  variant="ghost"
                >
                  {settlementsT("cancel")}
                </Button>
              </div>
            </div>
          ) : null}

          {(settlements?.paidSettlements ?? []).length ? (
            <div className="mt-5">
              <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                {settlementsT("paid")}
              </p>
              <ul className="mt-2 space-y-2 text-sm text-slate-600">
                {settlements?.paidSettlements.map((settlement) => (
                  <li className="flex items-center justify-between gap-3" key={settlement.id}>
                    <span>
                      {settlement.fromDisplayName} to {settlement.toDisplayName}
                    </span>
                    <span className="font-medium text-slate-900">
                      {formatMoney(settlement.amount.amount, settlement.amount.currency)}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          ) : null}
        </Card>
      </div>

      <p className="text-xs leading-5 text-slate-500">{t("disclaimer")}</p>
    </section>
  );
}

function AddExpenseDialog({
  users,
  currency,
  isSaving,
  onCancel,
  onSubmit
}: {
  users: ExpenseUserOption[];
  currency: string;
  isSaving: boolean;
  onCancel: () => void;
  onSubmit: (input: CreateExpenseInput) => Promise<void>;
}) {
  const t = useTranslations("expenses");
  const [title, setTitle] = useState("");
  const [amount, setAmount] = useState("");
  const [expenseCurrency, setExpenseCurrency] = useState(currency);
  const [category, setCategory] = useState<ExpenseCategory>("food");
  const [expenseDate, setExpenseDate] = useState(() => new Date().toISOString().slice(0, 10));
  const [paidByUserId, setPaidByUserId] = useState(users[0]?.id ?? "");
  const [splitType, setSplitType] = useState<ExpenseSplitType>("selected_equal");
  const [selectedUserIds, setSelectedUserIds] = useState<string[]>(() =>
    users.map((user) => user.id)
  );
  const [customValues, setCustomValues] = useState<Record<string, string>>({});
  const [notes, setNotes] = useState("");

  const numericAmount = Number.parseFloat(amount);
  const splitPreview = previewSplit({
    amount: Number.isFinite(numericAmount) ? numericAmount : 0,
    currency: expenseCurrency,
    customValues,
    paidByUserId,
    selectedUserIds,
    splitType,
    users
  });

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const participantUserIds =
      splitType === "selected_equal" || splitType === "custom_amounts" || splitType === "custom_percentages"
        ? selectedUserIds
        : undefined;
    await onSubmit({
      title,
      amount: { amount: Number.parseFloat(amount), currency: expenseCurrency },
      category,
      expenseDate,
      paidByUserId,
      splitType,
      participantUserIds,
      customShares:
        splitType === "custom_amounts"
          ? selectedUserIds.map((id) => ({
              userId: id,
              amount: Number.parseFloat(customValues[id] ?? "0"),
              currency: expenseCurrency
            }))
          : undefined,
      customPercentages:
        splitType === "custom_percentages"
          ? selectedUserIds.map((id) => ({
              userId: id,
              percentage: Number.parseFloat(customValues[id] ?? "0")
            }))
          : undefined,
      notes: notes.trim() || null,
      linkedAccommodation: false,
      metadata: {}
    });
  }

  function toggleUser(id: string) {
    setSelectedUserIds((current) =>
      current.includes(id) ? current.filter((item) => item !== id) : [...current, id]
    );
  }

  const customTotal = selectedUserIds.reduce(
    (total, id) => total + (Number.parseFloat(customValues[id] ?? "0") || 0),
    0
  );

  return (
    <Card>
      <form className="space-y-4" onSubmit={submit}>
        <div className="grid gap-4 md:grid-cols-2">
          <label className="space-y-1 text-sm font-medium text-slate-700">
            {t("form.title")}
            <Input
              onChange={(event) => setTitle(event.target.value)}
              required
              value={title}
            />
          </label>
          <label className="space-y-1 text-sm font-medium text-slate-700">
            {t("form.date")}
            <Input
              onChange={(event) => setExpenseDate(event.target.value)}
              required
              type="date"
              value={expenseDate}
            />
          </label>
          <label className="space-y-1 text-sm font-medium text-slate-700">
            {t("form.amount")}
            <Input
              min="0.01"
              onChange={(event) => setAmount(event.target.value)}
              required
              step="0.01"
              type="number"
              value={amount}
            />
          </label>
          <label className="space-y-1 text-sm font-medium text-slate-700">
            {t("form.currency")}
            <Input
              maxLength={3}
              onChange={(event) => setExpenseCurrency(event.target.value.toUpperCase())}
              required
              value={expenseCurrency}
            />
          </label>
          <label className="space-y-1 text-sm font-medium text-slate-700">
            {t("form.category")}
            <Select
              onChange={(event) => setCategory(event.target.value as ExpenseCategory)}
              value={category}
            >
              {EXPENSE_CATEGORIES.map((item) => (
                <option key={item} value={item}>
                  {t(`categories.${item}`)}
                </option>
              ))}
            </Select>
          </label>
          <label className="space-y-1 text-sm font-medium text-slate-700">
            {t("form.paidBy")}
            <Select
              onChange={(event) => setPaidByUserId(event.target.value)}
              required
              value={paidByUserId}
            >
              {users.map((user) => (
                <option key={user.id} value={user.id}>
                  {user.name}
                </option>
              ))}
            </Select>
          </label>
        </div>

        <label className="block space-y-1 text-sm font-medium text-slate-700">
          {t("form.splitType")}
          <Select
            onChange={(event) => setSplitType(event.target.value as ExpenseSplitType)}
            value={splitType}
          >
            {EXPENSE_SPLIT_TYPES.map((item) => (
              <option key={item} value={item}>
                {t(`splitTypes.${item}`)}
              </option>
            ))}
          </Select>
        </label>

        {splitType !== "equal" && splitType !== "payer_only" ? (
          <div>
            <p className="text-sm font-medium text-slate-700">{t("form.participants")}</p>
            <div className="mt-2 grid gap-2 sm:grid-cols-2">
              {users.map((user) => (
                <label
                  className="flex items-center justify-between gap-3 rounded-md border border-slate-200 px-3 py-2 text-sm text-slate-700"
                  key={user.id}
                >
                  <span>{user.name}</span>
                  <input
                    checked={selectedUserIds.includes(user.id)}
                    onChange={() => toggleUser(user.id)}
                    type="checkbox"
                  />
                </label>
              ))}
            </div>
          </div>
        ) : null}

        {splitType === "custom_amounts" || splitType === "custom_percentages" ? (
          <div className="grid gap-3 sm:grid-cols-2">
            {selectedUserIds.map((id) => {
              const user = users.find((item) => item.id === id);
              if (!user) {
                return null;
              }
              return (
                <label className="space-y-1 text-sm font-medium text-slate-700" key={id}>
                  {user.name}
                  <Input
                    min="0"
                    onChange={(event) =>
                      setCustomValues((current) => ({
                        ...current,
                        [id]: event.target.value
                      }))
                    }
                    step={splitType === "custom_amounts" ? "0.01" : "1"}
                    type="number"
                    value={customValues[id] ?? ""}
                  />
                </label>
              );
            })}
          </div>
        ) : null}

        {splitType === "custom_amounts" || splitType === "custom_percentages" ? (
          <p className="text-xs text-slate-500">
            {splitType === "custom_amounts"
              ? t("customAmountTotal", {
                  total: formatMoney(customTotal, expenseCurrency),
                  expected: formatMoney(numericAmount || 0, expenseCurrency)
                })
              : t("customPercentTotal", { total: Math.round(customTotal * 100) / 100 })}
          </p>
        ) : null}

        <div className="rounded-lg border border-slate-200 bg-slate-50 p-3">
          <p className="text-sm font-medium text-slate-950">{t("splitPreview")}</p>
          <ul className="mt-2 space-y-1 text-sm text-slate-600">
            {splitPreview.map((item) => (
              <li className="flex items-center justify-between gap-3" key={item.userId}>
                <span>{item.name}</span>
                <span>{formatMoney(item.amount, expenseCurrency)}</span>
              </li>
            ))}
          </ul>
        </div>

        <label className="block space-y-1 text-sm font-medium text-slate-700">
          {t("form.notes")}
          <Textarea onChange={(event) => setNotes(event.target.value)} value={notes} />
        </label>

        <div className="flex flex-wrap gap-2">
          <Button disabled={isSaving || users.length === 0} type="submit">
            {isSaving ? t("saving") : t("save")}
          </Button>
          <Button onClick={onCancel} type="button" variant="ghost">
            {t("cancel")}
          </Button>
        </div>
      </form>
    </Card>
  );
}

function ExpenseRow({
  expense,
  canDelete,
  deleting,
  onDelete
}: {
  expense: TripExpense;
  canDelete: boolean;
  deleting: boolean;
  onDelete: (expense: TripExpense) => void;
}) {
  const t = useTranslations("expenses");
  return (
    <div className="flex flex-col gap-3 py-4 sm:flex-row sm:items-start sm:justify-between">
      <div>
        <div className="flex flex-wrap items-center gap-2">
          <p className="font-medium text-slate-950">{expense.title}</p>
          <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
            {t(`categories.${expense.category}`)}
          </span>
        </div>
        <p className="mt-1 text-sm text-slate-600">
          {expense.paidByDisplayName} · {expense.expenseDate} · {t(`splitTypes.${expense.splitType}`)}
        </p>
        <p className="mt-2 text-xs text-slate-500">
          {expense.participants
            .map((participant) => `${participant.displayName} ${formatMoney(participant.shareAmount.amount, participant.shareAmount.currency)}`)
            .join(", ")}
        </p>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        <span className="font-semibold text-slate-950">
          {formatMoney(expense.amount.amount, expense.amount.currency)}
        </span>
        {canDelete ? (
          <Button
            disabled={deleting}
            onClick={() => onDelete(expense)}
            size="sm"
            type="button"
            variant="ghost"
          >
            {t("delete")}
          </Button>
        ) : null}
      </div>
    </div>
  );
}

function MetricCard({
  label,
  value,
  loading
}: {
  label: string;
  value: string;
  loading: boolean;
}) {
  return (
    <Card>
      <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">{label}</p>
      <p className="mt-2 text-2xl font-semibold text-slate-950">
        {loading ? "..." : value}
      </p>
    </Card>
  );
}

function SummaryRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3 text-sm">
      <span className="text-slate-600">{label}</span>
      <span className="text-right font-medium text-slate-950">{value}</span>
    </div>
  );
}

function buildExpenseUsers(
  trip: Trip,
  travelers: TripTraveler[],
  currentUserId?: string | null
): ExpenseUserOption[] {
  const users = new Map<string, string>();
  if (trip.userId) {
    users.set(trip.userId, "Trip owner");
  }
  for (const traveler of travelers) {
    if (traveler.status === "active" && traveler.linkedUserId) {
      users.set(traveler.linkedUserId, traveler.name);
    }
  }
  if (currentUserId && !users.has(currentUserId)) {
    users.set(currentUserId, "You");
  }
  return [...users.entries()]
    .map(([id, name]) => ({ id, name }))
    .sort((a, b) => a.name.localeCompare(b.name) || a.id.localeCompare(b.id));
}

function previewSplit({
  amount,
  currency,
  customValues,
  paidByUserId,
  selectedUserIds,
  splitType,
  users
}: {
  amount: number;
  currency: string;
  customValues: Record<string, string>;
  paidByUserId: string;
  selectedUserIds: string[];
  splitType: ExpenseSplitType;
  users: ExpenseUserOption[];
}) {
  const participantIds =
    splitType === "equal"
      ? users.map((user) => user.id)
      : splitType === "payer_only"
        ? [paidByUserId]
        : selectedUserIds;
  if (splitType === "custom_amounts") {
    return participantIds.map((id) => ({
      userId: id,
      name: users.find((user) => user.id === id)?.name ?? id.slice(0, 8),
      amount: Number.parseFloat(customValues[id] ?? "0") || 0,
      currency
    }));
  }
  if (splitType === "custom_percentages") {
    return participantIds.map((id) => ({
      userId: id,
      name: users.find((user) => user.id === id)?.name ?? id.slice(0, 8),
      amount: amount * ((Number.parseFloat(customValues[id] ?? "0") || 0) / 100),
      currency
    }));
  }
  const cents = Math.round(amount * 100);
  const base = participantIds.length > 0 ? Math.floor(cents / participantIds.length) : 0;
  const remainder = participantIds.length > 0 ? cents % participantIds.length : 0;
  return participantIds.map((id, index) => ({
    userId: id,
    name: users.find((user) => user.id === id)?.name ?? id.slice(0, 8),
    amount: (base + (index < remainder ? 1 : 0)) / 100,
    currency
  }));
}
