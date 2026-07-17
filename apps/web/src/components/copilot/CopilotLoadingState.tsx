import { useTranslations } from "next-intl";

export function CopilotLoadingState() {
  const t = useTranslations("copilot");
  return (
    <div aria-live="polite" className="mr-12 flex items-center gap-2 rounded-2xl rounded-bl-md border border-sand-300 bg-sand-50 px-3.5 py-3 text-sm text-cocoa-500">
      <span className="h-2 w-2 animate-pulse rounded-full bg-clay" />
      {t("checking")}
    </div>
  );
}
