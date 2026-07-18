"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { VerificationDetailRow } from "./VerificationDetailRow";
import { VerificationStatusBadge } from "./VerificationStatusBadge";
import type { RealWorldReadiness } from "@/types/verification";

export function RealWorldReadinessCard({
  readiness,
  sectionId
}: {
  readiness: RealWorldReadiness;
  sectionId?: string;
}) {
  const [message, setMessage] = useState<string | null>(null);
  const t = useTranslations("verification");
  return (
    <section id={sectionId} className="scroll-mt-24 rounded-[18px] border border-[#D9E4DD] bg-white p-5 shadow-sm">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.08em] text-[#5D796C]">{t("eyebrow")}</p>
          <h2 className="mt-1 text-xl font-semibold capitalize text-[#283831]">{t(`level.${readiness.level}`)}</h2>
          <p className="mt-1 text-sm text-[#66716A]">{t("disclaimer")}</p>
        </div>
        <div className="rounded-full bg-[#EEF5F0] px-4 py-2 text-lg font-semibold text-[#35654D]" aria-label={t("score", { score: readiness.score })}>
          {readiness.score}/100
        </div>
      </div>
      <dl className="mt-4 grid grid-cols-3 gap-2 text-center sm:grid-cols-6">
        <Metric label={t("status.verified")} value={readiness.summary.verifiedCount} />
        <Metric label={t("status.needs_review")} value={readiness.summary.needsReviewCount} />
        <Metric label={t("status.estimated")} value={readiness.summary.estimatedCount} />
        <Metric label={t("status.stale")} value={readiness.summary.staleCount} />
        <Metric label={t("status.missing")} value={readiness.summary.missingCount} />
        <Metric label={t("status.unavailable")} value={readiness.summary.unavailableCount} />
      </dl>
      {message ? <p className="mt-4 rounded-lg bg-[#F2F7F1] px-3 py-2 text-sm text-[#38543F]" role="status">{message}</p> : null}
      {readiness.topIssues.length ? (
        <ul className="mt-4">
          {readiness.topIssues.slice(0, 3).map((detail) => <VerificationDetailRow detail={detail} key={`${detail.scope}:${detail.entityId}:${detail.status}`} onActionComplete={setMessage} tripId={readiness.tripId} />)}
        </ul>
      ) : <div className="mt-4 flex items-center gap-2 text-sm text-[#426651]"><VerificationStatusBadge status="verified" /> {t("allCurrent")}</div>}
    </section>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return <div className="rounded-xl bg-[#F8FAF8] px-2 py-2"><dt className="text-[11px] text-[#6A756E]">{label}</dt><dd className="mt-0.5 text-base font-semibold text-[#33423A]">{value}</dd></div>;
}
