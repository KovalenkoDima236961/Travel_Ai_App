"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/shared/ui/button";
import type { TripRecapContent } from "@/types/recap";

export function RecapSummaryEditor({ recap, editable, pending, onSave }: { recap: TripRecapContent; editable: boolean; pending: boolean; onSave: (recap: TripRecapContent) => void }) {
  const t = useTranslations("recap");
  const [title, setTitle] = useState(recap.title);
  const [summary, setSummary] = useState(recap.summary);
  const [notes, setNotes] = useState(recap.userEditableNotes);
  useEffect(() => { setTitle(recap.title); setSummary(recap.summary); setNotes(recap.userEditableNotes); }, [recap]);
  const changed = title !== recap.title || summary !== recap.summary || notes !== recap.userEditableNotes;
  return <section className="rounded-3xl border border-sand-300 bg-white p-6 shadow-sm"><label className="block text-xs font-semibold uppercase tracking-wide text-cocoa-500" htmlFor="recap-title">{t("title")}</label><input className="mt-2 w-full rounded-xl border border-sand-300 px-3 py-2 text-xl font-semibold text-cocoa-900" disabled={!editable} id="recap-title" onChange={(event) => setTitle(event.target.value)} value={title}/><label className="mt-5 block text-xs font-semibold uppercase tracking-wide text-cocoa-500" htmlFor="recap-summary">{t("summary")}</label><textarea className="mt-2 min-h-28 w-full rounded-xl border border-sand-300 px-3 py-2 text-cocoa-800" disabled={!editable} id="recap-summary" onChange={(event) => setSummary(event.target.value)} value={summary}/><label className="mt-5 block text-xs font-semibold uppercase tracking-wide text-cocoa-500" htmlFor="recap-notes">{t("notes")}</label><textarea className="mt-2 min-h-24 w-full rounded-xl border border-sand-300 px-3 py-2 text-cocoa-800" disabled={!editable} id="recap-notes" onChange={(event) => setNotes(event.target.value)} value={notes}/>{editable && changed ? <Button className="mt-4" disabled={pending} onClick={() => onSave({ ...recap, title: title.trim(), summary: summary.trim(), userEditableNotes: notes.trim() })}>{pending ? t("saving") : t("save")}</Button> : null}</section>;
}
