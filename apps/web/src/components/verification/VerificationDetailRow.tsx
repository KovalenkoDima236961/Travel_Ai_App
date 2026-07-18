import { VerificationStatusBadge } from "./VerificationStatusBadge";
import { VerifyNowButton } from "./VerifyNowButton";
import { useTranslations } from "next-intl";
import type { VerificationDetail } from "@/types/verification";

export function VerificationDetailRow({ detail, tripId, onActionComplete }: { detail: VerificationDetail; tripId: string; onActionComplete?: (message: string) => void }) {
  const t = useTranslations("verification");
  return (
    <li className="flex flex-col gap-3 border-t border-[#E8E6DF] py-3 first:border-t-0 first:pt-0 sm:flex-row sm:items-start sm:justify-between">
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <p className="font-medium text-[#2C3B35]">{detail.title}</p>
          <VerificationStatusBadge status={detail.status} />
          {detail.provider ? <span className="text-xs text-[#6F766F]">{detail.provider}</span> : null}
        </div>
        <p className="mt-1 text-sm leading-5 text-[#626B65]">{detail.message}</p>
        {detail.checkedAt ? <p className="mt-1 text-xs text-[#7B817C]">{t("lastChecked", { value: new Date(detail.checkedAt).toLocaleString() })}</p> : null}
      </div>
      {detail.action ? <VerifyNowButton action={detail.action} entityId={detail.entityId} entityType={detail.entityType} onComplete={onActionComplete} scope={detail.scope} tripId={tripId} /> : null}
    </li>
  );
}
