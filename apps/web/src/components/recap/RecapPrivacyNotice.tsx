"use client";

import { useTranslations } from "next-intl";

export function RecapPrivacyNotice() {
  const t = useTranslations("recap");
  return <p className="rounded-2xl border border-sand-300 bg-sand-100/60 px-4 py-3 text-sm text-cocoa-600">{t("privacy")}</p>;
}
