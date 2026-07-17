import { useTranslations } from "next-intl";

export function CopilotDisclaimer() {
  const t = useTranslations("copilot");
  return <p className="px-4 pb-3 text-xs leading-5 text-cocoa-500">{t("disclaimer")}</p>;
}
