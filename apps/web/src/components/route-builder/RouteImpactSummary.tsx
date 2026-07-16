import { useTranslations } from "next-intl";
import type { RouteImpact } from "@/lib/route-builder/route-draft";

export function RouteImpactSummary({ impact }: { impact: RouteImpact }) {
  const t = useTranslations("route");
  const items = [
    impact.removedTransportOptionCount > 0
      ? t("impactRemoveTransport", { count: impact.removedTransportOptionCount })
      : null,
    impact.staleTransportOptionCount > 0
      ? t("impactStaleTransport", { count: impact.staleTransportOptionCount })
      : null,
    impact.itineraryImpact ? t("impactItinerary") : null,
    impact.budgetImpact ? t("impactBudget") : null,
    impact.reminderImpact ? t("impactReminders") : null,
    impact.approvalMayReset ? t("impactApproval") : null,
    impact.stopOrderChanged || impact.legCountChanged ? t("impactHealth") : null
  ].filter((item): item is string => Boolean(item));

  return (
    <div className="rounded-[14px] border border-amber-300 bg-amber-50 p-4">
      <p className="text-[13px] font-semibold text-amber-950">{t("thisChangeWill")}</p>
      <ul className="mt-2 space-y-1.5 text-[13px] leading-5 text-amber-900">
        {items.map((item) => <li key={item}>• {item}</li>)}
      </ul>
    </div>
  );
}
