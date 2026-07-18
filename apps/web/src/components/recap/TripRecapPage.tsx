"use client";

import Link from "next/link";
import { useState } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/shared/ui/button";
import { getErrorMessage } from "@/lib/utils";
import { useTripRecap, useTripRecapMutations, useTripRecapStatus } from "@/hooks/useTripRecap";
import { RecapLearningPanel } from "./RecapLearningPanel";
import { RecapPrivacyNotice } from "./RecapPrivacyNotice";
import { RecapSections } from "./RecapSections";
import { RecapStatusEmptyState } from "./RecapStatusEmptyState";
import { RecapSummaryEditor } from "./RecapSummaryEditor";
import { RecapTemplateDialog } from "./RecapTemplateDialog";

export function TripRecapPage({ tripId }: { tripId: string }) {
  const t = useTranslations("recap");
  const [templateOpen, setTemplateOpen] = useState(false);
  const [message, setMessage] = useState<string>();
  const status = useTripRecapStatus(tripId);
  const recapQuery = useTripRecap(tripId, Boolean(status.data?.hasRecap));
  const mutations = useTripRecapMutations(tripId);
  const recap = recapQuery.data?.recap;
  const permissions = recapQuery.data?.permissions;
  const run = (work: Promise<unknown>, success: string) => { setMessage(undefined); void work.then(() => setMessage(success)).catch((error: unknown) => setMessage(getErrorMessage(error))); };

  if (status.isLoading) return <main className="mx-auto max-w-5xl px-4 py-10 text-cocoa-600">{t("loading")}</main>;
  if (!status.data) return <main className="mx-auto max-w-5xl px-4 py-10"><p className="text-red-700">{t("unavailable")}</p></main>;
  if (!status.data.hasRecap) return <main className="mx-auto max-w-5xl space-y-4 px-4 py-10"><Link className="text-sm font-medium text-clay underline underline-offset-4" href={`/trips/${tripId}`}>{t("backToTrip")}</Link><RecapStatusEmptyState onGenerate={() => run(mutations.generate.mutateAsync(false), t("generated"))} pending={mutations.generate.isPending} status={status.data}/><RecapPrivacyNotice/>{message ? <p className="text-sm text-cocoa-600">{message}</p> : null}</main>;
  if (!recap || !permissions) return <main className="mx-auto max-w-5xl px-4 py-10 text-cocoa-600">{t("loading")}</main>;

  return <main className="mx-auto max-w-5xl space-y-5 px-4 py-10"><div className="flex flex-wrap items-center justify-between gap-3"><Link className="text-sm font-medium text-clay underline underline-offset-4" href={`/trips/${tripId}`}>{t("backToTrip")}</Link><span className="rounded-full bg-sand-200 px-3 py-1 text-sm font-medium text-cocoa-700">{t(`status.${recap.status}`)}</span></div>{message ? <p className="rounded-xl bg-sand-100 px-3 py-2 text-sm text-cocoa-700">{message}</p> : null}<RecapPrivacyNotice/><RecapSummaryEditor editable={permissions.canEdit} onSave={(content) => run(mutations.update.mutateAsync(content), t("saved"))} pending={mutations.update.isPending} recap={recap.recap}/><RecapSections recap={recap.recap}/><RecapLearningPanel allowed={permissions.canApplyLearning} candidates={recap.recap.futurePreferences} onApply={(candidate) => run(mutations.applyLearning.mutateAsync([candidate]), t("applied"))} pending={mutations.applyLearning.isPending}/><section className="flex flex-wrap gap-3"><Button disabled={!permissions.canFinalize || mutations.finalize.isPending} onClick={() => run(mutations.finalize.mutateAsync(), t("finalized"))}>{mutations.finalize.isPending ? t("finalizing") : t("finalize")}</Button>{permissions.canCreateTemplate ? <Button onClick={() => setTemplateOpen(true)} variant="secondary">{t("createTemplate")}</Button> : null}{permissions.canEdit ? <Button disabled={mutations.generate.isPending} onClick={() => run(mutations.generate.mutateAsync(true), t("generated"))} variant="ghost">{t("regenerate")}</Button> : null}</section>{templateOpen ? <RecapTemplateDialog onClose={() => setTemplateOpen(false)} onCreate={(input) => run(mutations.createTemplate.mutateAsync(input).then(() => setTemplateOpen(false)), t("templateCreated"))} pending={mutations.createTemplate.isPending} suggestedTitle={recap.recap.templateSuggestion.title}/> : null}</main>;
}
