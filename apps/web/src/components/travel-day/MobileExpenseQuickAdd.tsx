"use client";

import { FormEvent, useState } from "react";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { useCreateTripExpense } from "@/hooks/useCreateTripExpense";
import type { TravelDaySummary } from "@/types/travel-day";

export function MobileExpenseQuickAdd({ tripId, userId, currency, online }: { tripId: string; userId?: string | null; currency: TravelDaySummary["expenses"]["quickAddDefaults"]["currency"]; online: boolean }) {
  const mutation = useCreateTripExpense(tripId);
  const [open, setOpen] = useState(false); const [title, setTitle] = useState(""); const [amount, setAmount] = useState(""); const [error, setError] = useState<string | null>(null);
  if (!open) return <div id="expense"><Button className="min-h-12" disabled={!online} onClick={() => setOpen(true)} variant="secondary">Add expense</Button></div>;
  function submit(event: FormEvent) { event.preventDefault(); const value = Number(amount); if (!title.trim() || !Number.isFinite(value) || value <= 0 || !userId) { setError("Enter a title and a valid amount."); return; } mutation.mutate({ title: title.trim(), amount: { amount: value, currency }, category: "other", expenseDate: new Date().toISOString().slice(0, 10), paidByUserId: userId, splitType: "payer_only", participantUserIds: [userId], linkedAccommodation: false }, { onSuccess: () => { setOpen(false); setTitle(""); setAmount(""); }, onError: () => setError("Could not save this expense.") }); }
  return <form className="rounded-2xl border border-sand-300 bg-white p-4" onSubmit={submit}><div className="flex items-center justify-between"><h2 className="font-semibold text-cocoa-900">Quick expense</h2><button className="text-sm text-cocoa-500" onClick={() => setOpen(false)} type="button">Cancel</button></div><div className="mt-3 grid grid-cols-[1fr_7rem] gap-2"><Input aria-label="Expense title" onChange={(event) => setTitle(event.target.value)} placeholder="Coffee, ticket…" value={title}/><Input aria-label="Expense amount" min="0.01" onChange={(event) => setAmount(event.target.value)} placeholder={currency} step="0.01" type="number" value={amount}/></div>{error ? <p className="mt-2 text-xs text-red-700">{error}</p> : null}<Button className="mt-3" disabled={mutation.isPending} type="submit">{mutation.isPending ? "Saving…" : "Save expense"}</Button></form>;
}
