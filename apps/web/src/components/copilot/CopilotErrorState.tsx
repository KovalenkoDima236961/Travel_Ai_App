import Link from "next/link";
import { useTranslations } from "next-intl";
import { Button, buttonStyles } from "@/shared/ui/button";

export function CopilotErrorState({ tripId, onRetry }: { tripId: string; onRetry: () => void }) {
  const t = useTranslations("copilot");
  return (
    <div className="rounded-xl border border-[#E5C3B6] bg-[#FBF0EB] p-3 text-sm text-cocoa-700" role="alert">
      <p className="font-semibold">{t("unavailable")}</p>
      <p className="mt-1 text-xs leading-5">{t("unavailableDescription")}</p>
      <div className="mt-3 flex gap-2">
        <Link className={buttonStyles({ variant: "secondary", size: "sm" })} href={`/trips/${tripId}?tab=health`}>
          {t("openHealth")}
        </Link>
        <Button onClick={onRetry} size="sm" variant="secondary">{t("retry")}</Button>
      </div>
    </div>
  );
}
