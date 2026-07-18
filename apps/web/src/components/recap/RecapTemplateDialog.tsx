"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/shared/ui/button";

export function RecapTemplateDialog({ suggestedTitle, onClose, onCreate, pending }: { suggestedTitle?: string; onClose: () => void; onCreate: (input: { title: string; description?: string; visibility: "private" | "workspace"; tags?: string[]; useRecapLessons: boolean }) => void; pending: boolean }) {
  const t = useTranslations("recap");
  const [title, setTitle] = useState(suggestedTitle || "");
  const [description, setDescription] = useState("");
  return <div aria-modal="true" className="fixed inset-0 z-50 flex items-center justify-center bg-cocoa-900/40 p-4" role="dialog"><form className="w-full max-w-lg rounded-3xl bg-white p-6 shadow-xl" onSubmit={(event) => { event.preventDefault(); onCreate({ title: title.trim(), description: description.trim() || undefined, visibility: "private", useRecapLessons: true }); }}><h2 className="font-newsreader text-2xl text-cocoa-900">{t("templateTitle")}</h2><p className="mt-2 text-sm text-cocoa-600">{t("templateDescription")}</p><label className="mt-4 block text-sm font-medium text-cocoa-700" htmlFor="recap-template-title">{t("title")}</label><input className="mt-1 w-full rounded-xl border border-sand-300 px-3 py-2" id="recap-template-title" onChange={(event) => setTitle(event.target.value)} required value={title}/><label className="mt-4 block text-sm font-medium text-cocoa-700" htmlFor="recap-template-description">{t("description")}</label><textarea className="mt-1 min-h-20 w-full rounded-xl border border-sand-300 px-3 py-2" id="recap-template-description" onChange={(event) => setDescription(event.target.value)} value={description}/><div className="mt-5 flex justify-end gap-3"><Button onClick={onClose} type="button" variant="ghost">{t("cancel")}</Button><Button disabled={pending || !title.trim()} type="submit">{pending ? t("creating") : t("createTemplate")}</Button></div></form></div>;
}
