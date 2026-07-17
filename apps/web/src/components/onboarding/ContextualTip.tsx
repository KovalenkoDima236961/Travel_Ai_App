"use client";

import { useTranslations } from "next-intl";
import { useAuth } from "@/components/auth/AuthProvider";
import { useDismissedTips } from "@/hooks/useDismissedTips";
import { ONBOARDING_TIPS, type OnboardingTipId } from "@/lib/onboarding/tips";
import { FeatureHint } from "./FeatureHint";

export function ContextualTip({ tipId }: { tipId: OnboardingTipId }) {
  const t = useTranslations("onboarding.tips");
  const { user } = useAuth();
  const { isDismissed, dismiss } = useDismissedTips(user?.id);

  if (!user?.id || isDismissed(tipId)) {
    return null;
  }

  const key = ONBOARDING_TIPS[tipId];
  return (
    <FeatureHint
      title={t("title")}
      description={t(key)}
      action={
        <button
          type="button"
          onClick={() => dismiss(tipId)}
          aria-label={t("dismiss")}
          className="shrink-0 rounded-full px-2 py-1 text-[12px] font-semibold text-[#58705E] transition hover:bg-white focus:outline-none focus:ring-2 focus:ring-[#6B8B75]/30"
        >
          {t("dismiss")}
        </button>
      }
    />
  );
}
