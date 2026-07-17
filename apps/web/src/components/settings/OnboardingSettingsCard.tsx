"use client";

import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useAuth } from "@/components/auth/AuthProvider";
import { useDismissedTips } from "@/hooks/useDismissedTips";
import { useOnboardingState } from "@/hooks/useOnboardingState";
import { GhostButton, PrimaryButton, SectionHeading, SettingsCard } from "./controls";

export function OnboardingSettingsCard() {
  const t = useTranslations("settings.onboarding");
  const router = useRouter();
  const { user } = useAuth();
  const onboarding = useOnboardingState(user?.id);
  const tips = useDismissedTips(user?.id);
  const status = onboarding.state.onboardingCompleted
    ? t("completed")
    : onboarding.state.onboardingSkipped
      ? t("skipped")
      : t("inProgress");

  function restart() {
    onboarding.restart();
    router.push("/getting-started");
  }

  return (
    <SettingsCard>
      <SectionHeading title={t("title")} subtitle={t("description")} />
      <div className="mt-5 flex flex-wrap items-center justify-between gap-4 rounded-[14px] bg-sand-50 px-4 py-3.5">
        <div>
          <p className="text-[12px] font-semibold uppercase tracking-[0.08em] text-cocoa-400">{t("status")}</p>
          <p className="mt-1 text-[14px] font-semibold text-cocoa-900">{status}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <GhostButton type="button" onClick={tips.clear} disabled={tips.dismissed.length === 0}>{t("clearTips")}</GhostButton>
          <PrimaryButton type="button" onClick={restart}>{t("restart")}</PrimaryButton>
        </div>
      </div>
    </SettingsCard>
  );
}
