"use client";

import { useTranslations } from "next-intl";
import { ErrorState } from "@/components/ui";

export function PublicShareUnavailableState({
  expired = false,
  onRetry,
  retrying = false
}: {
  expired?: boolean;
  onRetry?: () => void;
  retrying?: boolean;
}) {
  const t = useTranslations("publicShare");
  return (
    <div className="mx-auto max-w-[600px] px-6 py-16 sm:py-24">
      <ErrorState
        className="rounded-[18px]"
        description={expired ? t("expiredDescription") : t("unavailableDescription")}
        retryAction={onRetry ? { onRetry, pending: retrying } : undefined}
        secondaryAction={{ href: "/", label: t("goHome") }}
        title={expired ? t("expiredTitle") : t("unavailableTitle")}
      />
    </div>
  );
}
