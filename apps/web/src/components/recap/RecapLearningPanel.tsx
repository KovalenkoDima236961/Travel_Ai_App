"use client";

import { useTranslations } from "next-intl";
import { Button } from "@/shared/ui/button";
import type { LearningCandidate } from "@/types/recap";

export function RecapLearningPanel({ candidates, allowed, pending, onApply }: { candidates: LearningCandidate[]; allowed: boolean; pending: boolean; onApply: (candidate: LearningCandidate) => void }) {
  const t = useTranslations("recap");
  if (!candidates.length) return null;
  return <section className="rounded-3xl border border-sand-300 bg-white p-6"><h2 className="font-newsreader text-2xl text-cocoa-900">{t("preferences")}</h2><p className="mt-2 text-sm text-cocoa-600">{t("preferencesDescription")}</p><div className="mt-4 space-y-3">{candidates.map((candidate, index) => <div className="flex flex-wrap items-center justify-between gap-3 rounded-2xl bg-sand-100 p-3" key={`${candidate.feedbackType}-${candidate.label}-${index}`}><div><p className="font-medium text-cocoa-900">{candidate.label}</p>{candidate.value ? <p className="text-sm text-cocoa-600">{candidate.value}</p> : null}</div>{allowed ? <Button disabled={pending} onClick={() => onApply({ ...candidate, approved: true })} size="sm" variant="secondary">{pending ? t("applying") : t("apply")}</Button> : null}</div>)}</div></section>;
}
