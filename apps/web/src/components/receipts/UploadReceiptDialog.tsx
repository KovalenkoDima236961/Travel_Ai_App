"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import { useTranslations } from "next-intl";
import { ButtonSpinner, FieldHint, InlineError } from "@/components/ui";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Textarea } from "@/shared/ui/textarea";
import { formatMoney } from "@/entities/budget/model";
import {
  EXPENSE_CATEGORIES,
  type CreateExpenseInput,
  type ExpenseCategory
} from "@/entities/expense/model";
import {
  RECEIPT_ALLOWED_TYPES,
  RECEIPT_MAX_FILE_SIZE_BYTES,
  type ExpenseReceipt
} from "@/entities/receipt/model";
import { useCreateExpenseFromReceipt } from "@/hooks/useCreateExpenseFromReceipt";
import { useUploadReceipt } from "@/hooks/useUploadReceipt";
import { getErrorMessage } from "@/lib/utils";
import { ReceiptConfidenceBadge } from "./ReceiptConfidenceBadge";
import { ReceiptPreview } from "./ReceiptPreview";
import { ReceiptWarningsList } from "./ReceiptWarningsList";

type ReceiptUserOption = {
  id: string;
  name: string;
};

export function UploadReceiptDialog({
  tripId,
  currency,
  users,
  onClose,
  onCreated
}: {
  tripId: string;
  currency: string;
  users: ReceiptUserOption[];
  onClose: () => void;
  onCreated?: () => void;
}) {
  const t = useTranslations("receipts");
  const expensesT = useTranslations("expenses");
  const errorsT = useTranslations("errors");
  const uploadMutation = useUploadReceipt(tripId);
  const createMutation = useCreateExpenseFromReceipt(tripId);
  const [file, setFile] = useState<File | null>(null);
  const [receipt, setReceipt] = useState<ExpenseReceipt | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [title, setTitle] = useState("");
  const [amount, setAmount] = useState("");
  const [expenseCurrency, setExpenseCurrency] = useState(currency);
  const [category, setCategory] = useState<ExpenseCategory>("other");
  const [expenseDate, setExpenseDate] = useState(() => new Date().toISOString().slice(0, 10));
  const [paidByUserId, setPaidByUserId] = useState(users[0]?.id ?? "");
  const [selectedUserIds, setSelectedUserIds] = useState<string[]>(() => users.map((user) => user.id));
  const [notes, setNotes] = useState("");

  useEffect(() => {
    if (!receipt?.ocrResult) {
      return;
    }
    const result = receipt.ocrResult;
    setTitle(result.suggestedTitle ?? result.merchant ?? t("receiptExpense"));
    setAmount(result.amount?.amount != null ? String(result.amount.amount) : "");
    setExpenseCurrency(result.amount?.currency ?? currency);
    setCategory(result.category ?? "other");
    setExpenseDate(result.expenseDate ?? new Date().toISOString().slice(0, 10));
    setNotes(t("createdFromReceiptNote"));
  }, [currency, receipt, t]);

  const splitPreview = useMemo(() => {
    const numericAmount = Number.parseFloat(amount);
    const total = Number.isFinite(numericAmount) ? numericAmount : 0;
    const ids = selectedUserIds.length > 0 ? selectedUserIds : users.map((user) => user.id);
    const cents = Math.round(total * 100);
    const base = ids.length > 0 ? Math.floor(cents / ids.length) : 0;
    const remainder = ids.length > 0 ? cents % ids.length : 0;
    return ids.map((id, index) => ({
      userId: id,
      name: users.find((user) => user.id === id)?.name ?? id.slice(0, 8),
      amount: (base + (index < remainder ? 1 : 0)) / 100
    }));
  }, [amount, selectedUserIds, users]);

  async function upload(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    if (!file) {
      setError(t("selectFile"));
      return;
    }
    const validation = validateReceiptFile(file, t);
    if (validation) {
      setError(validation);
      return;
    }
    try {
      const uploaded = await uploadMutation.mutateAsync({ file, runOcr: true });
      setReceipt(uploaded);
    } catch {
      setError(errorsT("receiptUploadDescription"));
    }
  }

  async function createExpense(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!receipt) {
      return;
    }
    setError(null);
    const input: CreateExpenseInput = {
      title,
      amount: { amount: Number.parseFloat(amount), currency: expenseCurrency },
      category,
      expenseDate,
      paidByUserId,
      splitType: "selected_equal",
      participantUserIds: selectedUserIds,
      notes: notes.trim() || null,
      linkedAccommodation: false,
      metadata: { receiptOcrReviewed: true }
    };
    try {
      await createMutation.mutateAsync({ receiptId: receipt.id, input });
      onCreated?.();
      onClose();
    } catch (err) {
      setError(getErrorMessage(err, t("createExpenseFailed")));
    }
  }

  function toggleUser(id: string) {
    setSelectedUserIds((current) =>
      current.includes(id) ? current.filter((item) => item !== id) : [...current, id]
    );
  }

  return (
    <Card>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h3 className="text-base font-semibold text-slate-950">{t("uploadReceipt")}</h3>
          <p className="mt-1 text-sm leading-6 text-slate-600">{t("confirmBeforeCreate")}</p>
        </div>
        <Button onClick={onClose} size="sm" type="button" variant="ghost">
          {expensesT("cancel")}
        </Button>
      </div>

      {error ? <InlineError className="mt-4 rounded-lg border border-red-200 bg-red-50 p-3" id="receipt-upload-error" message={error} /> : null}

      {!receipt ? (
        <form className="mt-4 space-y-4" onSubmit={upload}>
          <div className="space-y-2">
            <label className="block text-sm font-medium text-slate-700" htmlFor="receipt-file">
              {t("fileLabel")}
            </label>
            <Input
              accept={RECEIPT_ALLOWED_TYPES.join(",")}
              aria-describedby={`receipt-file-hint${error ? " receipt-upload-error" : ""}`}
              id="receipt-file"
              onChange={(event) => {
                const selected = event.target.files?.[0] ?? null;
                setFile(selected);
                setError(selected ? validateReceiptFile(selected, t) : null);
              }}
              required
              type="file"
            />
            <FieldHint id="receipt-file-hint">
              {t("supportedFiles", {
                maxSize: Math.round(RECEIPT_MAX_FILE_SIZE_BYTES / (1024 * 1024))
              })}
            </FieldHint>
            {file ? <p className="text-sm font-medium text-slate-700">{t("selectedFile", { name: file.name })}</p> : null}
            <p className="rounded-md border border-amber-200 bg-amber-50 p-3 text-xs leading-5 text-amber-900">
              {t("privacyNotice")}
            </p>
          </div>
          <Button disabled={uploadMutation.isPending} type="submit">
            {uploadMutation.isPending ? <ButtonSpinner className="mr-2" /> : null}
            {uploadMutation.isPending ? t("uploading") : t("uploadAndExtract")}
          </Button>
        </form>
      ) : (
        <div className="mt-4 grid gap-4 lg:grid-cols-[minmax(0,22rem)_minmax(0,1fr)]">
          <div className="space-y-3">
            <ReceiptPreview receipt={receipt} />
            <div className="flex flex-wrap items-center gap-2 text-sm text-slate-600">
              <span className="font-medium text-slate-900">{receipt.originalFilename}</span>
              <ReceiptConfidenceBadge confidence={receipt.ocrResult?.confidence} />
            </div>
            <ReceiptWarningsList warnings={receipt.ocrResult?.warnings} />
          </div>

          <form className="space-y-4" onSubmit={createExpense}>
            <div className="grid gap-4 md:grid-cols-2">
              <label className="space-y-1 text-sm font-medium text-slate-700">
                {expensesT("form.title")}
                <Input onChange={(event) => setTitle(event.target.value)} required value={title} />
              </label>
              <label className="space-y-1 text-sm font-medium text-slate-700">
                {expensesT("form.date")}
                <Input onChange={(event) => setExpenseDate(event.target.value)} required type="date" value={expenseDate} />
              </label>
              <label className="space-y-1 text-sm font-medium text-slate-700">
                {expensesT("form.amount")}
                <Input min="0.01" onChange={(event) => setAmount(event.target.value)} required step="0.01" type="number" value={amount} />
              </label>
              <label className="space-y-1 text-sm font-medium text-slate-700">
                {expensesT("form.currency")}
                <Input maxLength={3} onChange={(event) => setExpenseCurrency(event.target.value.toUpperCase())} required value={expenseCurrency} />
              </label>
              <label className="space-y-1 text-sm font-medium text-slate-700">
                {expensesT("form.category")}
                <Select onChange={(event) => setCategory(event.target.value as ExpenseCategory)} value={category}>
                  {EXPENSE_CATEGORIES.map((item) => (
                    <option key={item} value={item}>
                      {expensesT(`categories.${item}`)}
                    </option>
                  ))}
                </Select>
              </label>
              <label className="space-y-1 text-sm font-medium text-slate-700">
                {expensesT("form.paidBy")}
                <Select onChange={(event) => setPaidByUserId(event.target.value)} required value={paidByUserId}>
                  {users.map((user) => (
                    <option key={user.id} value={user.id}>
                      {user.name}
                    </option>
                  ))}
                </Select>
              </label>
            </div>

            <div>
              <p className="text-sm font-medium text-slate-700">{expensesT("form.participants")}</p>
              <div className="mt-2 grid gap-2 sm:grid-cols-2">
                {users.map((user) => (
                  <label className="flex items-center justify-between gap-3 rounded-md border border-slate-200 px-3 py-2 text-sm text-slate-700" key={user.id}>
                    <span>{user.name}</span>
                    <input checked={selectedUserIds.includes(user.id)} onChange={() => toggleUser(user.id)} type="checkbox" />
                  </label>
                ))}
              </div>
            </div>

            <div className="rounded-lg border border-slate-200 bg-slate-50 p-3">
              <p className="text-sm font-medium text-slate-950">{expensesT("splitPreview")}</p>
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
              {expensesT("form.notes")}
              <Textarea onChange={(event) => setNotes(event.target.value)} value={notes} />
            </label>

            <Button disabled={createMutation.isPending || users.length === 0} type="submit">
              {createMutation.isPending ? expensesT("saving") : t("createExpense")}
            </Button>
          </form>
        </div>
      )}
    </Card>
  );
}

export function validateReceiptFile(
  file: File,
  t: (key: "unsupportedFileType" | "fileTooLarge") => string
) {
  if (!RECEIPT_ALLOWED_TYPES.includes(file.type as (typeof RECEIPT_ALLOWED_TYPES)[number])) {
    return t("unsupportedFileType");
  }
  if (file.size > RECEIPT_MAX_FILE_SIZE_BYTES) {
    return t("fileTooLarge");
  }
  return null;
}
