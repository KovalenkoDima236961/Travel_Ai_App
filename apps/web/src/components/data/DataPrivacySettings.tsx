"use client";

import { AccountCleanupPanel } from "@/components/data/AccountCleanupPanel";
import { AccountExportPanel } from "@/components/data/AccountExportPanel";
import { NotificationCleanupPanel } from "@/components/data/NotificationCleanupPanel";
import { OfflineDataCleanupPanel } from "@/components/data/OfflineDataCleanupPanel";
import { SettingsCard } from "@/components/settings/controls";
import { useTranslations } from "next-intl";

export function DataPrivacySettings() {
  const t = useTranslations("dataPrivacy");
  return (
    <div className="scroll-mt-24" id="data-privacy">
    <SettingsCard>
      <div className="mb-6"><p className="text-xs font-semibold uppercase tracking-[0.16em] text-clay-deep">{t("eyebrow")}</p><h2 className="mt-2 font-newsreader text-2xl font-semibold text-cocoa-900">{t("title")}</h2><p className="mt-2 text-sm leading-6 text-cocoa-500">{t("description")}</p></div>
      <AccountExportPanel />
      <OfflineDataCleanupPanel />
      <NotificationCleanupPanel />
      <AccountCleanupPanel />
    </SettingsCard>
    </div>
  );
}
