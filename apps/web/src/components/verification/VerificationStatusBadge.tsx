import { cn } from "@/lib/utils";
import { useTranslations } from "next-intl";
import type { VerificationStatus } from "@/types/verification";

const tones: Record<VerificationStatus, string> = {
  verified: "bg-emerald-50 text-emerald-700 ring-emerald-200",
  needs_review: "bg-amber-50 text-amber-800 ring-amber-200",
  estimated: "bg-sky-50 text-sky-700 ring-sky-200",
  stale: "bg-orange-50 text-orange-800 ring-orange-200",
  missing: "bg-rose-50 text-rose-700 ring-rose-200",
  unavailable: "bg-red-50 text-red-700 ring-red-200",
  failed: "bg-red-50 text-red-700 ring-red-200",
  not_applicable: "bg-slate-100 text-slate-600 ring-slate-200"
};

export function VerificationStatusBadge({ status }: { status: VerificationStatus }) {
  const t = useTranslations("verification.status");
  return (
    <span className={cn("inline-flex rounded-full px-2 py-0.5 text-xs font-semibold ring-1 ring-inset", tones[status])}>
      {t(status)}
    </span>
  );
}
